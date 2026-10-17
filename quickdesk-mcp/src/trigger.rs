use std::collections::HashMap;
use std::sync::Arc;
use std::time::Duration;

use axum::extract::{Path, State};
use axum::http::{HeaderMap, StatusCode};
use axum::response::{IntoResponse, Response};
use axum::routing::{get, post};
use axum::{Json, Router};
use chrono::Utc;
use serde::Deserialize;
use serde_json::json;
use tokio::sync::Semaphore;
use tokio_util::sync::CancellationToken;

use crate::idempotency::{IdempotencyClaim, WorkflowIdempotencyStore};
use crate::server::{QuickDeskMcpServer, SharedState};
use crate::workflow_trigger::{WorkflowTrigger, WorkflowTriggerSpec, WorkflowTriggerStore};
use crate::ws_client::WsClient;

#[derive(Clone)]
struct TriggerState {
    token: Option<String>,
    ws: WsClient,
    shared_state: Arc<SharedState>,
    allowed_devices: Vec<String>,
    run_slots: Arc<Semaphore>,
    triggers: WorkflowTriggerStore,
    idempotency: WorkflowIdempotencyStore,
}

#[derive(Deserialize)]
struct TriggerRunRequest {
    device_id: String,
    overrides: Option<HashMap<String, serde_json::Value>>,
    stop_on_failure: Option<bool>,
}

#[derive(Deserialize)]
struct CreateTriggerRequest {
    name: String,
    workflow_id: String,
    device_id: String,
    overrides: Option<HashMap<String, serde_json::Value>>,
    stop_on_failure: Option<bool>,
    interval_seconds: Option<u64>,
}

pub fn router(
    token: Option<String>,
    ws: WsClient,
    shared_state: Arc<SharedState>,
    allowed_devices: Vec<String>,
    triggers: WorkflowTriggerStore,
    idempotency: WorkflowIdempotencyStore,
    run_slots: Arc<Semaphore>,
) -> Router {
    router_with_slots(
        token,
        ws,
        shared_state,
        allowed_devices,
        triggers,
        idempotency,
        run_slots,
    )
}

