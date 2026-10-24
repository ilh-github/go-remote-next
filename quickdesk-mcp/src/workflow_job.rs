use std::collections::HashMap;
use std::fs::OpenOptions;
use std::path::PathBuf;
use std::sync::{Arc, Mutex};
use std::time::{Duration, SystemTime};

use chrono::Utc;
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use tokio::sync::{Notify, Semaphore};
use tokio_util::sync::CancellationToken;
use uuid::Uuid;

use crate::callback::{
    CallbackAckActor, CallbackAckError, CallbackBatchAckResult, CallbackDelivery,
    CallbackDeliveryStatus, CallbackOutbox, CallbackRetryError,
};
use crate::server::{QuickDeskMcpServer, SharedState};
use crate::fair_queue::FairQueue;
use crate::distributed_claim::{ClaimLease, DistributedClaimStore};
#[cfg(test)]
use crate::distributed_claim::CLAIM_STALE_AFTER_SECONDS;
use crate::workflow::WorkflowRunReport;
use crate::ws_client::WsClient;

/// Bound on queued work per submitter. One submitter can hold the single
/// execution slot with a long backlog, but it must not be able to pin an
/// unbounded amount of other submitters' turns behind its own either.
const QUEUE_CAPACITY_PER_SUBMITTER: usize = 8;
/// Total bound across every submitter, so a fleet of busy callers cannot
/// grow the queue without limit.
const QUEUE_CAPACITY_TOTAL: usize = 256;
const JOB_CLAIM_STALE_AFTER: Duration = Duration::from_secs(30);

/// Owner key used when a caller does not identify itself. Every anonymous
/// submitter shares one lane, so a single anonymous flood cannot starve the
/// identified callers either - it just competes as one submitter.
const ANONYMOUS_SUBMITTER: &str = "anonymous";

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
    /// Directories other machines submit into. A worker that shares no
    /// filesystem with its peers can leave this empty; a fleet on shared
    /// storage lists the peers so a run is picked up whichever machine it was
    /// submitted on.
    peer_dirs: Arc<Mutex<Vec<PathBuf>>>,
}

struct WorkflowJobExecutionClaim {
    path: PathBuf,
}

impl Drop for WorkflowJobExecutionClaim {
    fn drop(&mut self) {
        if let Err(error) = std::fs::remove_file(&self.path) {
            if error.kind() != std::io::ErrorKind::NotFound {
                tracing::warn!(
                    "cannot release workflow job claim {}: {error}",
                    self.path.display()
                );
            }
        }
    }
}

impl WorkflowJobStore {
    pub fn new() -> Result<Self, String> {
        let base = dirs_data_dir().unwrap_or_else(|| PathBuf::from("."));
        Self::from_storage_dir(base.join(".quickdesk").join("workflow-jobs"))
    }

    fn from_storage_dir(storage_dir: PathBuf) -> Result<Self, String> {
        std::fs::create_dir_all(&storage_dir)
            .map_err(|error| format!("cannot create workflow job dir: {error}"))?;
        Ok(Self {
            storage_dir,
            peer_dirs: Arc::new(Mutex::new(Vec::new())),
        })
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
        let path = self.job_path_anywhere(run_id)?;
        let data = std::fs::read_to_string(path).ok()?;
        serde_json::from_str(&data).ok()
    }

    fn get_request(&self, run_id: &str) -> Option<WorkflowJobRequestSpec> {
        // The request file sits beside the job file, so it has to be looked up
        // in the same directory the run was found in.
        let path = self.request_path_anywhere(run_id)?;
        let data = std::fs::read_to_string(path).ok()?;
        serde_json::from_str(&data).ok()
    }

