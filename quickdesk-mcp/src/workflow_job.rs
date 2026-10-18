use std::collections::HashMap;
use std::path::PathBuf;
use std::sync::{Arc, Mutex};

use chrono::Utc;
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use tokio::sync::{mpsc, Semaphore};
use tokio_util::sync::CancellationToken;
use uuid::Uuid;

use crate::callback::{
    CallbackAckError, CallbackBatchAckResult, CallbackDelivery, CallbackDeliveryStatus,
    CallbackOutbox, CallbackRetryError,
};
use crate::server::{QuickDeskMcpServer, SharedState};
use crate::workflow::WorkflowRunReport;
use crate::ws_client::WsClient;

const QUEUE_CAPACITY: usize = 32;

#[derive(Clone, Debug, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum WorkflowJobStatus {
    Queued,
    Running,
    Cancelling,
    Completed,
    Failed,
    Cancelled,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct WorkflowRunJob {
    pub run_id: String,
    pub workflow_id: String,
    pub device_id: String,
    pub status: WorkflowJobStatus,
    pub created_at: String,
    pub started_at: Option<String>,
    pub completed_at: Option<String>,
    pub report: Option<WorkflowRunReport>,
    pub error: String,
    #[serde(default)]
    pub callback_delivery_id: Option<String>,
    #[serde(default)]
    pub callback_status: String,
    #[serde(default)]
    pub callback_error: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
struct WorkflowJobRequestSpec {
    #[serde(default)]
    overrides: HashMap<String, Value>,
    #[serde(default = "default_stop_on_failure")]
    stop_on_failure: bool,
    #[serde(default)]
    callback_url: Option<String>,
}

impl Default for WorkflowJobRequestSpec {
    fn default() -> Self {
        Self {
            overrides: HashMap::new(),
            stop_on_failure: true,
            callback_url: None,
        }
    }
}

fn default_stop_on_failure() -> bool {
    true
}

#[derive(Clone)]
pub struct WorkflowJobStore {
    storage_dir: PathBuf,
}

impl WorkflowJobStore {
    pub fn new() -> Result<Self, String> {
        let base = dirs_data_dir().unwrap_or_else(|| PathBuf::from("."));
        Self::from_storage_dir(base.join(".quickdesk").join("workflow-jobs"))
    }

    fn from_storage_dir(storage_dir: PathBuf) -> Result<Self, String> {
        std::fs::create_dir_all(&storage_dir)
            .map_err(|error| format!("cannot create workflow job dir: {error}"))?;
        Ok(Self { storage_dir })
    }

    #[cfg(test)]
    pub(crate) fn with_storage_dir(storage_dir: PathBuf) -> Self {
        Self::from_storage_dir(storage_dir).expect("test workflow job store")
    }

    fn create(
        &self,
        workflow_id: &str,
        device_id: &str,
        request: &WorkflowJobRequestSpec,
    ) -> Result<WorkflowRunJob, String> {
        let now = Utc::now().to_rfc3339();
        let job = WorkflowRunJob {
            run_id: Uuid::new_v4().to_string(),
            workflow_id: workflow_id.to_string(),
            device_id: device_id.to_string(),
            status: WorkflowJobStatus::Queued,
            created_at: now.clone(),
            started_at: None,
            completed_at: None,
            report: None,
            error: String::new(),
            callback_delivery_id: None,
            callback_status: String::new(),
            callback_error: String::new(),
        };
        self.save_request(&job.run_id, request)?;
        self.save(&job)?;
        Ok(job)
    }

    pub fn get(&self, run_id: &str) -> Option<WorkflowRunJob> {
        let path = self.job_path(run_id)?;
        let data = std::fs::read_to_string(path).ok()?;
        serde_json::from_str(&data).ok()
    }

    fn get_request(&self, run_id: &str) -> Option<WorkflowJobRequestSpec> {
        let path = self.request_path(run_id)?;
        let data = std::fs::read_to_string(path).ok()?;
        serde_json::from_str(&data).ok()
    }

    fn list(&self) -> Result<Vec<WorkflowRunJob>, String> {
        let entries = std::fs::read_dir(&self.storage_dir)
            .map_err(|error| format!("read workflow job dir: {error}"))?;
        let mut jobs: Vec<WorkflowRunJob> = Vec::new();
        for entry in entries {
            let entry = entry.map_err(|error| format!("read workflow job entry: {error}"))?;
            let path = entry.path();
            if path.extension().and_then(|value| value.to_str()) != Some("json")
                || path
                    .file_name()
                    .and_then(|value| value.to_str())
                    .is_some_and(|name| name.ends_with(".request.json"))
            {
                continue;
            }
            let data = match std::fs::read_to_string(&path) {
                Ok(data) => data,
                Err(error) => {
                    tracing::warn!("cannot read workflow job {}: {error}", path.display());
                    continue;
                }
            };
            match serde_json::from_str(&data) {
                Ok(job) => jobs.push(job),
                Err(error) => {
                    tracing::warn!("cannot parse workflow job {}: {error}", path.display());
                }
            }
        }
        jobs.sort_by(|left, right| left.created_at.cmp(&right.created_at));
        Ok(jobs)
    }

    pub fn mark_running(&self, run_id: &str) -> Result<(), String> {
        self.update(run_id, |job| {
            job.status = WorkflowJobStatus::Running;
            job.started_at = Some(Utc::now().to_rfc3339());
        })
    }

    pub fn mark_cancelling(&self, run_id: &str) -> Result<(), String> {
        self.update(run_id, |job| {
            if matches!(
                job.status,
                WorkflowJobStatus::Queued | WorkflowJobStatus::Running
            ) {
                job.status = WorkflowJobStatus::Cancelling;
            }
        })
    }

    pub fn mark_cancelled(&self, run_id: &str, error: &str) -> Result<(), String> {
        self.update(run_id, |job| {
            job.status = WorkflowJobStatus::Cancelled;
            job.completed_at = Some(Utc::now().to_rfc3339());
            job.error = error.to_string();
        })
    }

    pub fn finish(
        &self,
        run_id: &str,
        status: WorkflowJobStatus,
        report: Option<WorkflowRunReport>,
        error: &str,
    ) -> Result<(), String> {
        self.update(run_id, |job| {
            job.status = status;
            job.completed_at = Some(Utc::now().to_rfc3339());
            job.report = report;
            job.error = error.to_string();
        })
    }

    fn set_callback(
        &self,
        run_id: &str,
        delivery_id: Option<String>,
        status: &str,
        error: &str,
    ) -> Result<(), String> {
        self.update(run_id, |job| {
            job.callback_delivery_id = delivery_id;
            job.callback_status = status.to_string();
            job.callback_error = error.to_string();
        })
    }

    fn update<F>(&self, run_id: &str, apply: F) -> Result<(), String>
    where
        F: FnOnce(&mut WorkflowRunJob),
    {
        let mut job = self
            .get(run_id)
            .ok_or_else(|| "workflow job not found".to_string())?;
        apply(&mut job);
        self.save(&job)
    }

    fn save(&self, job: &WorkflowRunJob) -> Result<(), String> {
        let path = self
            .job_path(&job.run_id)
            .ok_or_else(|| "invalid workflow job id".to_string())?;
        let data = serde_json::to_string_pretty(job)
            .map_err(|error| format!("serialize workflow job: {error}"))?;
        std::fs::write(path, data).map_err(|error| format!("write workflow job: {error}"))
    }

    fn save_request(&self, run_id: &str, request: &WorkflowJobRequestSpec) -> Result<(), String> {
        let path = self
            .request_path(run_id)
            .ok_or_else(|| "invalid workflow job id".to_string())?;
        let data = serde_json::to_string_pretty(request)
            .map_err(|error| format!("serialize workflow job request: {error}"))?;
        std::fs::write(path, data).map_err(|error| format!("write workflow job request: {error}"))
    }

    fn job_path(&self, run_id: &str) -> Option<PathBuf> {
        Uuid::parse_str(run_id).ok()?;
        Some(self.storage_dir.join(format!("{run_id}.json")))
    }

    fn request_path(&self, run_id: &str) -> Option<PathBuf> {
        Uuid::parse_str(run_id).ok()?;
        Some(self.storage_dir.join(format!("{run_id}.request.json")))
    }
}

#[derive(Clone)]
pub struct WorkflowJobManager {
    store: WorkflowJobStore,
    sender: mpsc::Sender<WorkflowJobRequest>,
    active: Arc<Mutex<HashMap<String, CancellationToken>>>,
    callbacks: CallbackOutbox,
}

#[derive(Clone)]
struct WorkflowJobRequest {
    run_id: String,
    workflow_id: String,
    device_id: String,
    overrides: HashMap<String, Value>,
    stop_on_failure: bool,
    callback_url: Option<String>,
}

#[derive(Debug)]
pub enum CancelJobError {
    NotFound,
    NotCancellable,
    Store(String),
}

#[derive(Clone)]
struct WorkerContext {
    store: WorkflowJobStore,
    active: Arc<Mutex<HashMap<String, CancellationToken>>>,
    callbacks: CallbackOutbox,
    ws: WsClient,
    shared_state: Arc<SharedState>,
    allowed_devices: Vec<String>,
    run_slots: Arc<Semaphore>,
}

impl WorkflowJobManager {
    pub fn new(
        cancellation: CancellationToken,
        ws: WsClient,
        shared_state: Arc<SharedState>,
        allowed_devices: Vec<String>,
        run_slots: Arc<Semaphore>,
    ) -> Result<Self, String> {
        let callbacks = CallbackOutbox::new(cancellation.child_token())?;
        Self::with_store_and_callbacks(
            WorkflowJobStore::new()?,
            cancellation,
            ws,
            shared_state,
            allowed_devices,
            run_slots,
            callbacks,
        )
    }

    pub(crate) fn with_store_and_callbacks(
        store: WorkflowJobStore,
        cancellation: CancellationToken,
        ws: WsClient,
        shared_state: Arc<SharedState>,
        allowed_devices: Vec<String>,
        run_slots: Arc<Semaphore>,
        callbacks: CallbackOutbox,
    ) -> Result<Self, String> {
        let (sender, receiver) = mpsc::channel(QUEUE_CAPACITY);
        let manager = Self {
            store,
            sender,
            active: Arc::new(Mutex::new(HashMap::new())),
            callbacks: callbacks.clone(),
        };
        start_worker(
            receiver,
            cancellation,
            WorkerContext {
                store: manager.store.clone(),
                active: manager.active.clone(),
                callbacks,
                ws,
                shared_state,
                allowed_devices,
                run_slots,
            },
        );
        manager.recover_persisted_jobs()?;
        Ok(manager)
    }

    pub fn enqueue_callback(
        &self,
        url: Option<&str>,
        event: Value,
    ) -> Result<Option<String>, String> {
        self.callbacks.enqueue(url, event)
    }

    pub fn list_callback_deliveries(
        &self,
        status: Option<CallbackDeliveryStatus>,
        limit: usize,
    ) -> Result<Vec<CallbackDelivery>, String> {
        self.callbacks.list(status, limit)
    }

    pub fn retry_callback_delivery(
        &self,
        delivery_id: &str,
    ) -> Result<CallbackDelivery, CallbackRetryError> {
        self.callbacks.retry_dead(delivery_id)
    }

    pub fn acknowledge_callback_delivery(
        &self,
        delivery_id: &str,
    ) -> Result<CallbackDelivery, CallbackAckError> {
        self.callbacks.acknowledge(delivery_id)
    }

    pub fn acknowledge_callback_deliveries(
        &self,
        delivery_ids: &[String],
    ) -> Result<CallbackBatchAckResult, String> {
        self.callbacks.acknowledge_many(delivery_ids)
    }

    pub fn enqueue(
        &self,
        workflow_id: &str,
        device_id: &str,
        overrides: HashMap<String, Value>,
        stop_on_failure: bool,
        callback_url: Option<String>,
    ) -> Result<WorkflowRunJob, String> {
        let spec = WorkflowJobRequestSpec {
            overrides,
            stop_on_failure,
            callback_url,
        };
        let job = self.store.create(workflow_id, device_id, &spec)?;
        let token = CancellationToken::new();
        self.active
            .lock()
            .map_err(|_| "workflow job lock poisoned".to_string())?
            .insert(job.run_id.clone(), token);
        let request = WorkflowJobRequest {
            run_id: job.run_id.clone(),
            workflow_id: workflow_id.to_string(),
            device_id: device_id.to_string(),
            overrides: spec.overrides,
            stop_on_failure: spec.stop_on_failure,
            callback_url: spec.callback_url,
        };
        if let Err(error) = self.sender.try_send(request) {
            self.active.lock().unwrap().remove(&job.run_id);
            let message = format!("workflow job queue is full: {error}");
            let _ = self
                .store
                .finish(&job.run_id, WorkflowJobStatus::Failed, None, &message);
            return Err(message);
        }
        Ok(job)
    }

    fn recover_persisted_jobs(&self) -> Result<(), String> {
        for job in self.store.list()? {
            match job.status {
                WorkflowJobStatus::Queued => {
                    let spec = self.store.get_request(&job.run_id).unwrap_or_default();
                    let token = CancellationToken::new();
                    self.active
                        .lock()
                        .map_err(|_| "workflow job lock poisoned".to_string())?
                        .insert(job.run_id.clone(), token);
                    let request = WorkflowJobRequest {
                        run_id: job.run_id.clone(),
                        workflow_id: job.workflow_id,
                        device_id: job.device_id,
                        overrides: spec.overrides,
                        stop_on_failure: spec.stop_on_failure,
                        callback_url: spec.callback_url,
                    };
                    if let Err(error) = self.sender.try_send(request) {
                        self.active.lock().unwrap().remove(&job.run_id);
                        let _ = self.store.finish(
                            &job.run_id,
                            WorkflowJobStatus::Failed,
                            None,
                            &format!("workflow job recovery queue is full: {error}"),
                        );
                    }
                }
                WorkflowJobStatus::Running => {
                    self.store.finish(
                        &job.run_id,
                        WorkflowJobStatus::Failed,
                        None,
                        "workflow job was interrupted by process restart; side effects are unknown",
                    )?;
                }
                WorkflowJobStatus::Cancelling => {
                    self.store.mark_cancelled(
                        &job.run_id,
                        "cancellation completed during process restart; no further steps were run",
                    )?;
                }
                WorkflowJobStatus::Completed
                | WorkflowJobStatus::Failed
                | WorkflowJobStatus::Cancelled => {}
            }
        }
        Ok(())
    }

    pub fn get(&self, run_id: &str) -> Option<WorkflowRunJob> {
        self.store.get(run_id)
    }

    pub fn cancel(&self, run_id: &str) -> Result<WorkflowRunJob, CancelJobError> {
        let job = self.store.get(run_id).ok_or(CancelJobError::NotFound)?;
        match job.status {
            WorkflowJobStatus::Queued => {
                self.store
                    .mark_cancelled(run_id, "cancelled before execution")
                    .map_err(CancelJobError::Store)?;
                if let Some(token) = self.active.lock().unwrap().get(run_id) {
                    token.cancel();
                }
            }
            WorkflowJobStatus::Running => {
                self.store
                    .mark_cancelling(run_id)
                    .map_err(CancelJobError::Store)?;
                if let Some(token) = self.active.lock().unwrap().get(run_id) {
                    token.cancel();
                }
            }
            _ => return Err(CancelJobError::NotCancellable),
        }
        self.store.get(run_id).ok_or(CancelJobError::NotFound)
    }
}

fn start_worker(
    mut receiver: mpsc::Receiver<WorkflowJobRequest>,
    cancellation: CancellationToken,
    context: WorkerContext,
) {
    tokio::spawn(async move {
        loop {
            let request = tokio::select! {
                _ = cancellation.cancelled() => break,
                request = receiver.recv() => match request {
                    Some(request) => request,
                    None => break,
                },
            };
            let token = context
                .active
                .lock()
                .ok()
                .and_then(|active| active.get(&request.run_id).cloned())
                .unwrap_or_default();
            if token.is_cancelled() {
                let _ = context
                    .store
                    .mark_cancelled(&request.run_id, "cancelled before execution");
                context
                    .active
                    .lock()
                    .ok()
                    .map(|mut map| map.remove(&request.run_id));
                continue;
            }
            let permit = tokio::select! {
                _ = token.cancelled() => {
                    let _ = context
                        .store
                        .mark_cancelled(&request.run_id, "cancelled while queued");
                    context
                        .active
                        .lock()
                        .ok()
                        .map(|mut map| map.remove(&request.run_id));
                    continue;
                }
                permit = context.run_slots.clone().acquire_owned() => permit,
            };
            if permit.is_err() {
                let _ = context.store.finish(
                    &request.run_id,
                    WorkflowJobStatus::Failed,
                    None,
                    "workflow execution slot was closed",
                );
                context
                    .active
                    .lock()
                    .ok()
                    .map(|mut map| map.remove(&request.run_id));
                continue;
            }
            if token.is_cancelled() {
                let _ = context
                    .store
                    .mark_cancelled(&request.run_id, "cancelled before execution");
                context
                    .active
                    .lock()
                    .ok()
                    .map(|mut map| map.remove(&request.run_id));
                continue;
            }
            if context.store.mark_running(&request.run_id).is_err() {
                context
                    .active
                    .lock()
                    .ok()
                    .map(|mut map| map.remove(&request.run_id));
                continue;
            }
            let server = QuickDeskMcpServer::new(
                context.ws.clone(),
                context.allowed_devices.clone(),
                context.shared_state.clone(),
            );
            let result = server
                .execute_workflow_replay_cancellable(
                    &request.workflow_id,
                    &request.device_id,
                    request.overrides,
                    request.stop_on_failure,
                    request.run_id.clone(),
                    token.clone(),
                )
                .await;
            match result {
                Ok(report) => {
                    let callback_result = context.callbacks.enqueue(
                        request.callback_url.as_deref(),
                        json!({
                            "event": "workflow.run.completed",
                            "run_id": report.run_id.clone(),
                            "workflow_id": report.workflow_id.clone(),
                            "device_id": report.device_id.clone(),
                            "success": report.success,
                            "cancelled": report.cancelled,
                            "report": report.clone(),
                        }),
                    );
                    let status = if report.cancelled {
                        WorkflowJobStatus::Cancelled
                    } else if report.success {
                        WorkflowJobStatus::Completed
                    } else {
                        WorkflowJobStatus::Failed
                    };
                    let _ = context
                        .store
                        .finish(&request.run_id, status, Some(report), "");
                    record_callback_result(&context.store, &request.run_id, callback_result);
                }
                Err(error) => {
                    let callback_result = context.callbacks.enqueue(
                        request.callback_url.as_deref(),
                        json!({
                            "event": "workflow.run.failed",
                            "workflow_id": request.workflow_id.clone(),
                            "device_id": request.device_id.clone(),
                            "error": error.clone(),
                        }),
                    );
                    let _ = context.store.finish(
                        &request.run_id,
                        WorkflowJobStatus::Failed,
                        None,
                        &error,
                    );
                    record_callback_result(&context.store, &request.run_id, callback_result);
                }
            }
            context
                .active
                .lock()
                .ok()
                .map(|mut map| map.remove(&request.run_id));
            drop(permit);
        }
    });
}

fn record_callback_result(
    store: &WorkflowJobStore,
    run_id: &str,
    result: Result<Option<String>, String>,
) {
    match result {
        Ok(Some(delivery_id)) => {
            let _ = store.set_callback(run_id, Some(delivery_id), "queued", "");
        }
        Ok(None) => {
            let _ = store.set_callback(run_id, None, "not_requested", "");
        }
        Err(error) => {
            tracing::warn!("cannot enqueue workflow callback for {}: {error}", run_id);
            let _ = store.set_callback(run_id, None, "failed", &error);
        }
    }
}

fn dirs_data_dir() -> Option<PathBuf> {
    #[cfg(target_os = "windows")]
    {
        std::env::var("USERPROFILE").ok().map(PathBuf::from)
    }
    #[cfg(not(target_os = "windows"))]
    {
        std::env::var("HOME").ok().map(PathBuf::from)
    }
}