fn router_with_slots(
    token: Option<String>,
    ws: WsClient,
    shared_state: Arc<SharedState>,
    allowed_devices: Vec<String>,
    triggers: WorkflowTriggerStore,
    idempotency: WorkflowIdempotencyStore,
    run_slots: Arc<Semaphore>,
) -> Router {
    let state = TriggerState {
        token,
        ws,
        shared_state,
        allowed_devices,
        run_slots,
        triggers,
        idempotency,
    };
    Router::new()
        .route("/v1/workflows/{workflow_id}/run", post(run_workflow))
        .route("/v1/workflow-runs/{run_id}", get(get_workflow_run))
        .route("/v1/triggers", get(list_triggers).post(create_trigger))
        .route(
            "/v1/triggers/{trigger_id}",
            axum::routing::delete(delete_trigger),
        )
        .route("/v1/triggers/{trigger_id}/run", post(run_saved_trigger))
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
    let idempotency_key = match read_idempotency_key(&headers) {
        Ok(key) => key,
        Err(error) => return error_response(StatusCode::BAD_REQUEST, &error),
    };
    let device_id = request.device_id.trim();
    if device_id.is_empty() {
        return error_response(StatusCode::BAD_REQUEST, "device_id is required");
    }
    let overrides = request.overrides.unwrap_or_default();
    let stop_on_failure = request.stop_on_failure.unwrap_or(true);
    let idempotency_request = json!({
        "overrides": overrides.clone(),
        "stop_on_failure": stop_on_failure,
    });
    let Ok(_permit) = state.run_slots.clone().try_acquire_owned() else {
        return error_response(
            StatusCode::TOO_MANY_REQUESTS,
            "another business workflow is already running",
        );
    };
    if let Some(key) = idempotency_key.as_deref() {
        match state
            .idempotency
            .claim(key, &workflow_id, device_id, &idempotency_request)
        {
            Ok(IdempotencyClaim::Execute) => {}
            Ok(IdempotencyClaim::Replay {
                run_id,
                error,
                status_code,
            }) => return replay_idempotent_response(&state, run_id, error, status_code),
            Ok(IdempotencyClaim::InProgress) => {
                return error_response(
                    StatusCode::CONFLICT,
                    "a request with this Idempotency-Key is already running",
                )
            }
            Ok(IdempotencyClaim::Conflict) => {
                return error_response(
                    StatusCode::CONFLICT,
                    "Idempotency-Key was already used with a different request",
                )
            }
            Err(error) => return error_response(StatusCode::INTERNAL_SERVER_ERROR, &error),
        }
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
            overrides,
            stop_on_failure,
        )
        .await
    {
        Ok(report) => {
            if let Some(key) = idempotency_key.as_deref() {
                if let Err(error) = state.idempotency.complete(key, &report.run_id) {
                    return error_response(StatusCode::INTERNAL_SERVER_ERROR, &error);
                }
            }
            Json(report).into_response()
        }
        Err(error) => {
            let status = workflow_error_status(&error);
            if let Some(key) = idempotency_key.as_deref() {
                if let Err(store_error) =
                    state.idempotency.fail(key, status.as_u16(), &error)
                {
                    return error_response(StatusCode::INTERNAL_SERVER_ERROR, &store_error);
                }
            }
            error_response(status, &error)
        }
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

async fn list_triggers(State(state): State<TriggerState>, headers: HeaderMap) -> Response {
    if let Err(error) = authorize_request(&state, &headers) {
        return auth_error_response(error);
    }
    Json(state.triggers.list()).into_response()
}

async fn create_trigger(
    State(state): State<TriggerState>,
    headers: HeaderMap,
    Json(request): Json<CreateTriggerRequest>,
) -> Response {
    if let Err(error) = authorize_request(&state, &headers) {
        return auth_error_response(error);
    }
    if state
        .shared_state
        .workflow
        .get_workflow(&request.workflow_id)
        .is_none()
    {
        return error_response(StatusCode::NOT_FOUND, "workflow not found");
    }
    let trigger = match state.triggers.create(WorkflowTriggerSpec {
        name: request.name,
        workflow_id: request.workflow_id,
        device_id: request.device_id,
        overrides: request.overrides.unwrap_or_default(),
        stop_on_failure: request.stop_on_failure.unwrap_or(true),
        interval_seconds: request.interval_seconds,
    }) {
        Ok(trigger) => trigger,
        Err(error) => return error_response(StatusCode::BAD_REQUEST, &error),
    };
    (StatusCode::CREATED, Json(trigger)).into_response()
}

async fn delete_trigger(
    State(state): State<TriggerState>,
    Path(trigger_id): Path<String>,
    headers: HeaderMap,
) -> Response {
    if let Err(error) = authorize_request(&state, &headers) {
        return auth_error_response(error);
    }
    if state.triggers.delete(&trigger_id) {
        StatusCode::NO_CONTENT.into_response()
    } else {
        error_response(StatusCode::NOT_FOUND, "workflow trigger not found")
    }
}

async fn run_saved_trigger(
    State(state): State<TriggerState>,
    Path(trigger_id): Path<String>,
    headers: HeaderMap,
) -> Response {
    if let Err(error) = authorize_request(&state, &headers) {
        return auth_error_response(error);
    }
    let Some(trigger) = state.triggers.get(&trigger_id) else {
        return error_response(StatusCode::NOT_FOUND, "workflow trigger not found");
    };
    let Ok(_permit) = state.run_slots.clone().try_acquire_owned() else {
        return error_response(
            StatusCode::TOO_MANY_REQUESTS,
            "another business workflow is already running",
        );
    };
    run_trigger_once(&state, &trigger).await
}

async fn run_trigger_once(state: &TriggerState, trigger: &WorkflowTrigger) -> Response {
    let server = QuickDeskMcpServer::new(
        state.ws.clone(),
        state.allowed_devices.clone(),
        state.shared_state.clone(),
    );
    let result = server
        .execute_workflow_replay(
            &trigger.workflow_id,
            &trigger.device_id,
            trigger.overrides.clone(),
            trigger.stop_on_failure,
        )
        .await;
    match result {
        Ok(report) => {
            if let Err(error) = state.triggers.record_run(
                &trigger.id,
                &report.run_id,
                report.success,
                "",
                Utc::now(),
            ) {
                return error_response(StatusCode::INTERNAL_SERVER_ERROR, &error);
            }
            Json(report).into_response()
        }
        Err(error) => {
            let _ = state
                .triggers
                .record_run(&trigger.id, "", false, &error, Utc::now());
            workflow_error_response(error)
        }
    }
}

fn workflow_error_response(error: String) -> Response {
    error_response(workflow_error_status(&error), &error)
}

fn workflow_error_status(error: &str) -> StatusCode {
    if error == "workflow not found" {
        StatusCode::NOT_FOUND
    } else if error.starts_with("Device '") {
        StatusCode::FORBIDDEN
    } else {
        StatusCode::INTERNAL_SERVER_ERROR
    }
}

fn read_idempotency_key(headers: &HeaderMap) -> Result<Option<String>, String> {
    let Some(value) = headers.get("Idempotency-Key") else {
        return Ok(None);
    };
    let value = value
        .to_str()
        .map_err(|_| "Idempotency-Key must be valid UTF-8".to_string())?;
    let value = value.trim();
    if value.is_empty() || value.len() > 128 {
        return Err("Idempotency-Key must be 1..128 characters".to_string());
    }
    Ok(Some(value.to_string()))
}

fn replay_idempotent_response(
    state: &TriggerState,
    run_id: Option<String>,
    error: Option<String>,
    status_code: u16,
) -> Response {
    let status = StatusCode::from_u16(status_code).unwrap_or(StatusCode::INTERNAL_SERVER_ERROR);
    if status.is_success() {
        let Some(run_id) = run_id else {
            return error_response(
                StatusCode::INTERNAL_SERVER_ERROR,
                "idempotent run report is missing",
            );
        };
        let Some(report) = state.shared_state.workflow.get_run_report(&run_id) else {
            return error_response(
                StatusCode::INTERNAL_SERVER_ERROR,
                "idempotent workflow run report not found",
            );
        };
        return Json(report).into_response();
    }
    error_response(
        status,
        error
            .as_deref()
            .unwrap_or("the original request with this Idempotency-Key failed"),
    )
}

pub fn start_scheduler(
    cancellation: CancellationToken,
    ws: WsClient,
    shared_state: Arc<SharedState>,
    allowed_devices: Vec<String>,
    triggers: WorkflowTriggerStore,
    idempotency: WorkflowIdempotencyStore,
    run_slots: Arc<Semaphore>,
) {
    tokio::spawn(async move {
        let mut ticker = tokio::time::interval(Duration::from_secs(5));
        loop {
            tokio::select! {
                _ = cancellation.cancelled() => break,
                _ = ticker.tick() => {
                    let Ok(_permit) = run_slots.clone().try_acquire_owned() else {
                        continue;
                    };
                    let state = TriggerState {
                        token: None,
                        ws: ws.clone(),
                        shared_state: shared_state.clone(),
                        allowed_devices: allowed_devices.clone(),
                        run_slots: run_slots.clone(),
                        triggers: triggers.clone(),
                        idempotency: idempotency.clone(),
                    };
                    run_due_trigger(&state, Utc::now()).await;
                }
            }
        }
    });
}

async fn run_due_trigger(state: &TriggerState, now: chrono::DateTime<Utc>) -> usize {
    let due = state.triggers.claim_due_limit(now, 1);
    for trigger in &due {
        let _ = run_trigger_once(state, trigger).await;
    }
    due.len()
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
    use crate::workflow_trigger::WorkflowTriggerStore;
    use crate::ws_client::WsClient;
    use axum::body::{to_bytes, Body};
    use axum::http::{Request, StatusCode};
    use chrono::TimeZone;
    use futures_util::{SinkExt, StreamExt};
    use serde_json::json;
    use std::sync::atomic::{AtomicUsize, Ordering};
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
        let app = router(
            Some("business-secret".to_string()),
            ws,
            shared,
            Vec::new(),
            WorkflowTriggerStore::with_storage_dir(temp.join("triggers")),
            WorkflowIdempotencyStore::with_storage_path(temp.join("idempotency.json")),
            Arc::new(Semaphore::new(1)),
        );

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
    async fn repeated_idempotency_key_returns_the_first_workflow_run() {
        let temp = std::env::temp_dir().join(format!("quickdesk-trigger-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&temp).unwrap();

        let workflows = WorkflowStore::with_storage_dir(temp.join("workflows"));
        let workflow_id = workflows
            .start_recording("idempotent workflow", "device-1")
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
        let executions = Arc::new(AtomicUsize::new(0));
        let executions_for_server = executions.clone();
        let fake_quickdesk = tokio::spawn(async move {
            let (stream, _) = listener.accept().await.unwrap();
            let mut socket = accept_async(stream).await.unwrap();
            while let Some(Ok(Message::Text(text))) = socket.next().await {
                let request: serde_json::Value = serde_json::from_str(&text).unwrap();
                if let Some(id) = request.get("id").and_then(serde_json::Value::as_str) {
                    executions_for_server.fetch_add(1, Ordering::SeqCst);
                    let response = json!({"id": id, "result": {"processes": []}});
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
        let app = router(
            Some("business-secret".to_string()),
            ws,
            shared,
            Vec::new(),
            WorkflowTriggerStore::with_storage_dir(temp.join("triggers")),
            WorkflowIdempotencyStore::with_storage_path(temp.join("idempotency.json")),
            Arc::new(Semaphore::new(1)),
        );

        let idempotency_key = format!("request-{}", Uuid::new_v4());
        let request = Request::builder()
            .method("POST")
            .uri(format!("/v1/workflows/{workflow_id}/run"))
            .header("Authorization", "Bearer business-secret")
            .header("Idempotency-Key", &idempotency_key)
            .header("Content-Type", "application/json")
            .body(Body::from(json!({"device_id": "device-1"}).to_string()))
            .unwrap();
        let response = app.clone().oneshot(request).await.unwrap();
        assert_eq!(response.status(), StatusCode::OK);
        let body = to_bytes(response.into_body(), 1024 * 1024).await.unwrap();
        let first: WorkflowRunReport = serde_json::from_slice(&body).unwrap();

        let request = Request::builder()
            .method("POST")
            .uri(format!("/v1/workflows/{workflow_id}/run"))
            .header("Authorization", "Bearer business-secret")
            .header("Idempotency-Key", &idempotency_key)
            .header("Content-Type", "application/json")
            .body(Body::from(json!({"device_id": "device-1"}).to_string()))
            .unwrap();
        let response = app.oneshot(request).await.unwrap();
        assert_eq!(response.status(), StatusCode::OK);
        let body = to_bytes(response.into_body(), 1024 * 1024).await.unwrap();
        let second: WorkflowRunReport = serde_json::from_slice(&body).unwrap();

        assert_eq!(second.run_id, first.run_id);
        assert_eq!(executions.load(Ordering::SeqCst), 1);
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
            WorkflowTriggerStore::with_storage_dir(temp.join("triggers")),
            WorkflowIdempotencyStore::with_storage_path(temp.join("idempotency.json")),
            Arc::new(Semaphore::new(1)),
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
            WorkflowTriggerStore::with_storage_dir(temp.join("triggers")),
            WorkflowIdempotencyStore::with_storage_path(temp.join("idempotency.json")),
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

    #[tokio::test]
    async fn trigger_definitions_can_be_created_listed_and_deleted_over_http() {
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
        let shared = Arc::new(SharedState {
            memory: MemoryStore::with_db_path(temp.join("memory.db")).unwrap(),
            workflow: workflows,
            trust: TrustEngine::with_db_path(temp.join("trust.db")).unwrap(),
        });
        let app = router(
            Some("business-secret".to_string()),
            WsClient::disconnected(),
            shared,
            Vec::new(),
            WorkflowTriggerStore::with_storage_dir(temp.join("triggers")),
            WorkflowIdempotencyStore::with_storage_path(temp.join("idempotency.json")),
            Arc::new(Semaphore::new(1)),
        );

        let request = Request::builder()
            .method("POST")
            .uri("/v1/triggers")
            .header("Authorization", "Bearer business-secret")
            .header("Content-Type", "application/json")
            .body(Body::from(
                json!({
                    "name": "nightly",
                    "workflow_id": workflow_id,
                    "device_id": "device-1",
                    "interval_seconds": 60
                })
                .to_string(),
            ))
            .unwrap();
        let response = app.clone().oneshot(request).await.unwrap();
        assert_eq!(response.status(), StatusCode::CREATED);
        let body = to_bytes(response.into_body(), 1024 * 1024).await.unwrap();
        let trigger: WorkflowTrigger = serde_json::from_slice(&body).unwrap();
        assert_eq!(trigger.name, "nightly");
        assert!(trigger.next_run_at.is_some());

        let request = Request::builder()
            .uri("/v1/triggers")
            .header("Authorization", "Bearer business-secret")
            .body(Body::empty())
            .unwrap();
        let response = app.clone().oneshot(request).await.unwrap();
        assert_eq!(response.status(), StatusCode::OK);
        let body = to_bytes(response.into_body(), 1024 * 1024).await.unwrap();
        let triggers: Vec<WorkflowTrigger> = serde_json::from_slice(&body).unwrap();
        assert_eq!(triggers.len(), 1);

        let request = Request::builder()
            .method("DELETE")
            .uri(format!("/v1/triggers/{}", trigger.id))
            .header("Authorization", "Bearer business-secret")
            .body(Body::empty())
            .unwrap();
        let response = app.oneshot(request).await.unwrap();
        assert_eq!(response.status(), StatusCode::NO_CONTENT);

        let _ = std::fs::remove_dir_all(temp);
    }

    #[tokio::test]
    async fn due_trigger_runs_through_quickdesk_and_records_its_report() {
        let temp = std::env::temp_dir().join(format!("quickdesk-trigger-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&temp).unwrap();
        let workflows = WorkflowStore::with_storage_dir(temp.join("workflows"));
        let workflow_id = workflows
            .start_recording("scheduled workflow", "device-1")
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
                    let response = json!({"id": id, "result": {"processes": []}});
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
        let trigger_store = WorkflowTriggerStore::with_storage_dir(temp.join("triggers"));
        let created_at = Utc.with_ymd_and_hms(2026, 10, 17, 15, 0, 0).unwrap();
        let trigger = trigger_store
            .create_at(
                WorkflowTriggerSpec {
                    name: "every minute".to_string(),
                    workflow_id,
                    device_id: "device-1".to_string(),
                    overrides: HashMap::new(),
                    stop_on_failure: true,
                    interval_seconds: Some(60),
                },
                created_at,
            )
            .unwrap();
        let state = TriggerState {
            token: None,
            ws,
            shared_state: Arc::new(SharedState {
                memory: MemoryStore::with_db_path(temp.join("memory.db")).unwrap(),
                workflow: workflows,
                trust: TrustEngine::with_db_path(temp.join("trust.db")).unwrap(),
            }),
            allowed_devices: Vec::new(),
            run_slots: Arc::new(Semaphore::new(1)),
            triggers: trigger_store.clone(),
            idempotency: WorkflowIdempotencyStore::with_storage_path(
                temp.join("idempotency.json"),
            ),
        };

        let ran = run_due_trigger(&state, created_at + chrono::Duration::seconds(60)).await;

        assert_eq!(ran, 1);
        let loaded = trigger_store.get(&trigger.id).unwrap();
        assert_eq!(loaded.last_status, "completed");
        assert!(!loaded.last_run_id.is_empty());
        fake_quickdesk.abort();
        let _ = std::fs::remove_dir_all(temp);
    }
}
