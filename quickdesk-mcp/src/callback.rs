use std::collections::HashSet;
use std::fs::OpenOptions;
use std::path::PathBuf;
use std::time::{Duration, SystemTime, UNIX_EPOCH};

use chrono::{DateTime, Utc};
use http_body_util::Full;
use hyper::body::Bytes;
use hyper::Request;
use hyper_util::client::legacy::Client;
use hyper_util::rt::TokioExecutor;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use sha1::{Digest, Sha1};
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;
use uuid::Uuid;

const HMAC_BLOCK_SIZE: usize = 64;
const DEFAULT_MAX_ATTEMPTS: u32 = 8;
const DEFAULT_RETRY_BASE: Duration = Duration::from_secs(5);
const DEFAULT_RETRY_MAX: Duration = Duration::from_secs(60 * 60);
const DEFAULT_POLL_INTERVAL: Duration = Duration::from_millis(250);
const CALLBACK_REQUEST_TIMEOUT: Duration = Duration::from_secs(10);
const CALLBACK_CLAIM_STALE_AFTER: Duration = Duration::from_secs(30);

#[derive(Clone, Debug)]
pub struct CallbackRetryPolicy {
    pub max_attempts: u32,
    pub base_delay: Duration,
    pub max_delay: Duration,
    pub poll_interval: Duration,
}

impl Default for CallbackRetryPolicy {
    fn default() -> Self {
        Self {
            max_attempts: DEFAULT_MAX_ATTEMPTS,
            base_delay: DEFAULT_RETRY_BASE,
            max_delay: DEFAULT_RETRY_MAX,
            poll_interval: DEFAULT_POLL_INTERVAL,
        }
    }
}

#[derive(Clone, Copy, Debug, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum CallbackDeliveryStatus {
    Pending,
    Retry,
    Delivered,
    Dead,
}

impl std::str::FromStr for CallbackDeliveryStatus {
    type Err = String;

    fn from_str(value: &str) -> Result<Self, Self::Err> {
        match value {
            "pending" => Ok(Self::Pending),
            "retry" => Ok(Self::Retry),
            "delivered" => Ok(Self::Delivered),
            "dead" => Ok(Self::Dead),
            other => Err(format!("unknown callback delivery status '{other}'")),
        }
    }
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct CallbackDelivery {
    pub id: String,
    pub url: String,
    pub event: Value,
    pub status: CallbackDeliveryStatus,
    pub attempts: u32,
    pub next_attempt_at: String,
    pub last_error: String,
    #[serde(default)]
    pub acknowledged_at: Option<String>,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Clone, Debug, Serialize)]
pub struct CallbackBatchAckResult {
    pub acknowledged: Vec<CallbackDelivery>,
    pub skipped: Vec<String>,
    pub not_found: Vec<String>,
}

#[derive(Clone)]
struct CallbackStore {
    storage_dir: PathBuf,
}

struct CallbackDeliveryClaim {
    path: PathBuf,
}

impl Drop for CallbackDeliveryClaim {
    fn drop(&mut self) {
        if let Err(error) = std::fs::remove_file(&self.path) {
            if error.kind() != std::io::ErrorKind::NotFound {
                tracing::warn!(
                    "cannot release callback delivery claim {}: {error}",
                    self.path.display()
                );
            }
        }
    }
}

impl CallbackStore {
    fn new(storage_dir: PathBuf) -> Result<Self, String> {
        std::fs::create_dir_all(&storage_dir)
            .map_err(|error| format!("cannot create callback outbox dir: {error}"))?;
        Ok(Self { storage_dir })
    }

    fn create(&self, url: &str, event: Value) -> Result<CallbackDelivery, String> {
        let now = Utc::now().to_rfc3339();
        let delivery = CallbackDelivery {
            id: Uuid::new_v4().to_string(),
            url: url.to_string(),
            event,
            status: CallbackDeliveryStatus::Pending,
            attempts: 0,
            next_attempt_at: now.clone(),
            last_error: String::new(),
            acknowledged_at: None,
            created_at: now.clone(),
            updated_at: now,
        };
        self.save(&delivery)?;
        Ok(delivery)
    }

    fn get(&self, delivery_id: &str) -> Option<CallbackDelivery> {
        let path = self.delivery_path(delivery_id)?;
        let data = std::fs::read_to_string(path).ok()?;
        serde_json::from_str(&data).ok()
    }