    /// Every run this worker can see, in any state, oldest first.
    ///
    /// Covers the peer directories as well, because restart recovery has to
    /// reconcile runs that were submitted on another machine: a `Running` row
    /// from a peer whose claim has lapsed is exactly the run this worker should
    /// take over rather than leave stuck.
    fn list(&self) -> Result<Vec<WorkflowRunJob>, String> {
        let peers = match self.peer_dirs.lock() {
            Ok(peers) => peers.clone(),
            Err(poisoned) => poisoned.into_inner().clone(),
        };
        let mut seen: std::collections::HashSet<String> = std::collections::HashSet::new();
        let mut jobs: Vec<WorkflowRunJob> = Vec::new();
        for job in scan_job_dir(&self.storage_dir)
            .into_iter()
            .chain(peers.iter().flat_map(|dir| scan_job_dir(dir)))
        {
            if seen.insert(job.run_id.clone()) {
                jobs.push(job);
            }
        }
        jobs.sort_by(|left, right| {
            left.created_at
                .cmp(&right.created_at)
                .then_with(|| left.run_id.cmp(&right.run_id))
        });
        Ok(jobs)
    }

    /// Adds a directory another machine submits into, so this worker can see
    /// runs it did not create.
    ///
    /// This is the piece that turns a set of independent single-machine workers
    /// into a fleet: without it a run lives in the queue of whichever process
    /// accepted it, and that process being down strands the run. The shared
    /// claim store still decides who actually executes it, so watching a peer
    /// makes the work visible, not duplicated.
    pub fn watch_peer_dir(&self, dir: impl AsRef<std::path::Path>) {
        let dir = dir.as_ref().to_path_buf();
        let mut peers = match self.peer_dirs.lock() {
            Ok(peers) => peers,
            Err(poisoned) => poisoned.into_inner(),
        };
        // The same directory can be configured twice (a ring of peers, or a
        // restart that re-adds it), and listing it twice would show its runs
        // twice.
        if !peers.contains(&dir) {
            peers.push(dir);
        }
    }

    pub fn mark_running(&self, run_id: &str) -> Result<(), String> {
        self.update(run_id, |job| {
            job.status = WorkflowJobStatus::Running;
            job.started_at = Some(Utc::now().to_rfc3339());
        })
    }

    /// Takes the execution claim for a run.
    ///
    /// The local-file claim is checked first because it is the one that has
    /// always protected same-machine workers, and it is cheap. The distributed
    /// claim is then required on top of it when one is configured: without it
    /// two workers on two machines would each open their own claim file - each
    /// is atomic on its own host and invisible to the other - and both would
    /// drive the same remote desktop.
    ///
    /// `worker_id` identifies this process to the shared store, so a claim lost
    /// to a crash can be told apart from one this worker still holds.
    fn claim_execution_for(
        &self,
        run_id: &str,
        worker_id: &str,
        distributed: Option<&DistributedClaimStore>,
    ) -> Option<WorkflowJobExecutionClaim> {
        let claim = self.acquire_execution_claim(run_id)?;
        let job = self.get(run_id)?;
        if job.status != WorkflowJobStatus::Queued {
            return None;
        }
        // The distributed claim is taken only after the local one and the status
        // check, so a run that is not claimable never occupies a row in the
        // shared database.
        if let Some(store) = distributed {
            if !store.is_enabled() {
                return Some(claim);
            }
            match store.try_claim(run_id, worker_id) {
                Ok(true) => {}
                // Another machine holds it, or the shared store is unreachable.
                // Either way this worker must not drive the run.
                Ok(false) => return None,
                Err(error) => {
                    tracing::warn!(
                        "cannot claim workflow job {} in the shared store: {error}",
                        run_id
                    );
                    return None;
                }
            }
        }
        Some(claim)
    }

    fn acquire_execution_claim(&self, run_id: &str) -> Option<WorkflowJobExecutionClaim> {
        let path = self.claim_path(run_id)?;
        match OpenOptions::new().write(true).create_new(true).open(&path) {
            Ok(_) => Some(WorkflowJobExecutionClaim { path }),
            Err(error) if error.kind() == std::io::ErrorKind::AlreadyExists => {
                if self.claim_is_stale(&path) {
                    if let Err(remove_error) = std::fs::remove_file(&path) {
                        if remove_error.kind() != std::io::ErrorKind::NotFound {
                            tracing::warn!(
                                "cannot remove stale workflow job claim {}: {remove_error}",
                                path.display()
                            );
                            return None;
                        }
                    }
                    return match OpenOptions::new().write(true).create_new(true).open(&path) {
                        Ok(_) => Some(WorkflowJobExecutionClaim { path }),
                        Err(_) => None,
                    };
                }
                None
            }
            Err(error) => {
                tracing::warn!("cannot claim workflow job {}: {error}", path.display());
                None
            }
        }
    }

