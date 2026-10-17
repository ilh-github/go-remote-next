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

use crate::callback::deliver_workflow_callback;
use crate::idempotency::{IdempotencyClaim, WorkflowIdempotencyStore};
use crate::server::{QuickDeskMcpServer, SharedState};
use crate::workflow_job::{CancelJobError, WorkflowJobManager};
use crate::workflow_trigger::{WorkflowTrigger, WorkflowTriggerSpec, WorkflowTriggerStore};
use crate::ws_client::WsClient;

#[derive(Clone)]
pub struct TriggerRuntime {
    pub token: Option<String>,
    pub ws: WsClient,
    pub shared_state: Arc<SharedState>,
    pub allowed_devices: Vec<String>,
    pub run_slots: Arc<Semaphore>,
    pub triggers: WorkflowTriggerStore,
    pub idempotency: WorkflowIdempotencyStore,
    pub jobs: WorkflowJobManager,
}

type TriggerState = TriggerRuntime;

#[derive(Deserialize)]
struct TriggerRunRequest {
    device_id: String,
    overrides: Option<HashMap<String, serde_json::Value>>,
    stop_on_failure: Option<bool>,
    callback_url: Option<String>,
}

#[derive(Deserialize)]
struct CreateTriggerRequest {
    name: String,
    workflow_id: String,
    device_id: String,
    overrides: Option<HashMap<String, serde_json::Value>>,
    stop_on_failure: Option<bool>,
    interval_seconds: Option<u64>,
    event_type: Option<String>,
    event_device_id: Option<String>,
    event_filter: Option<serde_json::Value>,
    callback_url: Option<String>,
}

pub fn router(runtime: TriggerRuntime) -> Router {
    Router::new()
        .route("/v1/workflows/{workflow_id}/run", post(run_workflow))
        .route(
            "/v1/workflows/{workflow_id}/run-async",
            post(run_workflow_async),
        )
        .route("/v1/workflow-runs/{run_id}", get(get_workflow_run))
        .route(
            "/v1/workflow-runs/{run_id}/status",
            get(get_workflow_job),
        )
        .route(
            "/v1/workflow-runs/{run_id}/cancel",
            post(cancel_workflow_job),
        )
        .route("/v1/triggers", get(list_triggers).post(create_trigger))
        .route(
            "/v1/triggers/{trigger_id}",
            axum::routing::delete(delete_trigger),
        )
        .route("/v1/triggers/{trigger_id}/run", post(run_saved_trigger))
        .with_state(runtime)
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
    let callback_url = request.callback_url;
    let idempotency_request = json!({
        "overrides": overrides.clone(),
        "stop_on_failure": stop_on_failure,
        "callback_url": callback_url.clone(),
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
            deliver_workflow_callback(
                callback_url.as_deref(),
                json!({
                    "event": "workflow.run.completed",
                    "run_id": report.run_id.clone(),
                    "workflow_id": report.workflow_id.clone(),
                    "device_id": report.device_id.clone(),
                    "success": report.success,
                    "cancelled": report.cancelled,
                    "report": report.clone(),
                }),
            )
            .await;
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
            deliver_workflow_callback(
                callback_url.as_deref(),
                json!({
                    "event": "workflow.run.failed",
                    "workflow_id": workflow_id,
                    "device_id": device_id,
                    "status_code": status.as_u16(),
                    "error": error.clone(),
                }),
            )
            .await;
            error_response(status, &error)
        }
    }
}