    fn list_due(&self, now: DateTime<Utc>) -> Result<Vec<CallbackDelivery>, String> {
        let mut due: Vec<_> = self
            .read_all()?
            .into_iter()
            .filter(|delivery| {
                matches!(
                    delivery.status,
                    CallbackDeliveryStatus::Pending | CallbackDeliveryStatus::Retry
                )
            })
            .filter(|delivery| {
                DateTime::parse_from_rfc3339(&delivery.next_attempt_at)
                    .map(|value| value.with_timezone(&Utc) <= now)
                    .unwrap_or(false)
            })
            .collect();
        due.sort_by(|left, right| left.next_attempt_at.cmp(&right.next_attempt_at));
        Ok(due)
    }

    fn claim_due(&self, delivery_id: &str, now: DateTime<Utc>) -> Option<CallbackDeliveryClaim> {
        let claim = self.acquire_claim(delivery_id)?;
        let delivery = self.get(delivery_id)?;
        let due = matches!(
            delivery.status,
            CallbackDeliveryStatus::Pending | CallbackDeliveryStatus::Retry
        ) && DateTime::parse_from_rfc3339(&delivery.next_attempt_at)
            .map(|value| value.with_timezone(&Utc) <= now)
            .unwrap_or(false);
        due.then_some(claim)
    }

    fn acquire_claim(&self, delivery_id: &str) -> Option<CallbackDeliveryClaim> {
        let path = self.claim_path(delivery_id)?;
        match OpenOptions::new().write(true).create_new(true).open(&path) {
            Ok(_) => Some(CallbackDeliveryClaim { path }),
            Err(error) if error.kind() == std::io::ErrorKind::AlreadyExists => {
                if self.claim_is_stale(&path) {
                    if let Err(remove_error) = std::fs::remove_file(&path) {
                        if remove_error.kind() != std::io::ErrorKind::NotFound {
                            tracing::warn!(
                                "cannot remove stale callback delivery claim {}: {remove_error}",
                                path.display()
                            );
                            return None;
                        }
                    }
                    return match OpenOptions::new().write(true).create_new(true).open(&path) {
                        Ok(_) => Some(CallbackDeliveryClaim { path }),
                        Err(_) => None,
                    };
                }
                None
            }
            Err(error) => {
                tracing::warn!("cannot claim callback delivery {}: {error}", path.display());
                None
            }
        }
    }

    fn claim_is_stale(&self, path: &std::path::Path) -> bool {
        std::fs::metadata(path)
            .and_then(|metadata| metadata.modified())
            .ok()
            .and_then(|modified| SystemTime::now().duration_since(modified).ok())
            .is_some_and(|age| age >= CALLBACK_CLAIM_STALE_AFTER)
    }

    fn list(
        &self,
        status: Option<CallbackDeliveryStatus>,
        limit: usize,
    ) -> Result<Vec<CallbackDelivery>, String> {
        let mut deliveries = self.read_all()?;
        deliveries.retain(|delivery| status.is_none_or(|expected| delivery.status == expected));
        deliveries.sort_by(|left, right| right.created_at.cmp(&left.created_at));
        deliveries.truncate(limit);
        Ok(deliveries)
    }

    fn read_all(&self) -> Result<Vec<CallbackDelivery>, String> {
        let entries = std::fs::read_dir(&self.storage_dir)
            .map_err(|error| format!("read callback outbox: {error}"))?;
        let mut deliveries = Vec::new();
        for entry in entries {
            let entry = entry.map_err(|error| format!("read callback outbox entry: {error}"))?;
            let path = entry.path();
            if path.extension().and_then(|value| value.to_str()) != Some("json") {
                continue;
            }
            let data = match std::fs::read_to_string(&path) {
                Ok(data) => data,
                Err(error) => {
                    tracing::warn!("cannot read callback delivery {}: {error}", path.display());
                    continue;
                }
            };
            let delivery: CallbackDelivery = match serde_json::from_str(&data) {
                Ok(delivery) => delivery,
                Err(error) => {
                    tracing::warn!("cannot parse callback delivery {}: {error}", path.display());
                    continue;
                }
            };
            deliveries.push(delivery);
        }
        Ok(deliveries)
    }