    fn execution_claim_is_active(&self, run_id: &str) -> bool {
        self.claim_path(run_id)
            .is_some_and(|path| path.exists() && !self.claim_is_stale(&path))
    }

    fn claim_is_stale(&self, path: &std::path::Path) -> bool {
        std::fs::metadata(path)
            .and_then(|metadata| metadata.modified())
            .ok()
            .and_then(|modified| SystemTime::now().duration_since(modified).ok())
            .is_some_and(|age| age >= JOB_CLAIM_STALE_AFTER)
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

    /// The job file for a run, wherever it lives.
    ///
    /// A run this worker did not submit only exists in a peer directory, so
    /// resolving it against `storage_dir` alone would read nothing and the
    /// claim would be dropped as "not queued".
    fn job_path_anywhere(&self, run_id: &str) -> Option<PathBuf> {
        let local = self.job_path(run_id)?;
        if local.exists() {
            return Some(local);
        }
        let peers = match self.peer_dirs.lock() {
            Ok(peers) => peers.clone(),
            Err(poisoned) => poisoned.into_inner().clone(),
        };
        peers
            .into_iter()
            .map(|dir| dir.join(format!("{run_id}.json")))
            .find(|path| path.exists())
    }

    fn request_path(&self, run_id: &str) -> Option<PathBuf> {
        Uuid::parse_str(run_id).ok()?;
        Some(self.storage_dir.join(format!("{run_id}.request.json")))
    }

    fn request_path_anywhere(&self, run_id: &str) -> Option<PathBuf> {
        let local = self.request_path(run_id)?;
        if local.exists() {
            return Some(local);
        }
        let peers = match self.peer_dirs.lock() {
            Ok(peers) => peers.clone(),
            Err(poisoned) => poisoned.into_inner().clone(),
        };
        peers
            .into_iter()
            .map(|dir| dir.join(format!("{run_id}.request.json")))
            .find(|path| path.exists())
    }

    fn claim_path(&self, run_id: &str) -> Option<PathBuf> {
        Uuid::parse_str(run_id).ok()?;
        Some(self.storage_dir.join(format!("{run_id}.claim")))
    }
}

/// Reads every job file in one directory, oldest first.
///
/// A directory that does not exist or cannot be read yields nothing rather than
/// an error: a peer that is simply not mounted yet must not stop this worker
/// from serving the runs it can see.
fn scan_job_dir(dir: &std::path::Path) -> Vec<WorkflowRunJob> {
    let entries = match std::fs::read_dir(dir) {
        Ok(entries) => entries,
        Err(error) => {
            tracing::debug!("cannot read workflow job dir {}: {error}", dir.display());
            return Vec::new();
        }
    };
    let mut jobs: Vec<WorkflowRunJob> = Vec::new();
    for entry in entries {
        let entry = match entry {
            Ok(entry) => entry,
            Err(error) => {
                tracing::warn!("cannot read workflow job entry: {error}");
                continue;
            }
        };
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
    jobs
}

/// Turns the comma-separated `QUICKDESK_WORKFLOW_PEER_JOB_DIRS` value into
/// directories.
///
/// Blank entries are dropped rather than turned into empty paths: a trailing
/// comma is a normal thing to type, and an empty path would resolve to the
/// current directory, which is never what the operator meant.
pub(crate) fn parse_peer_dirs(list: &str) -> Vec<PathBuf> {
    list.split(',')
        .map(str::trim)
        .filter(|entry| !entry.is_empty())
        .map(PathBuf::from)
        .collect()
}

#[derive(Clone)]
pub struct WorkflowJobManager {
    store: WorkflowJobStore,
    /// Shared execution claims, when this deployment runs workers on more than
    /// one machine. `None` is the single-machine case and keeps the local file
    /// claim as the only guard.
    distributed_claims: Option<DistributedClaimStore>,
    /// How this process identifies itself in the shared claim store.
    worker_id: String,
    /// Fair, round-robin queue of pending work. The single execution slot
    /// makes ordering the fairness question: a plain FIFO lets the fastest
    /// submitter push everyone else's runs behind an unbounded number of its
    /// own.
    queue: Arc<Mutex<FairQueue<WorkflowJobRequest>>>,
    /// Wakes the worker when new work is pushed. `notify_one` is enough: the
    /// worker drains the queue until it is empty before waiting again.
    work_ready: Arc<Notify>,
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
    /// Tools the authorising grant denies. Carried with the job because the
    /// grant is resolved at enqueue time and the worker runs later.
    denied_tools: Vec<String>,
    /// Submitter identity, used as the fair-queue lane key. `None` (or an
    /// empty string) means the caller did not identify itself.
    submitter: Option<String>,
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
    /// Shared execution claims. `None` keeps the single-machine behaviour that
    /// shipped first; a configured store is what makes two workers on two
    /// machines unable to drive the same run.
    distributed_claims: Option<DistributedClaimStore>,
    /// Identifies this process in the shared store, so a claim held by a crashed
    /// worker can be told apart from one this worker still owns.
    worker_id: String,
    queue: Arc<Mutex<FairQueue<WorkflowJobRequest>>>,
    work_ready: Arc<Notify>,
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

    /// Attaches a shared claim store so two workers on two machines cannot both
    /// drive the same queued run.
    ///
    /// A setter rather than another constructor argument: the single-machine
    /// deployment never calls it, and the field it sets has to reach the worker
    /// context that is already built by `with_store_and_callbacks`.
    pub fn with_distributed_claims(mut self, claims: DistributedClaimStore, worker_id: String) -> Self {
        self.distributed_claims = Some(claims);
        self.worker_id = worker_id;
        self
    }

    /// Names the directories other machines submit workflow runs into, so this
    /// worker picks up runs it did not accept itself.
    ///
    /// Pairs with `with_distributed_claims`: watching a peer makes the run
    /// visible, the shared claim decides which machine executes it. Setting the
    /// peers without a shared claim store would let every machine that can see
    /// the directory drive the run, so the claim is what makes this safe rather
    /// than merely load-balanced.
    pub fn with_peer_job_dirs(self, dirs: Vec<PathBuf>) -> Self {
        for dir in dirs {
            self.store.watch_peer_dir(dir);
        }
        self
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
        let queue = Arc::new(Mutex::new(FairQueue::new()));
        let work_ready = Arc::new(Notify::new());
        let manager = Self {
            store,
            distributed_claims: None,
            worker_id: String::new(),
            queue: queue.clone(),
            work_ready: work_ready.clone(),
            active: Arc::new(Mutex::new(HashMap::new())),
            callbacks: callbacks.clone(),
        };
        start_worker(
            cancellation,
            WorkerContext {
                store: manager.store.clone(),
                distributed_claims: manager.distributed_claims.clone(),
                worker_id: manager.worker_id.clone(),
                queue,
                work_ready,
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
        actor: &CallbackAckActor,
    ) -> Result<CallbackDelivery, CallbackAckError> {
        self.callbacks.acknowledge(delivery_id, actor)
    }

    pub fn acknowledge_callback_deliveries(
        &self,
        delivery_ids: &[String],
        actor: &CallbackAckActor,
    ) -> Result<CallbackBatchAckResult, String> {
        self.callbacks.acknowledge_many(delivery_ids, actor)
    }

    // The job's own request plus the fair-queue lane key. Both are flat
    // values the caller already holds.
    #[allow(clippy::too_many_arguments)]
    pub fn enqueue(
        &self,
        workflow_id: &str,
        device_id: &str,
        overrides: HashMap<String, Value>,
        stop_on_failure: bool,
        callback_url: Option<String>,
        denied_tools: Vec<String>,
        submitter: Option<&str>,
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
            denied_tools,
            submitter: Some(normalize_submitter(submitter)),
        };
        let lane_key = request.submitter.clone().unwrap_or_default();
        let rejection = {
            let queue = self
                .queue
                .lock()
                .map_err(|_| "workflow job queue poisoned".to_string())?;
            Self::check_queue_capacity(&queue, &lane_key)
        };
        if let Some(message) = rejection {
            self.active.lock().unwrap().remove(&job.run_id);
            let _ = self
                .store
                .finish(&job.run_id, WorkflowJobStatus::Failed, None, &message);
            return Err(message);
        }
        {
            let mut queue = self
                .queue
                .lock()
                .map_err(|_| "workflow job queue poisoned".to_string())?;
            queue.push(&lane_key, request);
        }
        self.work_ready.notify_one();
        Ok(job)
    }

    /// Capacity is checked per submitter and in total. The per-submitter
    /// bound is the fairness rule: a caller that floods the queue is
    /// rejected rather than being allowed to hold everyone else's turn.
    /// `trigger.rs` maps the "queue is full" wording onto 429, so the
    /// message keeps that phrase for both cases.
    fn check_queue_capacity(
        queue: &FairQueue<WorkflowJobRequest>,
        submitter: &str,
    ) -> Option<String> {
        if queue.len() >= QUEUE_CAPACITY_TOTAL {
            return Some(format!(
                "workflow job queue is full: {QUEUE_CAPACITY_TOTAL} jobs are already waiting"
            ));
        }
        let waiting = queue.waiting_for(submitter);
        if waiting >= QUEUE_CAPACITY_PER_SUBMITTER {
            return Some(format!(
                "workflow job queue is full: {waiting} jobs from this caller are already waiting (limit {QUEUE_CAPACITY_PER_SUBMITTER})"
            ));
        }
        None
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
                        denied_tools: Vec::new(),
                        // A recovered job has no live caller, so it cannot be
                        // attributed to a submitter. It gets its own lane so
                        // recovery cannot evict a live caller's turn.
                        submitter: Some(format!("recovered:{}", job.run_id)),
                    };
                    let lane_key = request.submitter.clone().unwrap_or_default();
                    let rejection = {
                        let mut queue = self
                            .queue
                            .lock()
                            .map_err(|_| "workflow job queue poisoned".to_string())?;
                        let rejection = Self::check_queue_capacity(&queue, &lane_key);
                        if rejection.is_none() {
                            queue.push(&lane_key, request);
                        }
                        rejection
                    };
                    if let Some(message) = rejection {
                        self.active.lock().unwrap().remove(&job.run_id);
                        let _ = self.store.finish(
                            &job.run_id,
                            WorkflowJobStatus::Failed,
                            None,
                            &message,
                        );
                    }
                }
                WorkflowJobStatus::Running => {
                    if !self.store.execution_claim_is_active(&job.run_id) {
                        self.store.finish(
                            &job.run_id,
                            WorkflowJobStatus::Failed,
                            None,
                            "workflow job was interrupted by process restart; side effects are unknown",
                        )?;
                    }
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
        self.work_ready.notify_one();
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
    cancellation: CancellationToken,
    context: WorkerContext,
) {
    tokio::spawn(async move {
        loop {
            // Take one job per wake-up so a newly pushed job cannot be served
            // ahead of work that was already waiting: the queue is drained in
            // fair order, one item at a time.
            let request = tokio::select! {
                _ = cancellation.cancelled() => break,
                _ = context.work_ready.notified() => {
                    match context.queue.lock().ok().and_then(|mut queue| queue.pop()) {
                        Some(request) => request,
                        // A notification can arrive between the last pop and
                        // the wait; the queue being empty is not a reason to
                        // stop the worker.
                        None => continue,
                    }
                }
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
            // The local file claim protects this machine's workers; the shared
            // lease protects the ones on other machines, heartbeats while the
            // run executes, and releases when it ends. A disabled store yields
            // no lease and changes nothing.
            let Some(_execution_claim) = context.store.claim_execution_for(
                &request.run_id,
                &context.worker_id,
                context.distributed_claims.as_ref(),
            ) else {
                context
                    .active
                    .lock()
                    .ok()
                    .map(|mut map| map.remove(&request.run_id));
                continue;
            };
            let shared_lease = match context.distributed_claims.as_ref() {
                Some(store) => match ClaimLease::acquire(store, &request.run_id, &context.worker_id).await {
                    // `Some` means this worker now holds the shared claim; `None`
                    // means another machine does. Either way it is the answer.
                    Ok(lease) => lease,
                    Err(error) => {
                        // An unreachable shared store is not a reason to drive
                        // the run: the claim is the only thing standing between
                        // this worker and a second one on another machine.
                        tracing::warn!(
                            "cannot take the shared claim for {}: {error}",
                            request.run_id
                        );
                        None
                    }
                },
                // No shared store configured: nothing to take, and nothing to
                // refuse. The local file claim is the whole guard, as before.
                None => None,
            };
            if context.distributed_claims.is_some() && shared_lease.is_none() {
                // Another machine holds the run, or the shared store could not be
                // read. Both mean this worker must not drive it.
                context
                    .active
                    .lock()
                    .ok()
                    .map(|mut map| map.remove(&request.run_id));
                continue;
            }
            // Held for the rest of the run: it heartbeats while the steps execute
            // and releases when the run ends, including on the failure paths.
            let _shared_lease = shared_lease;
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
                    request.denied_tools.clone(),
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

/// Collapses a caller-supplied submitter into a lane key.
///
/// `None` or blank means the caller did not identify itself, and every
/// anonymous caller shares one lane: a flood of anonymous work then competes
/// as a single submitter instead of starving the identified ones. The value
/// is length-bounded so a hostile header cannot grow the lane map.
fn normalize_submitter(submitter: Option<&str>) -> String {
    match submitter.map(str::trim).filter(|value| !value.is_empty()) {
        Some(value) if value.len() <= 128 => value.to_string(),
        Some(value) => value.chars().take(128).collect(),
        None => ANONYMOUS_SUBMITTER.to_string(),
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

#[cfg(test)]
mod tests {
    use super::*;

    // What the worker acts on: the queued runs it can see, in the order it will
    // see them. Recovery reads the same list, so the helper exercises the
    // production path rather than a test-only shortcut.
    fn queued_ids(store: &WorkflowJobStore) -> Vec<String> {
        store
            .list()
            .unwrap()
            .into_iter()
            .filter(|job| job.status == WorkflowJobStatus::Queued)
            .map(|job| job.run_id)
            .collect()
    }

    // The single execution slot makes queue order the fairness question. A
    // busy submitter with several queued runs must not push a one-run
    // submitter behind all of them: with round-robin the quiet submitter's
    // job runs second.
    #[tokio::test]
    async fn a_busy_submitter_cannot_delay_a_quiet_one() {
        let temp = std::env::temp_dir().join(format!("quickdesk-job-fair-{}", Uuid::new_v4()));
        let store = WorkflowJobStore::with_storage_dir(temp.join("jobs"));

        let busy_runs = [
            store
                .create("workflow-busy", "device-1", &WorkflowJobRequestSpec::default())
                .unwrap()
                .run_id,
            store
                .create("workflow-busy", "device-1", &WorkflowJobRequestSpec::default())
                .unwrap()
                .run_id,
            store
                .create("workflow-busy", "device-1", &WorkflowJobRequestSpec::default())
                .unwrap()
                .run_id,
        ];
        let quiet_run = store
            .create("workflow-quiet", "device-1", &WorkflowJobRequestSpec::default())
            .unwrap()
            .run_id;

        // The queue is drained directly rather than through the worker: this
        // test is about ordering, not about execution.
        let mut queue = FairQueue::new();
        for run_id in &busy_runs {
            queue.push("operator:busy", run_id.clone());
        }
        queue.push("operator:quiet", quiet_run.clone());

        let mut served = Vec::new();
        while let Some(run_id) = queue.pop() {
            served.push(run_id);
        }
        assert_eq!(served.len(), 4);
        assert_eq!(served[0], busy_runs[0]);
        assert_eq!(
            served[1], quiet_run,
            "the single quiet job must be served second, not last"
        );
        assert_eq!(served[2], busy_runs[1]);
        assert_eq!(served[3], busy_runs[2]);

        let _ = std::fs::remove_dir_all(temp);
    }

    // The whole point of the shared store. Two managers on two machines - which
    // in a test means two stores whose directories cannot see each other, the
    // same situation a second host is in - must not both drive one run. Before
    // the shared claim each would open its own claim file and both would run.
    #[test]
    fn two_stores_that_cannot_see_each_other_share_one_claim() {
        let root = std::env::temp_dir().join(format!("quickdesk-xmachine-{}", Uuid::new_v4()));
        let local_a = WorkflowJobStore::with_storage_dir(root.join("machine-a").join("jobs"));
        let local_b = WorkflowJobStore::with_storage_dir(root.join("machine-b").join("jobs"));
        // One shared database, the only thing the two machines have in common.
        let claims = DistributedClaimStore::open(root.join("shared").join("claims.db"))
            .expect("open shared claims");

        // Both machines hold a copy of the same queued run. That is what a real
        // deployment looks like: each worker enqueued against its own store, so
        // the run id is the one thing the two directories agree on.
        let job = local_a
            .create("workflow-1", "device-1", &WorkflowJobRequestSpec::default())
            .unwrap();
        local_b.save(&job).expect("machine-b holds the same run");
        assert_eq!(
            local_b.get(&job.run_id).map(|row| row.run_id),
            Some(job.run_id.clone())
        );

        let held_by_a = local_a.claim_execution_for(&job.run_id, "machine-a", Some(&claims));
        assert!(held_by_a.is_some(), "the first machine must take the run");

        let held_by_b = local_b.claim_execution_for(&job.run_id, "machine-b", Some(&claims));
        assert!(
            held_by_b.is_none(),
            "the second machine drove a run another machine already holds"
        );
        assert_eq!(
            claims
                .holder(&job.run_id)
                .expect("holder")
                .map(|row| row.worker_id),
            Some("machine-a".to_string())
        );

        // When the holder goes away the run becomes drivable again, which is what
        // keeps a crashed worker from stranding the queue.
        drop(held_by_a);
        let stale = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_secs() as i64
            - (CLAIM_STALE_AFTER_SECONDS + 1);
        claims
            .age_claim_for_test(&job.run_id, stale)
            .expect("age the shared claim");

        assert!(
            local_b
                .claim_execution_for(&job.run_id, "machine-b", Some(&claims))
                .is_some(),
            "a dead holder's run was never recoverable"
        );

        let _ = std::fs::remove_dir_all(root);
    }

    // Without a shared store the behaviour is exactly what it was: the local
    // claim is the only guard, and a second store that cannot see the first one's
    // claim file claims the run too. This test documents the gap the shared store
    // closes rather than pretending it was never there.
    #[test]
    fn without_a_shared_store_two_blind_stores_both_claim() {
        let root = std::env::temp_dir().join(format!("quickdesk-noshare-{}", Uuid::new_v4()));
        let local_a = WorkflowJobStore::with_storage_dir(root.join("a").join("jobs"));
        let local_b = WorkflowJobStore::with_storage_dir(root.join("b").join("jobs"));
        let job_a = local_a
            .create("workflow-1", "device-1", &WorkflowJobRequestSpec::default())
            .unwrap();
        let job_b = local_b
            .create("workflow-1", "device-1", &WorkflowJobRequestSpec::default())
            .unwrap();

        assert!(local_a.claim_execution_for(&job_a.run_id, "a", None).is_some());
        assert!(
            local_b.claim_execution_for(&job_b.run_id, "b", None).is_some(),
            "the local-only path is unchanged, which is the documented gap"
        );

        let _ = std::fs::remove_dir_all(root);
    }

    #[test]
    fn stale_queued_job_claim_can_be_reacquired() {
        let temp = std::env::temp_dir().join(format!("quickdesk-job-{}", Uuid::new_v4()));
        let store = WorkflowJobStore::with_storage_dir(temp.join("jobs"));
        let job = store
            .create("workflow-1", "device-1", &WorkflowJobRequestSpec::default())
            .unwrap();
        let claim_path = store.claim_path(&job.run_id).unwrap();
        let claim = OpenOptions::new()
            .write(true)
            .create_new(true)
            .open(&claim_path)
            .unwrap();
        claim
            .set_times(
                std::fs::FileTimes::new().set_modified(SystemTime::now() - Duration::from_secs(31)),
            )
            .unwrap();
        drop(claim);

        let recovered = store.claim_execution_for(&job.run_id, "worker-a", None);
        assert!(recovered.is_some());
        drop(recovered);

        let _ = std::fs::remove_dir_all(temp);
    }

    // A fleet of workers is the whole point of the distributed queue: one
    // machine submits, another picks the run up. Submission writes into the
    // submitting machine's own directory, so the worker on the second machine
    // has to be able to name that directory - without it, every run sits in the
    // submitter's queue forever unless that one process happens to be up.
    #[test]
    fn a_worker_can_claim_a_run_submitted_on_another_machine() {
        let root = std::env::temp_dir().join(format!("quickdesk-fleet-{}", Uuid::new_v4()));
        let submitter = WorkflowJobStore::with_storage_dir(root.join("machine-a").join("jobs"));
        let worker = WorkflowJobStore::with_storage_dir(root.join("machine-b").join("jobs"));
        let job = submitter
            .create("workflow-1", "device-1", &WorkflowJobRequestSpec::default())
            .unwrap();

        // The second machine watches the submitter's directory as well as its
        // own, so the run is visible there even though nothing wrote it locally.
        worker.watch_peer_dir(root.join("machine-a").join("jobs"));
        assert_eq!(
            worker
                .list()
                .unwrap()
                .into_iter()
                .filter(|row| row.status == WorkflowJobStatus::Queued)
                .map(|row| row.run_id)
                .collect::<Vec<_>>(),
            vec![job.run_id.clone()],
            "a run submitted elsewhere must be visible to the worker"
        );

        // The shared claim is what stops the submitter's own worker and this one
        // from both driving the same run.
        let claims = DistributedClaimStore::open(root.join("shared").join("claims.db"))
            .expect("open shared claims");
        let taken = worker.claim_execution_for(&job.run_id, "machine-b", Some(&claims));
        assert!(
            taken.is_some(),
            "the watching worker must be able to claim it"
        );
        assert!(
            submitter
                .claim_execution_for(&job.run_id, "machine-a", Some(&claims))
                .is_none(),
            "the submitter must not drive a run the other machine already holds"
        );
        drop(taken);

        let _ = std::fs::remove_dir_all(root);
    }

    // The value comes from an environment variable, so trailing commas and
    // stray spaces are ordinary. An empty path would resolve to the current
    // directory, which is not what the operator asked for.
    #[test]
    fn peer_dirs_are_parsed_from_a_comma_separated_list() {
        assert_eq!(parse_peer_dirs(""), Vec::<PathBuf>::new());
        assert_eq!(parse_peer_dirs("   "), Vec::<PathBuf>::new());
        assert_eq!(parse_peer_dirs(","), Vec::<PathBuf>::new());
        assert_eq!(
            parse_peer_dirs(" /srv/a , /srv/b ,"),
            vec![PathBuf::from("/srv/a"), PathBuf::from("/srv/b")]
        );
    }

    // A run that is queued in a peer directory must not be listed twice when
    // that same directory is also watched by name, or when two peers both hold
    // a copy of it.
    #[test]
    fn a_run_seen_in_two_directories_is_listed_once() {
        let root = std::env::temp_dir().join(format!("quickdesk-dedupe-{}", Uuid::new_v4()));
        let submitter = WorkflowJobStore::with_storage_dir(root.join("jobs"));
        let worker = WorkflowJobStore::with_storage_dir(root.join("worker-jobs"));
        let job = submitter
            .create("workflow-1", "device-1", &WorkflowJobRequestSpec::default())
            .unwrap();

        // The same directory twice, which a ring of peers produces.
        worker.watch_peer_dir(root.join("jobs"));
        worker.watch_peer_dir(root.join("jobs"));
        assert_eq!(queued_ids(&worker).len(), 1, "the same run must not repeat");

        // And a peer that holds its own copy of the run.
        let other = WorkflowJobStore::with_storage_dir(root.join("other-jobs"));
        other.save(&job).expect("peer holds a copy");
        worker.watch_peer_dir(root.join("other-jobs"));
        assert_eq!(
            queued_ids(&worker).len(),
            1,
            "two copies of one run are still one run"
        );

        let _ = std::fs::remove_dir_all(root);
    }
}
