use std::collections::HashMap;
use std::sync::Arc;

use axum::extract::{Path, State};
use axum::http::{HeaderMap, StatusCode};
use axum::response::{IntoResponse, Response};
use axum::routing::{get, post};
use axum::{Json, Router};
use serde::Deserialize;
use serde_json::json;
use tokio::sync::Semaphore;

use crate::server::{QuickDeskMcpServer, SharedState};
use crate::ws_client::WsClient;

#[derive(Clone)]
struct TriggerState {
    token: Option<String>,
    ws: WsClient,
    shared_state: Arc<SharedState>,
    allowed_devices: Vec<String>,
    run_slots: Arc<Semaphore>,
}

#[derive(Deserialize)]
struct TriggerRunRequest {
    device_id: String,
    overrides: Option<HashMap<String, serde_json::Value>>,
    stop_on_failure: Option<bool>,
}

pub fn router(
    token: Option<String>,
    ws: WsClient,
    shared_state: Arc<SharedState>,
    allowed_devices: Vec<String>,
) -> Router {
    router_with_slots(
        token,
        ws,
        shared_state,
        allowed_devices,
        Arc::new(Semaphore::new(1)),
    )
}

fn router_with_slots(
    token: Option<String>,
    ws: WsClient,
    shared_state: Arc<SharedState>,
    allowed_devices: Vec<String>,
    run_slots: Arc<Semaphore>,
) -> Router {
    let state = TriggerState {
        token,
        ws,
        shared_state,
        allowed_devices,
        run_slots,
    };
    Router::new()
        .route("/v1/workflows/{workflow_id}/run", post(run_workflow))
        .route("/v1/workflow-runs/{run_id}", get(get_workflow_run))
        .with_state(state)
}

fn error_response(status: StatusCode, message: &str) -> Response {
    (status, Json(json!({"error": message}))).into_response()
}

enum TriggerAuthError {
    Disabled,
    Unauthorized,
}

fn auth_error_response(error: TriggerAuthError) -> Response {
    match error {
        TriggerAuthError::Disabled => error_response(StatusCode::NOT_FOUND, "not found"),
        TriggerAuthError::Unauthorized => error_response(StatusCode::UNAUTHORIZED, "unauthorized"),
    }
}

fn authorize_request(state: &TriggerState, headers: &HeaderMap) -> Result<(), TriggerAuthError> {
    if state.token.as_deref().is_none_or(str::is_empty) {
        return Err(TriggerAuthError::Disabled);
    }
    let authorization = headers
        .get(axum::http::header::AUTHORIZATION)
        .and_then(|value| value.to_str().ok());
    if !authorize_bearer(state.token.as_deref(), authorization) {
        return Err(TriggerAuthError::Unauthorized);
    }
    Ok(())
}

async fn run_workflow(
    State(state): State<TriggerState>,
    Path(workflow_id): Path<String>,
    headers: HeaderMap,
    Json(request): Json<TriggerRunRequest>,
) -> Response {
    if let Err(error) = authorize_request(&state, &headers) {
        return auth_error_response(error);
    }
    let Ok(_permit) = state.run_slots.clone().try_acquire_owned() else {
        return error_response(
            StatusCode::TOO_MANY_REQUESTS,
            "another business workflow is already running",
        );
    };
    let device_id = request.device_id.trim();
    if device_id.is_empty() {
        return error_response(StatusCode::BAD_REQUEST, "device_id is required");
    }

    let server = QuickDeskMcpServer::new(
        state.ws.clone(),
        state.allowed_devices.clone(),
        state.shared_state.clone(),
    );
    match server
        .execute_workflow_replay(
            &workflow_id,
            device_id,
            request.overrides.unwrap_or_default(),
            request.stop_on_failure.unwrap_or(true),
        )
        .await
    {
        Ok(report) => Json(report).into_response(),
        Err(error) if error == "workflow not found" => {
            error_response(StatusCode::NOT_FOUND, &error)
        }
        Err(error) if error.starts_with("Device '") => {
            error_response(StatusCode::FORBIDDEN, &error)
        }
        Err(error) => error_response(StatusCode::INTERNAL_SERVER_ERROR, &error),
    }
}