    fn retry_dead(&self, delivery_id: &str) -> Result<CallbackDelivery, CallbackRetryError> {
        let mut delivery = self.get(delivery_id).ok_or(CallbackRetryError::NotFound)?;
        if delivery.status != CallbackDeliveryStatus::Dead {
            return Err(CallbackRetryError::NotDead);
        }
        let now = Utc::now().to_rfc3339();
        delivery.status = CallbackDeliveryStatus::Pending;
        delivery.attempts = 0;
        delivery.next_attempt_at = now.clone();
        delivery.last_error.clear();
        delivery.updated_at = now;
        self.save(&delivery).map_err(CallbackRetryError::Store)?;
        Ok(delivery)
    }

    fn acknowledge(&self, delivery_id: &str) -> Result<CallbackDelivery, CallbackAckError> {
        let mut delivery = self.get(delivery_id).ok_or(CallbackAckError::NotFound)?;
        if !matches!(
            delivery.status,
            CallbackDeliveryStatus::Delivered | CallbackDeliveryStatus::Dead
        ) {
            return Err(CallbackAckError::NotTerminal);
        }
        if delivery.acknowledged_at.is_some() {
            return Ok(delivery);
        }
        let now = Utc::now().to_rfc3339();
        delivery.acknowledged_at = Some(now.clone());
        delivery.updated_at = now;
        self.save(&delivery).map_err(CallbackAckError::Store)?;
        Ok(delivery)
    }

    fn mark_delivered(&self, delivery_id: &str) -> Result<CallbackDelivery, String> {
        self.update(delivery_id, |delivery| {
            delivery.status = CallbackDeliveryStatus::Delivered;
            delivery.attempts += 1;
            delivery.last_error.clear();
            delivery.updated_at = Utc::now().to_rfc3339();
        })
    }

    fn mark_failed(
        &self,
        delivery_id: &str,
        error: &str,
        policy: &CallbackRetryPolicy,
    ) -> Result<CallbackDelivery, String> {
        self.update(delivery_id, |delivery| {
            delivery.attempts += 1;
            delivery.last_error = error.to_string();
            delivery.updated_at = Utc::now().to_rfc3339();
            if delivery.attempts >= policy.max_attempts {
                delivery.status = CallbackDeliveryStatus::Dead;
                return;
            }
            delivery.status = CallbackDeliveryStatus::Retry;
            let exponent = delivery.attempts.saturating_sub(1).min(31);
            let delay = policy
                .base_delay
                .saturating_mul(1_u32 << exponent)
                .min(policy.max_delay);
            delivery.next_attempt_at =
                (Utc::now() + chrono::Duration::from_std(delay).unwrap_or_default()).to_rfc3339();
        })
    }

    fn update<F>(&self, delivery_id: &str, apply: F) -> Result<CallbackDelivery, String>
    where
        F: FnOnce(&mut CallbackDelivery),
    {
        let mut delivery = self
            .get(delivery_id)
            .ok_or_else(|| "callback delivery not found".to_string())?;
        apply(&mut delivery);
        self.save(&delivery)?;
        Ok(delivery)
    }

    fn save(&self, delivery: &CallbackDelivery) -> Result<(), String> {
        let path = self
            .delivery_path(&delivery.id)
            .ok_or_else(|| "invalid callback delivery id".to_string())?;
        let data = serde_json::to_string_pretty(delivery)
            .map_err(|error| format!("serialize callback delivery: {error}"))?;
        let temp_path = path.with_extension("json.tmp");
        std::fs::write(&temp_path, data)
            .map_err(|error| format!("write callback delivery: {error}"))?;
        if path.exists() {
            std::fs::remove_file(&path)
                .map_err(|error| format!("replace callback delivery: {error}"))?;
        }
        std::fs::rename(&temp_path, &path)
            .map_err(|error| format!("publish callback delivery: {error}"))
    }

    fn delivery_path(&self, delivery_id: &str) -> Option<PathBuf> {
        Uuid::parse_str(delivery_id).ok()?;
        Some(self.storage_dir.join(format!("{delivery_id}.json")))
    }

    fn claim_path(&self, delivery_id: &str) -> Option<PathBuf> {
        Uuid::parse_str(delivery_id).ok()?;
        Some(self.storage_dir.join(format!("{delivery_id}.claim")))
    }
}

#[derive(Clone)]
pub struct CallbackOutbox {
    store: CallbackStore,
    wake_tx: mpsc::UnboundedSender<String>,
}

