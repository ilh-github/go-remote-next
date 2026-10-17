use std::collections::HashMap;
use std::path::PathBuf;
use std::sync::{Arc, Mutex};

use chrono::Utc;
use serde::{Deserialize, Serialize};
use serde_json::Value;

#[derive(Clone, Debug, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum IdempotencyStatus {
    InProgress,
    Completed,
    Failed,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct IdempotencyRecord {
    pub key: String,
    pub workflow_id: String,
    pub device_id: String,
    pub request: Value,
    pub status: IdempotencyStatus,
    pub run_id: Option<String>,
    pub error: Option<String>,
    pub status_code: Option<u16>,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub enum IdempotencyClaim {
    Execute,
    Replay {
        run_id: Option<String>,
        error: Option<String>,
        status_code: u16,
    },
    InProgress,
    Conflict,
}

#[derive(Clone)]
pub struct WorkflowIdempotencyStore {
    path: PathBuf,
    records: Arc<Mutex<HashMap<String, IdempotencyRecord>>>,
}

impl WorkflowIdempotencyStore {
    pub fn new() -> Result<Self, String> {
        let base = dirs_data_dir().unwrap_or_else(|| PathBuf::from("."));
        Self::from_path(base.join(".quickdesk").join("workflow-idempotency.json"))
    }

    fn from_path(path: PathBuf) -> Result<Self, String> {
        let records = if path.exists() {
            let data = std::fs::read_to_string(&path)
                .map_err(|error| format!("read workflow idempotency store: {error}"))?;
            serde_json::from_str(&data)
                .map_err(|error| format!("parse workflow idempotency store: {error}"))?
        } else {
            HashMap::new()
        };
        Ok(Self {
            path,
            records: Arc::new(Mutex::new(records)),
        })
    }

    #[cfg(test)]
    pub(crate) fn with_storage_path(path: PathBuf) -> Self {
        Self::from_path(path).expect("test workflow idempotency store")
    }

    pub fn claim(
        &self,
        key: &str,
        workflow_id: &str,
        device_id: &str,
        request: &Value,
    ) -> Result<IdempotencyClaim, String> {
        validate_key(key)?;
        let mut records = self
            .records
            .lock()
            .map_err(|_| "workflow idempotency store lock poisoned".to_string())?;

        if let Some(existing) = records.get(key) {
            if existing.workflow_id != workflow_id
                || existing.device_id != device_id
                || existing.request != *request
            {
                return Ok(IdempotencyClaim::Conflict);
            }
            return Ok(match existing.status {
                IdempotencyStatus::InProgress => IdempotencyClaim::InProgress,
                IdempotencyStatus::Completed | IdempotencyStatus::Failed => {
                    IdempotencyClaim::Replay {
                        run_id: existing.run_id.clone(),
                        error: existing.error.clone(),
                        status_code: existing.status_code.unwrap_or(500),
                    }
                }
            });
        }

        let now = Utc::now().to_rfc3339();
        records.insert(
            key.to_string(),
            IdempotencyRecord {
                key: key.to_string(),
                workflow_id: workflow_id.to_string(),
                device_id: device_id.to_string(),
                request: request.clone(),
                status: IdempotencyStatus::InProgress,
                run_id: None,
                error: None,
                status_code: None,
                created_at: now.clone(),
                updated_at: now,
            },
        );
        self.save_locked(&records)?;
        Ok(IdempotencyClaim::Execute)
    }

    pub fn complete(&self, key: &str, run_id: &str) -> Result<(), String> {
        self.update(key, |record| {
            record.status = IdempotencyStatus::Completed;
            record.run_id = Some(run_id.to_string());
            record.status_code = Some(200);
            record.error = None;
        })
    }

    pub fn fail(&self, key: &str, status_code: u16, error: &str) -> Result<(), String> {
        self.update(key, |record| {
            record.status = IdempotencyStatus::Failed;
            record.status_code = Some(status_code);
            record.error = Some(error.to_string());
        })
    }

    fn update<F>(&self, key: &str, apply: F) -> Result<(), String>
    where
        F: FnOnce(&mut IdempotencyRecord),
    {
        let mut records = self
            .records
            .lock()
            .map_err(|_| "workflow idempotency store lock poisoned".to_string())?;
        let record = records
            .get_mut(key)
            .ok_or_else(|| "workflow idempotency record not found".to_string())?;
        apply(record);
        record.updated_at = Utc::now().to_rfc3339();
        self.save_locked(&records)
    }

    fn save_locked(&self, records: &HashMap<String, IdempotencyRecord>) -> Result<(), String> {
        if let Some(parent) = self.path.parent() {
            std::fs::create_dir_all(parent)
                .map_err(|error| format!("create workflow idempotency dir: {error}"))?;
        }
        let data = serde_json::to_string_pretty(records)
            .map_err(|error| format!("serialize workflow idempotency store: {error}"))?;
        std::fs::write(&self.path, data)
            .map_err(|error| format!("write workflow idempotency store: {error}"))
    }
}

fn validate_key(key: &str) -> Result<(), String> {
    if key.is_empty() || key.len() > 128 {
        return Err("Idempotency-Key must be 1..128 characters".to_string());
    }
    Ok(())
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
    use serde_json::json;
    use uuid::Uuid;

    #[test]
    fn completed_record_replays_after_the_store_is_reopened() {
        let dir = std::env::temp_dir().join(format!("quickdesk-idempotency-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&dir).unwrap();
        let path = dir.join("idempotency.json");
        let store = WorkflowIdempotencyStore::with_storage_path(path.clone());
        let request = json!({"overrides": {}, "stop_on_failure": true});

        assert_eq!(
            store
                .claim("request-1", "workflow-1", "device-1", &request)
                .unwrap(),
            IdempotencyClaim::Execute
        );
        store.complete("request-1", "run-1").unwrap();

        let reopened = WorkflowIdempotencyStore::with_storage_path(path);
        assert_eq!(
            reopened
                .claim("request-1", "workflow-1", "device-1", &request)
                .unwrap(),
            IdempotencyClaim::Replay {
                run_id: Some("run-1".to_string()),
                error: None,
                status_code: 200,
            }
        );
        let _ = std::fs::remove_dir_all(dir);
    }

    #[test]
    fn reusing_a_key_with_a_different_request_is_a_conflict() {
        let dir = std::env::temp_dir().join(format!("quickdesk-idempotency-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&dir).unwrap();
        let store = WorkflowIdempotencyStore::with_storage_path(dir.join("idempotency.json"));
        let request = json!({"overrides": {}, "stop_on_failure": true});

        assert_eq!(
            store
                .claim("request-1", "workflow-1", "device-1", &request)
                .unwrap(),
            IdempotencyClaim::Execute
        );
        assert_eq!(
            store
                .claim("request-1", "workflow-1", "device-2", &request)
                .unwrap(),
            IdempotencyClaim::Conflict
        );
        let _ = std::fs::remove_dir_all(dir);
    }
}