async fn get_workflow_run(
    State(state): State<TriggerState>,
    Path(run_id): Path<String>,
    headers: HeaderMap,
) -> Response {
    if let Err(error) = authorize_request(&state, &headers) {
        return auth_error_response(error);
    }
    match state.shared_state.workflow.get_run_report(&run_id) {
        Some(report) => Json(report).into_response(),
        None => error_response(StatusCode::NOT_FOUND, "workflow run not found"),
    }
}

fn constant_time_eq(left: &str, right: &str) -> bool {
    let left = left.as_bytes();
    let right = right.as_bytes();
    if left.len() != right.len() {
        return false;
    }
    left.iter()
        .zip(right)
        .fold(0u8, |diff, (a, b)| diff | (a ^ b))
        == 0
}

fn authorize_bearer(configured_token: Option<&str>, authorization: Option<&str>) -> bool {
    let Some(configured) = configured_token.filter(|token| !token.is_empty()) else {
        return false;
    };
    let Some(value) = authorization else {
        return false;
    };
    let Some(provided) = value.strip_prefix("Bearer ") else {
        return false;
    };
    constant_time_eq(configured, provided)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::event_bus::EventBus;
    use crate::memory::MemoryStore;
    use crate::server::SharedState;
    use crate::trust::TrustEngine;
    use crate::workflow::{StepExecutor, WorkflowRunReport, WorkflowStore};
    use crate::ws_client::WsClient;
    use axum::body::{to_bytes, Body};
    use axum::http::{Request, StatusCode};
    use futures_util::{SinkExt, StreamExt};
    use serde_json::json;
    use std::sync::Arc;
    use tokio_tungstenite::{accept_async, tungstenite::Message};
    use tower::ServiceExt;
    use uuid::Uuid;

    #[test]
    fn bearer_auth_requires_the_exact_configured_trigger_token() {
        let token = "business-secret";

        assert!(authorize_bearer(
            Some(token),
            Some("Bearer business-secret")
        ));
        assert!(!authorize_bearer(Some(token), Some("Bearer wrong-secret")));
        assert!(!authorize_bearer(
            Some(token),
            Some("Basic business-secret")
        ));
        assert!(!authorize_bearer(Some(token), None));
        assert!(!authorize_bearer(None, Some("Bearer business-secret")));
    }

    #[tokio::test]
    async fn business_trigger_executes_workflow_and_exposes_persisted_report() {
        let temp = std::env::temp_dir().join(format!("quickdesk-trigger-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&temp).unwrap();

        let workflows = WorkflowStore::with_storage_dir(temp.join("workflows"));
        let workflow_id = workflows
            .start_recording("business workflow", "device-1")
            .unwrap();
        workflows.record_step(
            "device-1",
            "list_processes",
            &json!({}),
            "processes",
            true,
            StepExecutor::SkillHost,
        );
        workflows.stop_recording("device-1", "", &[]).unwrap();

        let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
        let address = listener.local_addr().unwrap();
        let fake_quickdesk = tokio::spawn(async move {
            let (stream, _) = listener.accept().await.unwrap();
            let mut socket = accept_async(stream).await.unwrap();
            while let Some(Ok(Message::Text(text))) = socket.next().await {
                let request: serde_json::Value = serde_json::from_str(&text).unwrap();
                if let Some(id) = request.get("id").and_then(serde_json::Value::as_str) {
                    let response = json!({"id": id, "result": {"processes": ["fake"]}});
                    socket
                        .send(Message::Text(response.to_string().into()))
                        .await
                        .unwrap();
                }
            }
        });
        let ws = WsClient::connect(&format!("ws://{address}"), None, EventBus::new(100))
            .await
            .unwrap();

        let shared = Arc::new(SharedState {
            memory: MemoryStore::with_db_path(temp.join("memory.db")).unwrap(),
            workflow: workflows,
            trust: TrustEngine::with_db_path(temp.join("trust.db")).unwrap(),
        });
        let app = router(Some("business-secret".to_string()), ws, shared, Vec::new());

        let request = Request::builder()
            .method("POST")
            .uri(format!("/v1/workflows/{workflow_id}/run"))
            .header("Authorization", "Bearer business-secret")
            .header("Content-Type", "application/json")
            .body(Body::from(json!({"device_id": "device-1"}).to_string()))
            .unwrap();
        let response = app.clone().oneshot(request).await.unwrap();

        assert_eq!(response.status(), StatusCode::OK);
        let body = to_bytes(response.into_body(), 1024 * 1024).await.unwrap();
        let report: WorkflowRunReport = serde_json::from_slice(&body).unwrap();
        assert!(report.success);
        assert_eq!(report.succeeded_steps, 1);

        let request = Request::builder()
            .uri(format!("/v1/workflow-runs/{}", report.run_id))
            .header("Authorization", "Bearer business-secret")
            .body(Body::empty())
            .unwrap();
        let response = app.oneshot(request).await.unwrap();
        assert_eq!(response.status(), StatusCode::OK);
        let body = to_bytes(response.into_body(), 1024 * 1024).await.unwrap();
        let persisted: WorkflowRunReport = serde_json::from_slice(&body).unwrap();
        assert_eq!(persisted, report);

        fake_quickdesk.abort();
        let _ = std::fs::remove_dir_all(temp);
    }

    #[tokio::test]
    async fn business_trigger_rejects_a_bad_token_and_disallowed_device() {
        let temp = std::env::temp_dir().join(format!("quickdesk-trigger-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&temp).unwrap();
        let shared = Arc::new(SharedState {
            memory: MemoryStore::with_db_path(temp.join("memory.db")).unwrap(),
            workflow: WorkflowStore::with_storage_dir(temp.join("workflows")),
            trust: TrustEngine::with_db_path(temp.join("trust.db")).unwrap(),
        });
        let app = router(
            Some("business-secret".to_string()),
            WsClient::disconnected(),
            shared,
            vec!["allowed-device".to_string()],
        );
        let workflow_id = Uuid::new_v4();

        let request = Request::builder()
            .method("POST")
            .uri(format!("/v1/workflows/{workflow_id}/run"))
            .header("Authorization", "Bearer wrong-secret")
            .header("Content-Type", "application/json")
            .body(Body::from(
                json!({"device_id": "blocked-device"}).to_string(),
            ))
            .unwrap();
        let response = app.clone().oneshot(request).await.unwrap();
        assert_eq!(response.status(), StatusCode::UNAUTHORIZED);

        let request = Request::builder()
            .method("POST")
            .uri(format!("/v1/workflows/{workflow_id}/run"))
            .header("Authorization", "Bearer business-secret")
            .header("Content-Type", "application/json")
            .body(Body::from(
                json!({"device_id": "blocked-device"}).to_string(),
            ))
            .unwrap();
        let response = app.oneshot(request).await.unwrap();
        assert_eq!(response.status(), StatusCode::FORBIDDEN);

        let _ = std::fs::remove_dir_all(temp);
    }

    #[tokio::test]
    async fn business_trigger_returns_busy_when_a_workflow_run_is_active() {
        let temp = std::env::temp_dir().join(format!("quickdesk-trigger-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&temp).unwrap();
        let shared = Arc::new(SharedState {
            memory: MemoryStore::with_db_path(temp.join("memory.db")).unwrap(),
            workflow: WorkflowStore::with_storage_dir(temp.join("workflows")),
            trust: TrustEngine::with_db_path(temp.join("trust.db")).unwrap(),
        });
        let slots = Arc::new(tokio::sync::Semaphore::new(1));
        let _held = slots.clone().try_acquire_owned().unwrap();
        let app = router_with_slots(
            Some("business-secret".to_string()),
            WsClient::disconnected(),
            shared,
            Vec::new(),
            slots,
        );

        let request = Request::builder()
            .method("POST")
            .uri(format!("/v1/workflows/{}/run", Uuid::new_v4()))
            .header("Authorization", "Bearer business-secret")
            .header("Content-Type", "application/json")
            .body(Body::from(json!({"device_id": "device-1"}).to_string()))
            .unwrap();
        let response = app.oneshot(request).await.unwrap();

        assert_eq!(response.status(), StatusCode::TOO_MANY_REQUESTS);
        let _ = std::fs::remove_dir_all(temp);
    }
}