#[derive(Debug, PartialEq, Eq)]
pub enum CallbackRetryError {
    NotFound,
    NotDead,
    Store(String),
}

#[derive(Debug, PartialEq, Eq)]
pub enum CallbackAckError {
    NotFound,
    NotTerminal,
    Store(String),
}

impl CallbackOutbox {
    pub fn new(cancellation: CancellationToken) -> Result<Self, String> {
        let base = dirs_data_dir().unwrap_or_else(|| PathBuf::from("."));
        Self::with_storage_dir(
            base.join(".quickdesk").join("callback-outbox"),
            cancellation,
            CallbackRetryPolicy::default(),
            std::env::var("QUICKDESK_CALLBACK_SECRET")
                .ok()
                .filter(|value| !value.is_empty()),
        )
    }

    fn with_storage_dir(
        storage_dir: PathBuf,
        cancellation: CancellationToken,
        policy: CallbackRetryPolicy,
        secret: Option<String>,
    ) -> Result<Self, String> {
        let store = CallbackStore::new(storage_dir)?;
        let (wake_tx, wake_rx) = mpsc::unbounded_channel();
        start_worker(
            store.clone(),
            wake_rx,
            cancellation,
            CallbackClient::with_secret(secret),
            policy,
        );
        Ok(Self { store, wake_tx })
    }

    pub fn enqueue(&self, url: Option<&str>, event: Value) -> Result<Option<String>, String> {
        let Some(url) = url.map(str::trim).filter(|value| !value.is_empty()) else {
            return Ok(None);
        };
        let delivery = self.store.create(url, event)?;
        if self.wake_tx.send(delivery.id.clone()).is_err() {
            return Err("callback outbox worker is not running".to_string());
        }
        Ok(Some(delivery.id))
    }

    pub fn list(
        &self,
        status: Option<CallbackDeliveryStatus>,
        limit: usize,
    ) -> Result<Vec<CallbackDelivery>, String> {
        self.store.list(status, limit)
    }

    pub fn retry_dead(&self, delivery_id: &str) -> Result<CallbackDelivery, CallbackRetryError> {
        let delivery = self.store.retry_dead(delivery_id)?;
        self.wake_tx.send(delivery.id.clone()).map_err(|_| {
            CallbackRetryError::Store("callback outbox worker is not running".to_string())
        })?;
        Ok(delivery)
    }

    pub fn acknowledge(&self, delivery_id: &str) -> Result<CallbackDelivery, CallbackAckError> {
        self.store.acknowledge(delivery_id)
    }

    pub fn acknowledge_many(
        &self,
        delivery_ids: &[String],
    ) -> Result<CallbackBatchAckResult, String> {
        let mut result = CallbackBatchAckResult {
            acknowledged: Vec::new(),
            skipped: Vec::new(),
            not_found: Vec::new(),
        };
        let mut seen = HashSet::new();
        for delivery_id in delivery_ids {
            let delivery_id = delivery_id.trim();
            if !seen.insert(delivery_id.to_string()) {
                continue;
            }
            match self.store.acknowledge(delivery_id) {
                Ok(delivery) => result.acknowledged.push(delivery),
                Err(CallbackAckError::NotTerminal) => {
                    result.skipped.push(delivery_id.to_string());
                }
                Err(CallbackAckError::NotFound) => {
                    result.not_found.push(delivery_id.to_string());
                }
                Err(CallbackAckError::Store(error)) => return Err(error),
            }
        }
        Ok(result)
    }

    #[cfg(test)]
    pub(crate) fn with_storage_dir_and_secret(
        storage_dir: PathBuf,
        cancellation: CancellationToken,
        policy: CallbackRetryPolicy,
        secret: &str,
    ) -> Result<Self, String> {
        Self::with_storage_dir(storage_dir, cancellation, policy, Some(secret.to_string()))
    }

    #[cfg(test)]
    pub(crate) fn get(&self, delivery_id: &str) -> Option<CallbackDelivery> {
        self.store.get(delivery_id)
    }
}