async fn run_workflow_async(
    State(state): State<TriggerState>,
    Path(workflow_id): Path<String>,
    headers: HeaderMap,
    Json(request): Json<TriggerRunRequest>,
) -> Response {
    if let Err(error) = authorize_request(&state, &headers) {
        return auth_error_response(error);
    }
    let device_id = request.device_id.trim();
    if device_id.is_empty() {
        return error_response(StatusCode::BAD_REQUEST, "device_id is required");
    }
    if state.shared_state.workflow.get_workflow(&workflow_id).is_none() {
        return error_response(StatusCode::NOT_FOUND, "workflow not found");
    }
    match state.jobs.enqueue(
        &workflow_id,
        device_id,
        request.overrides.unwrap_or_default(),
        request.stop_on_failure.unwrap_or(true),
        request.callback_url,
    ) {
        Ok(job) => (StatusCode::ACCEPTED, Json(job)).into_response(),
        Err(error) if error.contains("queue is full") => {
            error_response(StatusCode::TOO_MANY_REQUESTS, &error)
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

async fn get_workflow_job(
    State(state): State<TriggerState>,
    Path(run_id): Path<String>,
    headers: HeaderMap,
) -> Response {
    if let Err(error) = authorize_request(&state, &headers) {
        return auth_error_response(error);
    }
    match state.jobs.get(&run_id) {
        Some(job) => Json(job).into_response(),
        None => error_response(StatusCode::NOT_FOUND, "workflow job not found"),
    }
}

async fn cancel_workflow_job(
    State(state): State<TriggerState>,
    Path(run_id): Path<String>,
    headers: HeaderMap,
) -> Response {
    if let Err(error) = authorize_request(&state, &headers) {
        return auth_error_response(error);
    }
    match state.jobs.cancel(&run_id) {
        Ok(job) => (StatusCode::ACCEPTED, Json(job)).into_response(),
        Err(CancelJobError::NotFound) => {
            error_response(StatusCode::NOT_FOUND, "workflow job not found")
        }
        Err(CancelJobError::NotCancellable) => {
            error_response(StatusCode::CONFLICT, "workflow job is not cancellable")
        }
        Err(CancelJobError::Store(error)) => {
            error_response(StatusCode::INTERNAL_SERVER_ERROR, &error)
        }
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
        event_type: request.event_type,
        event_device_id: request.event_device_id,
        event_filter: request.event_filter,
        callback_url: request.callback_url,
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
            deliver_workflow_callback(
                trigger.callback_url.as_deref(),
                json!({
                    "event": "workflow.run.completed",
                    "run_id": report.run_id.clone(),
                    "workflow_id": report.workflow_id.clone(),
                    "device_id": report.device_id.clone(),
                    "success": report.success,
                    "cancelled": report.cancelled,
                    "report": report.clone(),
                }),
            )
            .await;
            Json(report).into_response()
        }
        Err(error) => {
            let _ = state
                .triggers
                .record_run(&trigger.id, "", false, &error, Utc::now());
            deliver_workflow_callback(
                trigger.callback_url.as_deref(),
                json!({
                    "event": "workflow.run.failed",
                    "workflow_id": trigger.workflow_id,
                    "device_id": trigger.device_id,
                    "error": error.clone(),
                }),
            )
            .await;
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
    runtime: TriggerRuntime,
) {
    start_event_worker(cancellation.clone(), runtime.clone());
    tokio::spawn(async move {
        let mut ticker = tokio::time::interval(Duration::from_secs(5));
        loop {
            tokio::select! {
                _ = cancellation.cancelled() => break,
                _ = ticker.tick() => {
                    let Ok(_permit) = runtime.run_slots.clone().try_acquire_owned() else {
                        continue;
                    };
                    let mut state = runtime.clone();
                    state.token = None;
                    run_due_trigger(&state, Utc::now()).await;
                }
            }
        }
    });
}

pub fn start_event_worker(cancellation: CancellationToken, runtime: TriggerRuntime) {
    tokio::spawn(async move {
        let mut events = runtime.ws.event_bus().subscribe();
        loop {
            let received = tokio::select! {
                _ = cancellation.cancelled() => break,
                received = events.recv() => received,
            };
            match received {
                Ok(event) if event.event == "controlPlaneEvent" => {
                    handle_control_plane_event(&runtime, event.data, &cancellation).await;
                }
                Ok(_) => {}
                Err(tokio::sync::broadcast::error::RecvError::Lagged(skipped)) => {
                    tracing::warn!(
                        "workflow event trigger subscriber lagged and skipped {skipped} events"
                    );
                }
                Err(tokio::sync::broadcast::error::RecvError::Closed) => break,
            }
        }
    });
}

async fn handle_control_plane_event(
    runtime: &TriggerRuntime,
    envelope: serde_json::Value,
    cancellation: &CancellationToken,
) {
    let event_type = envelope
        .get("type")
        .and_then(serde_json::Value::as_str)
        .unwrap_or_default();
    let event_device_id = envelope
        .get("device_id")
        .and_then(serde_json::Value::as_str)
        .or_else(|| {
            envelope
                .get("data")
                .and_then(|data| data.get("device_id"))
                .and_then(serde_json::Value::as_str)
        })
        .unwrap_or_default();
    let event_id = envelope.get("id").and_then(serde_json::Value::as_str);
    let server_rev = envelope
        .get("server_rev")
        .and_then(serde_json::Value::as_i64)
        .unwrap_or(0);
    if event_type.is_empty() || (server_rev <= 0 && event_id.is_none()) {
        return;
    }
    let event_data = envelope
        .get("data")
        .cloned()
        .unwrap_or(serde_json::Value::Null);

    let matching = runtime
        .triggers
        .list()
        .into_iter()
        .filter(|trigger| trigger.enabled)
        .filter(|trigger| trigger.event_type.as_deref() == Some(event_type))
        .filter(|trigger| {
            trigger
                .event_device_id
                .as_deref()
                .is_none_or(|device_id| device_id == event_device_id)
        })
        .filter(|trigger| {
            trigger
                .event_filter
                .as_ref()
                .is_none_or(|filter| event_filter_matches(&event_data, filter))
        })
        .collect::<Vec<_>>();
    if matching.is_empty() {
        return;
    }

    let permit = tokio::select! {
        _ = cancellation.cancelled() => return,
        permit = runtime.run_slots.clone().acquire_owned() => permit,
    };
    let Ok(_permit) = permit else {
        return;
    };

    for trigger in matching {
        let claimed = runtime
            .triggers
            .claim_event(&trigger.id, event_id, server_rev, Utc::now());
        match claimed {
            Ok(true) => {
                let _ = run_trigger_once(runtime, &trigger).await;
            }
            Ok(false) => {}
            Err(error) => {
                tracing::warn!(
                    "workflow event trigger {} could not claim event {}: {}",
                    trigger.id,
                    event_id.unwrap_or("-"),
                    error
                );
            }
        }
    }
}

fn event_filter_matches(data: &serde_json::Value, filter: &serde_json::Value) -> bool {
    let (Some(data), Some(filter)) = (data.as_object(), filter.as_object()) else {
        return data == filter;
    };
    filter
        .iter()
        .all(|(key, expected)| data.get(key) == Some(expected))
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
    use crate::workflow_job::WorkflowJobStore;
    use crate::workflow_trigger::WorkflowTriggerStore;
    use crate::ws_client::WsClient;
    use axum::body::{to_bytes, Body};
    use axum::http::{Request, StatusCode};
    use chrono::TimeZone;
    use futures_util::{SinkExt, StreamExt};
    use serde_json::json;
    use std::sync::atomic::{AtomicUsize, Ordering};
    use std::sync::Arc;
    use tokio::io::{AsyncReadExt, AsyncWriteExt};
    use tokio_tungstenite::{accept_async, tungstenite::Message};
    use tower::ServiceExt;
    use uuid::Uuid;

    fn test_jobs(
        temp: &std::path::Path,
        ws: WsClient,
        shared: Arc<SharedState>,
        slots: Arc<Semaphore>,
    ) -> WorkflowJobManager {
        WorkflowJobManager::with_store(
            WorkflowJobStore::with_storage_dir(temp.join("jobs")),
            CancellationToken::new(),
            ws,
            shared,
            Vec::new(),
            slots,
        )
        .unwrap()
    }

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
        let slots = Arc::new(Semaphore::new(1));
        let jobs = test_jobs(&temp, ws.clone(), shared.clone(), slots.clone());
        let app = router(TriggerRuntime {
            token: Some("business-secret".to_string()),
            ws,
            shared_state: shared,
            allowed_devices: Vec::new(),
            run_slots: slots,
            triggers: WorkflowTriggerStore::with_storage_dir(temp.join("triggers")),
            idempotency: WorkflowIdempotencyStore::with_storage_path(
                temp.join("idempotency.json"),
            ),
            jobs,
        });

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
    async fn async_workflow_run_returns_accepted_job() {
        let temp = std::env::temp_dir().join(format!("quickdesk-trigger-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&temp).unwrap();
        let workflows = WorkflowStore::with_storage_dir(temp.join("workflows"));
        let workflow_id = workflows
            .start_recording("async workflow", "device-1")
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
        let slots = Arc::new(Semaphore::new(1));
        let jobs = test_jobs(
            &temp,
            WsClient::disconnected(),
            shared.clone(),
            slots.clone(),
        );
        let app = router(TriggerRuntime {
            token: Some("business-secret".to_string()),
            ws: WsClient::disconnected(),
            shared_state: shared,
            allowed_devices: Vec::new(),
            run_slots: slots,
            triggers: WorkflowTriggerStore::with_storage_dir(temp.join("triggers")),
            idempotency: WorkflowIdempotencyStore::with_storage_path(
                temp.join("idempotency.json"),
            ),
            jobs,
        });

        let request = Request::builder()
            .method("POST")
            .uri(format!("/v1/workflows/{workflow_id}/run-async"))
            .header("Authorization", "Bearer business-secret")
            .header("Content-Type", "application/json")
            .body(Body::from(json!({"device_id": "device-1"}).to_string()))
            .unwrap();
        let response = app.oneshot(request).await.unwrap();
        assert_eq!(response.status(), StatusCode::ACCEPTED);
        let body = to_bytes(response.into_body(), 1024 * 1024).await.unwrap();
        let job: serde_json::Value = serde_json::from_slice(&body).unwrap();
        assert_eq!(job["workflow_id"], workflow_id);
        assert_eq!(job["status"], "queued");
        assert!(job["run_id"].as_str().is_some_and(|id| !id.is_empty()));
        let _ = std::fs::remove_dir_all(temp);
    }

    #[tokio::test]
    async fn async_workflow_job_executes_and_reports_completion() {
        let temp = std::env::temp_dir().join(format!("quickdesk-trigger-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&temp).unwrap();
        std::env::set_var("QUICKDESK_CALLBACK_SECRET", "async-callback-test-secret");
        let workflows = WorkflowStore::with_storage_dir(temp.join("workflows"));
        let workflow_id = workflows
            .start_recording("async execution", "device-1")
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
        let callback_listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
        let callback_address = callback_listener.local_addr().unwrap();
        let (callback_tx, callback_rx) = tokio::sync::oneshot::channel();
        let callback_server = tokio::spawn(async move {
            let (mut stream, _) = callback_listener.accept().await.unwrap();
            let mut buffer = vec![0u8; 8192];
            let read = stream.read(&mut buffer).await.unwrap();
            let request = String::from_utf8_lossy(&buffer[..read]).to_string();
            let _ = callback_tx.send(request);
            stream
                .write_all(b"HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n")
                .await
                .unwrap();
        });
        let shared = Arc::new(SharedState {
            memory: MemoryStore::with_db_path(temp.join("memory.db")).unwrap(),
            workflow: workflows,
            trust: TrustEngine::with_db_path(temp.join("trust.db")).unwrap(),
        });
        let slots = Arc::new(Semaphore::new(1));
        let jobs = test_jobs(&temp, ws.clone(), shared.clone(), slots.clone());
        let app = router(TriggerRuntime {
            token: Some("business-secret".to_string()),
            ws,
            shared_state: shared,
            allowed_devices: Vec::new(),
            run_slots: slots,
            triggers: WorkflowTriggerStore::with_storage_dir(temp.join("triggers")),
            idempotency: WorkflowIdempotencyStore::with_storage_path(
                temp.join("idempotency.json"),
            ),
            jobs,
        });

        let request = Request::builder()
            .method("POST")
            .uri(format!("/v1/workflows/{workflow_id}/run-async"))
            .header("Authorization", "Bearer business-secret")
            .header("Content-Type", "application/json")
            .body(Body::from(
                json!({
                    "device_id": "device-1",
                    "callback_url": format!("http://{callback_address}/async-callback")
                })
                .to_string(),
            ))
            .unwrap();
        let response = app.clone().oneshot(request).await.unwrap();
        assert_eq!(response.status(), StatusCode::ACCEPTED);
        let body = to_bytes(response.into_body(), 1024 * 1024).await.unwrap();
        let accepted: serde_json::Value = serde_json::from_slice(&body).unwrap();
        let run_id = accepted["run_id"].as_str().unwrap().to_string();

        let status = tokio::time::timeout(Duration::from_secs(2), async {
            loop {
                let request = Request::builder()
                    .uri(format!("/v1/workflow-runs/{run_id}/status"))
                    .header("Authorization", "Bearer business-secret")
                    .body(Body::empty())
                    .unwrap();
                let response = app.clone().oneshot(request).await.unwrap();
                assert_eq!(response.status(), StatusCode::OK);
                let body = to_bytes(response.into_body(), 1024 * 1024).await.unwrap();
                let status: serde_json::Value = serde_json::from_slice(&body).unwrap();
                if status["status"] == "completed" {
                    return status;
                }
                tokio::time::sleep(Duration::from_millis(10)).await;
            }
        })
        .await
        .unwrap();

        assert_eq!(status["run_id"], run_id);
        assert_eq!(status["report"]["success"], true);
        assert_eq!(status["report"]["cancelled"], false);
        let callback = tokio::time::timeout(Duration::from_secs(2), callback_rx)
            .await
            .unwrap()
            .unwrap();
        assert!(callback.starts_with("POST /async-callback HTTP/1.1"));
        assert!(callback.contains("\"event\":\"workflow.run.completed\""));
        callback_server.abort();
        fake_quickdesk.abort();
        let _ = std::fs::remove_dir_all(temp);
    }

    #[tokio::test]
    async fn cancelling_a_running_async_job_produces_a_cancelled_report() {
        let temp = std::env::temp_dir().join(format!("quickdesk-trigger-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&temp).unwrap();
        let workflows = WorkflowStore::with_storage_dir(temp.join("workflows"));
        let workflow_id = workflows
            .start_recording("cancel async workflow", "device-1")
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
        let (started_tx, started_rx) = tokio::sync::oneshot::channel();
        let fake_quickdesk = tokio::spawn(async move {
            let (stream, _) = listener.accept().await.unwrap();
            let mut socket = accept_async(stream).await.unwrap();
            if let Some(Ok(Message::Text(_text))) = socket.next().await {
                let _ = started_tx.send(());
            }
            while socket.next().await.is_some() {}
        });
        let ws = WsClient::connect(&format!("ws://{address}"), None, EventBus::new(100))
            .await
            .unwrap();
        let shared = Arc::new(SharedState {
            memory: MemoryStore::with_db_path(temp.join("memory.db")).unwrap(),
            workflow: workflows,
            trust: TrustEngine::with_db_path(temp.join("trust.db")).unwrap(),
        });
        let slots = Arc::new(Semaphore::new(1));
        let jobs = test_jobs(&temp, ws.clone(), shared.clone(), slots.clone());
        let app = router(TriggerRuntime {
            token: Some("business-secret".to_string()),
            ws,
            shared_state: shared,
            allowed_devices: Vec::new(),
            run_slots: slots,
            triggers: WorkflowTriggerStore::with_storage_dir(temp.join("triggers")),
            idempotency: WorkflowIdempotencyStore::with_storage_path(
                temp.join("idempotency.json"),
            ),
            jobs,
        });

        let request = Request::builder()
            .method("POST")
            .uri(format!("/v1/workflows/{workflow_id}/run-async"))
            .header("Authorization", "Bearer business-secret")
            .header("Content-Type", "application/json")
            .body(Body::from(json!({"device_id": "device-1"}).to_string()))
            .unwrap();
        let response = app.clone().oneshot(request).await.unwrap();
        assert_eq!(response.status(), StatusCode::ACCEPTED);
        let body = to_bytes(response.into_body(), 1024 * 1024).await.unwrap();
        let accepted: serde_json::Value = serde_json::from_slice(&body).unwrap();
        let run_id = accepted["run_id"].as_str().unwrap().to_string();

        tokio::time::timeout(Duration::from_secs(2), started_rx)
            .await
            .unwrap()
            .unwrap();

        let request = Request::builder()
            .method("POST")
            .uri(format!("/v1/workflow-runs/{run_id}/cancel"))
            .header("Authorization", "Bearer business-secret")
            .body(Body::empty())
            .unwrap();
        let response = app.clone().oneshot(request).await.unwrap();
        assert_eq!(response.status(), StatusCode::ACCEPTED);
        let body = to_bytes(response.into_body(), 1024 * 1024).await.unwrap();
        let cancelled: serde_json::Value = serde_json::from_slice(&body).unwrap();
        assert_eq!(cancelled["status"], "cancelling");

        let status = tokio::time::timeout(Duration::from_secs(2), async {
            loop {
                let request = Request::builder()
                    .uri(format!("/v1/workflow-runs/{run_id}/status"))
                    .header("Authorization", "Bearer business-secret")
                    .body(Body::empty())
                    .unwrap();
                let response = app.clone().oneshot(request).await.unwrap();
                assert_eq!(response.status(), StatusCode::OK);
                let body = to_bytes(response.into_body(), 1024 * 1024).await.unwrap();
                let status: serde_json::Value = serde_json::from_slice(&body).unwrap();
                if status["status"] == "cancelled" {
                    return status;
                }
                tokio::time::sleep(Duration::from_millis(10)).await;
            }
        })
        .await
        .unwrap();

        assert_eq!(status["report"]["cancelled"], true);
        assert_eq!(status["report"]["success"], false);
        fake_quickdesk.abort();
        let _ = std::fs::remove_dir_all(temp);
    }

    #[tokio::test]
    async fn completed_workflow_posts_a_signed_callback() {
        let temp = std::env::temp_dir().join(format!("quickdesk-trigger-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&temp).unwrap();
        std::env::set_var("QUICKDESK_CALLBACK_SECRET", "callback-test-secret");
        let workflows = WorkflowStore::with_storage_dir(temp.join("workflows"));
        let workflow_id = workflows
            .start_recording("callback workflow", "device-1")
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

        let quickdesk_listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
        let quickdesk_address = quickdesk_listener.local_addr().unwrap();
        let fake_quickdesk = tokio::spawn(async move {
            let (stream, _) = quickdesk_listener.accept().await.unwrap();
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
        let ws = WsClient::connect(&format!("ws://{quickdesk_address}"), None, EventBus::new(100))
            .await
            .unwrap();

        let callback_listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
        let callback_address = callback_listener.local_addr().unwrap();
        let (callback_tx, callback_rx) = tokio::sync::oneshot::channel();
        let callback_server = tokio::spawn(async move {
            let (mut stream, _) = callback_listener.accept().await.unwrap();
            let mut buffer = vec![0u8; 8192];
            let read = stream.read(&mut buffer).await.unwrap();
            let request = String::from_utf8_lossy(&buffer[..read]).to_string();
            let _ = callback_tx.send(request);
            stream
                .write_all(b"HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n")
                .await
                .unwrap();
        });

        let shared = Arc::new(SharedState {
            memory: MemoryStore::with_db_path(temp.join("memory.db")).unwrap(),
            workflow: workflows,
            trust: TrustEngine::with_db_path(temp.join("trust.db")).unwrap(),
        });
        let slots = Arc::new(Semaphore::new(1));
        let jobs = test_jobs(&temp, ws.clone(), shared.clone(), slots.clone());
        let app = router(TriggerRuntime {
            token: Some("business-secret".to_string()),
            ws,
            shared_state: shared,
            allowed_devices: Vec::new(),
            run_slots: slots,
            triggers: WorkflowTriggerStore::with_storage_dir(temp.join("triggers")),
            idempotency: WorkflowIdempotencyStore::with_storage_path(
                temp.join("idempotency.json"),
            ),
            jobs,
        });

        let request = Request::builder()
            .method("POST")
            .uri(format!("/v1/workflows/{workflow_id}/run"))
            .header("Authorization", "Bearer business-secret")
            .header("Content-Type", "application/json")
            .body(Body::from(
                json!({
                    "device_id": "device-1",
                    "callback_url": format!("http://{callback_address}/workflow-callback")
                })
                .to_string(),
            ))
            .unwrap();
        let response = app.oneshot(request).await.unwrap();
        assert_eq!(response.status(), StatusCode::OK);

        let callback = tokio::time::timeout(Duration::from_secs(2), callback_rx)
            .await
            .unwrap()
            .unwrap();
        assert!(callback.starts_with("POST /workflow-callback HTTP/1.1"));
        assert!(callback
            .to_lowercase()
            .contains("x-quickdesk-signature: hmac-sha1="));
        assert!(callback.contains("\"event\":\"workflow.run.completed\""));
        callback_server.abort();
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
        let slots = Arc::new(Semaphore::new(1));
        let jobs = test_jobs(&temp, ws.clone(), shared.clone(), slots.clone());
        let app = router(TriggerRuntime {
            token: Some("business-secret".to_string()),
            ws,
            shared_state: shared,
            allowed_devices: Vec::new(),
            run_slots: slots,
            triggers: WorkflowTriggerStore::with_storage_dir(temp.join("triggers")),
            idempotency: WorkflowIdempotencyStore::with_storage_path(
                temp.join("idempotency.json"),
            ),
            jobs,
        });

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
        let slots = Arc::new(Semaphore::new(1));
        let jobs = test_jobs(
            &temp,
            WsClient::disconnected(),
            shared.clone(),
            slots.clone(),
        );
        let app = router(TriggerRuntime {
            token: Some("business-secret".to_string()),
            ws: WsClient::disconnected(),
            shared_state: shared,
            allowed_devices: vec!["allowed-device".to_string()],
            run_slots: slots,
            triggers: WorkflowTriggerStore::with_storage_dir(temp.join("triggers")),
            idempotency: WorkflowIdempotencyStore::with_storage_path(
                temp.join("idempotency.json"),
            ),
            jobs,
        });
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
        let jobs = test_jobs(
            &temp,
            WsClient::disconnected(),
            shared.clone(),
            slots.clone(),
        );
        let app = router(TriggerRuntime {
            token: Some("business-secret".to_string()),
            ws: WsClient::disconnected(),
            shared_state: shared,
            allowed_devices: Vec::new(),
            run_slots: slots,
            triggers: WorkflowTriggerStore::with_storage_dir(temp.join("triggers")),
            idempotency: WorkflowIdempotencyStore::with_storage_path(
                temp.join("idempotency.json"),
            ),
            jobs,
        });

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
        let slots = Arc::new(Semaphore::new(1));
        let jobs = test_jobs(
            &temp,
            WsClient::disconnected(),
            shared.clone(),
            slots.clone(),
        );
        let app = router(TriggerRuntime {
            token: Some("business-secret".to_string()),
            ws: WsClient::disconnected(),
            shared_state: shared,
            allowed_devices: Vec::new(),
            run_slots: slots,
            triggers: WorkflowTriggerStore::with_storage_dir(temp.join("triggers")),
            idempotency: WorkflowIdempotencyStore::with_storage_path(
                temp.join("idempotency.json"),
            ),
            jobs,
        });

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
            .method("POST")
            .uri("/v1/triggers")
            .header("Authorization", "Bearer business-secret")
            .header("Content-Type", "application/json")
            .body(Body::from(
                json!({
                    "name": "denied",
                    "workflow_id": workflow_id,
                    "device_id": "device-1",
                    "event_type": "device.access.denied",
                    "event_filter": {"reason": "policy"}
                })
                .to_string(),
            ))
            .unwrap();
        let response = app.clone().oneshot(request).await.unwrap();
        assert_eq!(response.status(), StatusCode::CREATED);
        let body = to_bytes(response.into_body(), 1024 * 1024).await.unwrap();
        let event_trigger: WorkflowTrigger = serde_json::from_slice(&body).unwrap();
        assert_eq!(
            event_trigger.event_type.as_deref(),
            Some("device.access.denied")
        );
        assert_eq!(
            event_trigger.event_device_id.as_deref(),
            Some("device-1")
        );
        assert!(event_trigger.next_run_at.is_none());

        let request = Request::builder()
            .method("POST")
            .uri("/v1/triggers")
            .header("Authorization", "Bearer business-secret")
            .header("Content-Type", "application/json")
            .body(Body::from(
                json!({
                    "name": "no trigger",
                    "workflow_id": workflow_id,
                    "device_id": "device-1"
                })
                .to_string(),
            ))
            .unwrap();
        let response = app.clone().oneshot(request).await.unwrap();
        assert_eq!(response.status(), StatusCode::BAD_REQUEST);

        let request = Request::builder()
            .uri("/v1/triggers")
            .header("Authorization", "Bearer business-secret")
            .body(Body::empty())
            .unwrap();
        let response = app.clone().oneshot(request).await.unwrap();
        assert_eq!(response.status(), StatusCode::OK);
        let body = to_bytes(response.into_body(), 1024 * 1024).await.unwrap();
        let triggers: Vec<WorkflowTrigger> = serde_json::from_slice(&body).unwrap();
        assert_eq!(triggers.len(), 2);

        let request = Request::builder()
            .method("DELETE")
            .uri(format!("/v1/triggers/{}", trigger.id))
            .header("Authorization", "Bearer business-secret")
            .body(Body::empty())
            .unwrap();
        let response = app.clone().oneshot(request).await.unwrap();
        assert_eq!(response.status(), StatusCode::NO_CONTENT);

        let request = Request::builder()
            .method("DELETE")
            .uri(format!("/v1/triggers/{}", event_trigger.id))
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
        std::env::set_var("QUICKDESK_CALLBACK_SECRET", "scheduled-callback-secret");
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
        let callback_listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
        let callback_address = callback_listener.local_addr().unwrap();
        let (callback_tx, callback_rx) = tokio::sync::oneshot::channel();
        let callback_server = tokio::spawn(async move {
            let (mut stream, _) = callback_listener.accept().await.unwrap();
            let mut buffer = vec![0u8; 8192];
            let read = stream.read(&mut buffer).await.unwrap();
            let request = String::from_utf8_lossy(&buffer[..read]).to_string();
            let _ = callback_tx.send(request);
            stream
                .write_all(b"HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n")
                .await
                .unwrap();
        });
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
                    event_type: None,
                    event_device_id: None,
                    event_filter: None,
                    callback_url: Some(format!("http://{callback_address}/scheduled-callback")),
                },
                created_at,
            )
            .unwrap();
        let shared = Arc::new(SharedState {
            memory: MemoryStore::with_db_path(temp.join("memory.db")).unwrap(),
            workflow: workflows,
            trust: TrustEngine::with_db_path(temp.join("trust.db")).unwrap(),
        });
        let slots = Arc::new(Semaphore::new(1));
        let jobs = test_jobs(&temp, ws.clone(), shared.clone(), slots.clone());
        let state = TriggerState {
            token: None,
            ws,
            shared_state: shared,
            allowed_devices: Vec::new(),
            run_slots: slots,
            triggers: trigger_store.clone(),
            idempotency: WorkflowIdempotencyStore::with_storage_path(
                temp.join("idempotency.json"),
            ),
            jobs,
        };

        let ran = run_due_trigger(&state, created_at + chrono::Duration::seconds(60)).await;

        assert_eq!(ran, 1);
        let loaded = trigger_store.get(&trigger.id).unwrap();
        assert_eq!(loaded.last_status, "completed");
        assert!(!loaded.last_run_id.is_empty());
        let callback = tokio::time::timeout(Duration::from_secs(2), callback_rx)
            .await
            .unwrap()
            .unwrap();
        assert!(callback.starts_with("POST /scheduled-callback HTTP/1.1"));
        assert!(callback.contains("\"event\":\"workflow.run.completed\""));
        callback_server.abort();
        fake_quickdesk.abort();
        let _ = std::fs::remove_dir_all(temp);
    }

    #[tokio::test]
    async fn control_plane_event_runs_matching_trigger_once() {
        let temp = std::env::temp_dir().join(format!("quickdesk-trigger-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&temp).unwrap();
        let workflows = WorkflowStore::with_storage_dir(temp.join("workflows"));
        let workflow_id = workflows
            .start_recording("event workflow", "device-1")
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
        let requests = Arc::new(AtomicUsize::new(0));
        let request_count = requests.clone();
        let fake_quickdesk = tokio::spawn(async move {
            let (stream, _) = listener.accept().await.unwrap();
            let mut socket = accept_async(stream).await.unwrap();
            while let Some(Ok(Message::Text(text))) = socket.next().await {
                let request: serde_json::Value = serde_json::from_str(&text).unwrap();
                if let Some(id) = request.get("id").and_then(serde_json::Value::as_str) {
                    request_count.fetch_add(1, Ordering::SeqCst);
                    let response = json!({"id": id, "result": {"processes": []}});
                    socket
                        .send(Message::Text(response.to_string().into()))
                        .await
                        .unwrap();
                }
            }
        });
        let ws = WsClient::connect(
            &format!("ws://{address}"),
            None,
            EventBus::new(100),
        )
        .await
        .unwrap();
        let trigger_store = WorkflowTriggerStore::with_storage_dir(temp.join("triggers"));
        let trigger = trigger_store
            .create(
                WorkflowTriggerSpec {
                    name: "denied event".to_string(),
                    workflow_id,
                    device_id: "device-1".to_string(),
                    overrides: HashMap::new(),
                    stop_on_failure: true,
                    interval_seconds: None,
                    event_type: Some("device.access.denied".to_string()),
                    event_device_id: Some("device-1".to_string()),
                    event_filter: Some(json!({"reason": "policy"})),
                    callback_url: None,
                },
            )
            .unwrap();
        let task_trigger = trigger_store
            .create(
                WorkflowTriggerSpec {
                    name: "scheduled task failure".to_string(),
                    workflow_id: trigger.workflow_id.clone(),
                    device_id: "device-1".to_string(),
                    overrides: HashMap::new(),
                    stop_on_failure: true,
                    interval_seconds: None,
                    event_type: Some("scheduled_task.failed".to_string()),
                    event_device_id: None,
                    event_filter: Some(json!({"kind": "sign_out_all_sessions"})),
                    callback_url: None,
                },
            )
            .unwrap();
        let shared = Arc::new(SharedState {
            memory: MemoryStore::with_db_path(temp.join("memory.db")).unwrap(),
            workflow: workflows,
            trust: TrustEngine::with_db_path(temp.join("trust.db")).unwrap(),
        });
        let slots = Arc::new(Semaphore::new(1));
        let jobs = test_jobs(&temp, ws.clone(), shared.clone(), slots.clone());
        let state = TriggerState {
            token: None,
            ws: ws.clone(),
            shared_state: shared,
            allowed_devices: Vec::new(),
            run_slots: slots,
            triggers: trigger_store.clone(),
            idempotency: WorkflowIdempotencyStore::with_storage_path(
                temp.join("idempotency.json"),
            ),
            jobs,
        };
        let cancellation = CancellationToken::new();
        start_event_worker(cancellation.clone(), state);
        tokio::time::sleep(Duration::from_millis(20)).await;

        for ignored in [
            json!({
                "id": "evt-other-device",
                "type": "device.access.denied",
                "server_rev": 40,
                "data": {"device_id": "device-2", "reason": "policy"}
            }),
            json!({
                "id": "evt-other-reason",
                "type": "device.access.denied",
                "server_rev": 41,
                "data": {"device_id": "device-1", "reason": "expired"}
            }),
        ] {
            ws.event_bus()
                .publish(crate::event_bus::Event {
                    event: "controlPlaneEvent".to_string(),
                    data: ignored,
                    timestamp: 0,
                })
                .await;
        }
        tokio::time::sleep(Duration::from_millis(100)).await;
        assert_eq!(
            requests.load(Ordering::SeqCst),
            0,
            "device and filter mismatches must not run the workflow"
        );

        let event = json!({
            "id": "evt-denied-1",
            "type": "device.access.denied",
            "server_rev": 42,
            "data": {"device_id": "device-1", "reason": "policy"}
        });
        ws.event_bus()
            .publish(crate::event_bus::Event {
                event: "controlPlaneEvent".to_string(),
                data: event.clone(),
                timestamp: 1,
            })
            .await;

        let completed = tokio::time::timeout(Duration::from_secs(2), async {
            loop {
                if trigger_store
                    .get(&trigger.id)
                    .is_some_and(|current| current.last_status == "completed")
                {
                    break;
                }
                tokio::time::sleep(Duration::from_millis(20)).await;
            }
        })
        .await;
        assert!(completed.is_ok(), "event trigger did not complete");
        assert_eq!(requests.load(Ordering::SeqCst), 1);

        ws.event_bus()
            .publish(crate::event_bus::Event {
                event: "controlPlaneEvent".to_string(),
                data: event,
                timestamp: 2,
            })
            .await;
        tokio::time::sleep(Duration::from_millis(150)).await;
        assert_eq!(
            requests.load(Ordering::SeqCst),
            1,
            "duplicate server_rev must not execute twice"
        );

        ws.event_bus()
            .publish(crate::event_bus::Event {
                event: "controlPlaneEvent".to_string(),
                data: json!({
                    "id": "evt-task-failed-1",
                    "type": "scheduled_task.failed",
                    "server_rev": 43,
                    "data": {
                        "task_id": 7,
                        "kind": "sign_out_all_sessions",
                        "status": "failed"
                    }
                }),
                timestamp: 3,
            })
            .await;
        let task_completed = tokio::time::timeout(Duration::from_secs(2), async {
            loop {
                if trigger_store
                    .get(&task_trigger.id)
                    .is_some_and(|current| current.last_status == "completed")
                {
                    break;
                }
                tokio::time::sleep(Duration::from_millis(20)).await;
            }
        })
        .await;
        assert!(
            task_completed.is_ok(),
            "non-device event trigger did not complete"
        );
        assert_eq!(requests.load(Ordering::SeqCst), 2);

        cancellation.cancel();
        fake_quickdesk.abort();
        let _ = std::fs::remove_dir_all(temp);
    }
}