fn start_worker(
    store: CallbackStore,
    mut wake_rx: mpsc::UnboundedReceiver<String>,
    cancellation: CancellationToken,
    client: CallbackClient,
    policy: CallbackRetryPolicy,
) {
    tokio::spawn(async move {
        loop {
            if cancellation.is_cancelled() {
                break;
            }
            match store.list_due(Utc::now()) {
                Ok(deliveries) => {
                    for delivery in deliveries {
                        if cancellation.is_cancelled() {
                            return;
                        }
                        let Some(_claim) = store.claim_due(&delivery.id, Utc::now()) else {
                            continue;
                        };
                        let result = client
                            .deliver(&delivery.url, &delivery.event, &delivery.id)
                            .await;
                        match result {
                            Ok(()) => {
                                if let Err(error) = store.mark_delivered(&delivery.id) {
                                    tracing::warn!(
                                        "cannot mark callback {} delivered: {error}",
                                        delivery.id
                                    );
                                }
                            }
                            Err(error) => match store.mark_failed(&delivery.id, &error, &policy) {
                                Ok(updated) => {
                                    tracing::warn!(
                                        "workflow callback {} failed on attempt {}: {}",
                                        delivery.id,
                                        updated.attempts,
                                        error
                                    );
                                }
                                Err(store_error) => {
                                    tracing::warn!(
                                        "cannot record callback {} failure: {store_error}",
                                        delivery.id
                                    );
                                }
                            },
                        }
                    }
                }
                Err(error) => tracing::warn!("scan callback outbox failed: {error}"),
            }

            tokio::select! {
                _ = cancellation.cancelled() => break,
                message = wake_rx.recv() => {
                    if message.is_none() {
                        break;
                    }
                }
                _ = tokio::time::sleep(policy.poll_interval) => {}
            }
        }
    });
}

pub struct CallbackClient {
    client: Client<hyper_util::client::legacy::connect::HttpConnector, Full<Bytes>>,
    secret: Option<String>,
}

impl CallbackClient {
    fn with_secret(secret: Option<String>) -> Self {
        let client = Client::builder(TokioExecutor::new())
            .pool_idle_timeout(Duration::from_secs(30))
            .build_http();
        Self { client, secret }
    }

    async fn deliver(&self, url: &str, event: &Value, delivery_id: &str) -> Result<(), String> {
        let secret = self
            .secret
            .as_deref()
            .ok_or_else(|| "QUICKDESK_CALLBACK_SECRET is not configured".to_string())?;
        let parsed: hyper::Uri = url
            .parse()
            .map_err(|error| format!("invalid callback URL: {error}"))?;
        if parsed.scheme_str() != Some("http") {
            return Err(
                "callback URL must use http; put HTTPS behind a local TLS reverse proxy"
                    .to_string(),
            );
        }

        let body = serde_json::to_vec(event)
            .map_err(|error| format!("serialize callback event: {error}"))?;
        let timestamp = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map_err(|error| format!("callback clock error: {error}"))?
            .as_secs()
            .to_string();
        let signature = sign(secret, &timestamp, &body);
        let event_name = event
            .get("event")
            .and_then(Value::as_str)
            .unwrap_or("workflow.run");

        let request = Request::builder()
            .method("POST")
            .uri(parsed)
            .header("Content-Type", "application/json")
            .header("X-QuickDesk-Event", event_name)
            .header("X-QuickDesk-Delivery-Id", delivery_id)
            .header("X-QuickDesk-Timestamp", timestamp)
            .header("X-QuickDesk-Signature", format!("hmac-sha1={signature}"))
            .body(Full::new(Bytes::from(body)))
            .map_err(|error| format!("build callback request: {error}"))?;
        let response = tokio::time::timeout(CALLBACK_REQUEST_TIMEOUT, self.client.request(request))
            .await
            .map_err(|_| "callback request timed out".to_string())?
            .map_err(|error| format!("callback request failed: {error}"))?;
        if !response.status().is_success() {
            return Err(format!("callback returned HTTP {}", response.status()));
        }
        Ok(())
    }
}

fn sign(secret: &str, timestamp: &str, body: &[u8]) -> String {
    let mut message = Vec::with_capacity(timestamp.len() + 1 + body.len());
    message.extend_from_slice(timestamp.as_bytes());
    message.push(b'.');
    message.extend_from_slice(body);
    hex(&hmac_sha1(secret.as_bytes(), &message))
}

fn hmac_sha1(key: &[u8], message: &[u8]) -> Vec<u8> {
    let key = if key.len() > HMAC_BLOCK_SIZE {
        Sha1::digest(key).to_vec()
    } else {
        key.to_vec()
    };
    let mut inner_pad = vec![0x36; HMAC_BLOCK_SIZE];
    let mut outer_pad = vec![0x5c; HMAC_BLOCK_SIZE];
    for (index, byte) in key.iter().enumerate() {
        inner_pad[index] ^= byte;
        outer_pad[index] ^= byte;
    }

    let mut inner = Sha1::new();
    inner.update(&inner_pad);
    inner.update(message);
    let inner_digest = inner.finalize();

    let mut outer = Sha1::new();
    outer.update(&outer_pad);
    outer.update(inner_digest);
    outer.finalize().to_vec()
}

fn hex(bytes: &[u8]) -> String {
    const HEX: &[u8; 16] = b"0123456789abcdef";
    let mut output = String::with_capacity(bytes.len() * 2);
    for byte in bytes {
        output.push(HEX[(byte >> 4) as usize] as char);
        output.push(HEX[(byte & 0x0f) as usize] as char);
    }
    output
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
    use std::sync::atomic::{AtomicUsize, Ordering};
    use std::sync::Arc;
    use tokio::io::{AsyncReadExt, AsyncWriteExt};

    #[test]
    fn hmac_sha1_matches_rfc_2202_case_1() {
        let digest = hmac_sha1(&[0x0b; 20], b"Hi There");
        assert_eq!(hex(&digest), "b617318655057264e28bc0b6fb378c8ef146be00");
    }

    #[tokio::test]
    async fn callback_outbox_resumes_a_pending_delivery_after_restart() {
        let temp = std::env::temp_dir().join(format!("quickdesk-callback-{}", Uuid::new_v4()));
        let outbox_dir = temp.join("outbox");
        let store = CallbackStore::new(outbox_dir.clone()).unwrap();
        let pending = store
            .create(
                "http://127.0.0.1:1/resume",
                serde_json::json!({"event": "workflow.run.completed", "run_id": "resume"}),
            )
            .unwrap();

        let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
        let address = listener.local_addr().unwrap();
        let (request_tx, request_rx) = tokio::sync::oneshot::channel();
        let server = tokio::spawn(async move {
            let (mut stream, _) = listener.accept().await.unwrap();
            let mut buffer = vec![0u8; 8192];
            let read = stream.read(&mut buffer).await.unwrap();
            let request = String::from_utf8_lossy(&buffer[..read]).to_string();
            let _ = request_tx.send(request);
            stream
                .write_all(b"HTTP/1.1 200 OK\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
                .await
                .unwrap();
        });
        let pending_path = outbox_dir.join(format!("{}.json", pending.id));
        let mut persisted: CallbackDelivery =
            serde_json::from_str(&std::fs::read_to_string(&pending_path).unwrap()).unwrap();
        persisted.url = format!("http://{address}/resume");
        store.save(&persisted).unwrap();

        let cancellation = CancellationToken::new();
        let outbox = CallbackOutbox::with_storage_dir_and_secret(
            outbox_dir,
            cancellation.clone(),
            CallbackRetryPolicy {
                max_attempts: 3,
                base_delay: Duration::from_millis(10),
                max_delay: Duration::from_millis(50),
                poll_interval: Duration::from_millis(10),
            },
            "callback-outbox-secret",
        )
        .unwrap();
        let request = tokio::time::timeout(Duration::from_secs(2), request_rx)
            .await
            .expect("pending callback should resume after restart")
            .unwrap();
        assert!(request.contains(&pending.id));
        tokio::time::timeout(Duration::from_secs(2), async {
            loop {
                if outbox
                    .get(&pending.id)
                    .is_some_and(|delivery| delivery.status == CallbackDeliveryStatus::Delivered)
                {
                    break;
                }
                tokio::time::sleep(Duration::from_millis(10)).await;
            }
        })
        .await
        .expect("callback should be marked delivered");
        cancellation.cancel();
        server.abort();
        let _ = std::fs::remove_dir_all(temp);
    }

    #[tokio::test]
    async fn two_outboxes_do_not_deliver_the_same_callback_concurrently() {
        let temp = std::env::temp_dir().join(format!("quickdesk-callback-{}", Uuid::new_v4()));
        let outbox_dir = temp.join("outbox");
        let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
        let address = listener.local_addr().unwrap();
        let attempts = Arc::new(AtomicUsize::new(0));
        let server_cancellation = CancellationToken::new();
        let server_attempts = attempts.clone();
        let server_cancellation_for_task = server_cancellation.clone();
        let server = tokio::spawn(async move {
            loop {
                let accepted = tokio::select! {
                    _ = server_cancellation_for_task.cancelled() => break,
                    accepted = listener.accept() => accepted,
                };
                let Ok((mut stream, _)) = accepted else {
                    break;
                };
                let attempts = server_attempts.clone();
                tokio::spawn(async move {
                    let mut buffer = vec![0u8; 8192];
                    let _ = stream.read(&mut buffer).await;
                    attempts.fetch_add(1, Ordering::SeqCst);
                    tokio::time::sleep(Duration::from_millis(500)).await;
                    let _ = stream
                        .write_all(
                            b"HTTP/1.1 200 OK\r\nContent-Length: 0\r\nConnection: close\r\n\r\n",
                        )
                        .await;
                });
            }
        });

        let policy = CallbackRetryPolicy {
            max_attempts: 3,
            base_delay: Duration::from_millis(10),
            max_delay: Duration::from_millis(50),
            poll_interval: Duration::from_millis(10),
        };
        let first_cancellation = CancellationToken::new();
        let first = CallbackOutbox::with_storage_dir_and_secret(
            outbox_dir.clone(),
            first_cancellation.clone(),
            policy.clone(),
            "callback-outbox-secret",
        )
        .unwrap();
        let second_cancellation = CancellationToken::new();
        let _second = CallbackOutbox::with_storage_dir_and_secret(
            outbox_dir,
            second_cancellation.clone(),
            policy,
            "callback-outbox-secret",
        )
        .unwrap();

        let delivery_id = first
            .enqueue(
                Some(&format!("http://{address}/shared-outbox")),
                serde_json::json!({"event": "workflow.run.completed", "run_id": "shared"}),
            )
            .unwrap()
            .expect("callback should be queued");
        tokio::time::timeout(Duration::from_secs(3), async {
            loop {
                if first
                    .get(&delivery_id)
                    .is_some_and(|delivery| delivery.status == CallbackDeliveryStatus::Delivered)
                {
                    break;
                }
                tokio::time::sleep(Duration::from_millis(10)).await;
            }
        })
        .await
        .expect("callback should be delivered");
        tokio::time::sleep(Duration::from_millis(300)).await;

        assert_eq!(attempts.load(Ordering::SeqCst), 1);
        first_cancellation.cancel();
        second_cancellation.cancel();
        server_cancellation.cancel();
        server.abort();
        let _ = std::fs::remove_dir_all(temp);
    }

    #[tokio::test]
    async fn callback_outbox_recovers_a_stale_delivery_claim() {
        let temp = std::env::temp_dir().join(format!("quickdesk-callback-{}", Uuid::new_v4()));
        let outbox_dir = temp.join("outbox");
        let store = CallbackStore::new(outbox_dir.clone()).unwrap();
        let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
        let address = listener.local_addr().unwrap();
        let pending = store
            .create(
                &format!("http://{address}/stale-claim"),
                serde_json::json!({"event": "workflow.run.completed", "run_id": "stale"}),
            )
            .unwrap();
        let claim_path = outbox_dir.join(format!("{}.claim", pending.id));
        let claim = std::fs::OpenOptions::new()
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

        let server = tokio::spawn(async move {
            let (mut stream, _) = listener.accept().await.unwrap();
            let mut buffer = vec![0u8; 8192];
            let _ = stream.read(&mut buffer).await;
            stream
                .write_all(b"HTTP/1.1 200 OK\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
                .await
                .unwrap();
        });
        let cancellation = CancellationToken::new();
        let outbox = CallbackOutbox::with_storage_dir_and_secret(
            outbox_dir,
            cancellation.clone(),
            CallbackRetryPolicy {
                max_attempts: 3,
                base_delay: Duration::from_millis(10),
                max_delay: Duration::from_millis(50),
                poll_interval: Duration::from_millis(10),
            },
            "callback-outbox-secret",
        )
        .unwrap();
        tokio::time::timeout(Duration::from_secs(2), async {
            loop {
                if outbox
                    .get(&pending.id)
                    .is_some_and(|delivery| delivery.status == CallbackDeliveryStatus::Delivered)
                {
                    break;
                }
                tokio::time::sleep(Duration::from_millis(10)).await;
            }
        })
        .await
        .expect("stale claim should be recovered");

        cancellation.cancel();
        server.abort();
        let _ = std::fs::remove_dir_all(temp);
    }
}
