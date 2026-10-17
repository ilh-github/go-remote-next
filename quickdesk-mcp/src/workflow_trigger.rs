use std::collections::HashMap;
use std::path::PathBuf;

use chrono::{DateTime, Duration, Utc};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use uuid::Uuid;

const MIN_INTERVAL_SECONDS: u64 = 60;
const MAX_INTERVAL_SECONDS: u64 = 365 * 24 * 60 * 60;

#[derive(Clone, Debug)]
pub struct WorkflowTriggerSpec {
    pub name: String,
    pub workflow_id: String,
    pub device_id: String,
    pub overrides: HashMap<String, Value>,
    pub stop_on_failure: bool,
    pub interval_seconds: Option<u64>,
}

#[derive(Clone, Debug, Serialize, Deserialize, PartialEq)]
pub struct WorkflowTrigger {
    pub id: String,
    pub name: String,
    pub workflow_id: String,
    pub device_id: String,
    pub overrides: HashMap<String, Value>,
    pub stop_on_failure: bool,
    pub interval_seconds: Option<u64>,
    pub enabled: bool,
    pub next_run_at: Option<String>,
    pub last_run_at: Option<String>,
    pub last_run_id: String,
    pub last_status: String,
    pub last_error: String,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Clone)]
pub struct WorkflowTriggerStore {
    storage_dir: PathBuf,
}

impl WorkflowTriggerStore {
    pub fn new() -> Result<Self, String> {
        let base = dirs_data_dir().unwrap_or_else(|| PathBuf::from("."));
        Self::from_storage_dir(base.join(".quickdesk").join("workflow-triggers"))
    }

    fn from_storage_dir(storage_dir: PathBuf) -> Result<Self, String> {
        std::fs::create_dir_all(&storage_dir)
            .map_err(|error| format!("cannot create workflow trigger dir: {error}"))?;
        Ok(Self { storage_dir })
    }

    #[cfg(test)]
    pub(crate) fn with_storage_dir(storage_dir: PathBuf) -> Self {
        Self::from_storage_dir(storage_dir).expect("test workflow trigger store")
    }

    pub fn create(&self, spec: WorkflowTriggerSpec) -> Result<WorkflowTrigger, String> {
        self.create_at(spec, Utc::now())
    }

    pub(crate) fn create_at(
        &self,
        spec: WorkflowTriggerSpec,
        now: DateTime<Utc>,
    ) -> Result<WorkflowTrigger, String> {
        let name = spec.name.trim();
        if name.is_empty() || name.len() > 128 {
            return Err("trigger name must be 1..128 characters".to_string());
        }
        Uuid::parse_str(spec.workflow_id.trim())
            .map_err(|_| "trigger workflow_id must be a UUID".to_string())?;
        let device_id = spec.device_id.trim();
        if device_id.is_empty() {
            return Err("trigger device_id is required".to_string());
        }
        if let Some(interval) = spec.interval_seconds {
            if !(MIN_INTERVAL_SECONDS..=MAX_INTERVAL_SECONDS).contains(&interval) {
                return Err(format!(
                    "trigger interval_seconds must be {MIN_INTERVAL_SECONDS}..{MAX_INTERVAL_SECONDS}"
                ));
            }
        }

        let now_text = now.to_rfc3339();
        let trigger = WorkflowTrigger {
            id: Uuid::new_v4().to_string(),
            name: name.to_string(),
            workflow_id: spec.workflow_id.trim().to_string(),
            device_id: device_id.to_string(),
            overrides: spec.overrides,
            stop_on_failure: spec.stop_on_failure,
            interval_seconds: spec.interval_seconds,
            enabled: true,
            next_run_at: spec
                .interval_seconds
                .map(|seconds| (now + Duration::seconds(seconds as i64)).to_rfc3339()),
            last_run_at: None,
            last_run_id: String::new(),
            last_status: String::new(),
            last_error: String::new(),
            created_at: now_text.clone(),
            updated_at: now_text,
        };
        self.save(&trigger)?;
        Ok(trigger)
    }

    pub fn list(&self) -> Vec<WorkflowTrigger> {
        let Ok(entries) = std::fs::read_dir(&self.storage_dir) else {
            return Vec::new();
        };
        let mut triggers = entries
            .flatten()
            .filter(|entry| entry.path().extension().is_some_and(|ext| ext == "json"))
            .filter_map(|entry| std::fs::read_to_string(entry.path()).ok())
            .filter_map(|data| serde_json::from_str::<WorkflowTrigger>(&data).ok())
            .collect::<Vec<_>>();
        triggers.sort_by(|left, right| right.created_at.cmp(&left.created_at));
        triggers
    }

    pub fn get(&self, id: &str) -> Option<WorkflowTrigger> {
        let path = self.trigger_path(id)?;
        let data = std::fs::read_to_string(path).ok()?;
        serde_json::from_str(&data).ok()
    }

    pub fn delete(&self, id: &str) -> bool {
        let Some(path) = self.trigger_path(id) else {
            return false;
        };
        std::fs::remove_file(path).is_ok()
    }

    #[cfg(test)]
    pub fn claim_due(&self, now: DateTime<Utc>) -> Vec<WorkflowTrigger> {
        self.claim_due_limit(now, usize::MAX)
    }

    pub fn claim_due_limit(&self, now: DateTime<Utc>, limit: usize) -> Vec<WorkflowTrigger> {
        let mut claimed = Vec::new();
        if limit == 0 {
            return claimed;
        }
        for mut trigger in self.list() {
            if !trigger.enabled || trigger.interval_seconds.is_none() {
                continue;
            }
            let Some(next_run_at) = trigger.next_run_at.as_deref() else {
                continue;
            };
            let Ok(next_run_at) = DateTime::parse_from_rfc3339(next_run_at) else {
                trigger.last_status = "failed".to_string();
                trigger.last_error = "invalid next_run_at".to_string();
                let _ = self.save(&trigger);
                continue;
            };
            if next_run_at.with_timezone(&Utc) > now {
                continue;
            }
            let interval = trigger.interval_seconds.unwrap_or(MIN_INTERVAL_SECONDS) as i64;
            trigger.next_run_at = Some((now + Duration::seconds(interval)).to_rfc3339());
            trigger.updated_at = now.to_rfc3339();
            if self.save(&trigger).is_ok() {
                claimed.push(trigger);
                if claimed.len() >= limit {
                    break;
                }
            }
        }
        claimed
    }

    pub fn record_run(
        &self,
        id: &str,
        run_id: &str,
        success: bool,
        error: &str,
        at: DateTime<Utc>,
    ) -> Result<(), String> {
        let mut trigger = self
            .get(id)
            .ok_or_else(|| "workflow trigger not found".to_string())?;
        trigger.last_run_at = Some(at.to_rfc3339());
        trigger.last_run_id = run_id.to_string();
        trigger.last_status = if success { "completed" } else { "failed" }.to_string();
        trigger.last_error = truncate(error, 512);
        trigger.updated_at = at.to_rfc3339();
        self.save(&trigger)
    }

    fn save(&self, trigger: &WorkflowTrigger) -> Result<(), String> {
        let path = self
            .trigger_path(&trigger.id)
            .ok_or_else(|| "invalid workflow trigger id".to_string())?;
        let data = serde_json::to_string_pretty(trigger)
            .map_err(|error| format!("serialize workflow trigger: {error}"))?;
        std::fs::write(path, data).map_err(|error| format!("write workflow trigger: {error}"))
    }

    fn trigger_path(&self, id: &str) -> Option<PathBuf> {
        Uuid::parse_str(id).ok()?;
        Some(self.storage_dir.join(format!("{id}.json")))
    }
}

fn truncate(value: &str, max: usize) -> String {
    if value.len() <= max {
        value.to_string()
    } else {
        format!("{}...", &value[..max])
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
    use chrono::{Duration, TimeZone, Utc};
    use serde_json::json;
    use uuid::Uuid;

    #[test]
    fn persists_trigger_definitions_and_runs() {
        let dir = std::env::temp_dir().join(format!("quickdesk-trigger-store-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&dir).unwrap();
        let store = WorkflowTriggerStore::with_storage_dir(dir.clone());
        let now = Utc.with_ymd_and_hms(2026, 10, 17, 15, 0, 0).unwrap();
        let workflow_id = Uuid::new_v4().to_string();

        let trigger = store
            .create_at(
                WorkflowTriggerSpec {
                    name: "nightly report".to_string(),
                    workflow_id: workflow_id.clone(),
                    device_id: "device-1".to_string(),
                    overrides: HashMap::from([("target".to_string(), json!("daily"))]),
                    stop_on_failure: true,
                    interval_seconds: Some(3600),
                },
                now,
            )
            .unwrap();

        assert_eq!(trigger.workflow_id, workflow_id);
        assert_eq!(
            trigger.next_run_at,
            Some((now + Duration::seconds(3600)).to_rfc3339())
        );
        assert_eq!(store.get(&trigger.id).unwrap().name, "nightly report");
        assert_eq!(store.list().len(), 1);

        store
            .record_run(&trigger.id, "run-1", true, "", now + Duration::seconds(60))
            .unwrap();
        let loaded = store.get(&trigger.id).unwrap();
        assert_eq!(loaded.last_run_id, "run-1");
        assert_eq!(loaded.last_status, "completed");
        assert!(loaded.last_error.is_empty());

        assert!(store.delete(&trigger.id));
        assert!(store.get(&trigger.id).is_none());
        let _ = std::fs::remove_dir_all(dir);
    }

    #[test]
    fn claims_a_due_interval_once_and_advances_it() {
        let dir = std::env::temp_dir().join(format!("quickdesk-trigger-store-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&dir).unwrap();
        let store = WorkflowTriggerStore::with_storage_dir(dir.clone());
        let created = Utc.with_ymd_and_hms(2026, 10, 17, 15, 0, 0).unwrap();
        let trigger = store
            .create_at(
                WorkflowTriggerSpec {
                    name: "every minute".to_string(),
                    workflow_id: Uuid::new_v4().to_string(),
                    device_id: "device-1".to_string(),
                    overrides: HashMap::new(),
                    stop_on_failure: true,
                    interval_seconds: Some(60),
                },
                created,
            )
            .unwrap();

        let first = store.claim_due(created + Duration::seconds(60));
        assert_eq!(first.len(), 1);
        assert_eq!(first[0].id, trigger.id);
        assert!(store.claim_due(created + Duration::seconds(61)).is_empty());

        let loaded = store.get(&trigger.id).unwrap();
        assert_eq!(
            loaded.next_run_at,
            Some((created + Duration::seconds(120)).to_rfc3339())
        );
        let _ = std::fs::remove_dir_all(dir);
    }

    #[test]
    fn rejects_invalid_trigger_ids_and_interval_bounds() {
        let dir = std::env::temp_dir().join(format!("quickdesk-trigger-store-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&dir).unwrap();
        let store = WorkflowTriggerStore::with_storage_dir(dir.clone());
        let base = WorkflowTriggerSpec {
            name: "invalid".to_string(),
            workflow_id: Uuid::new_v4().to_string(),
            device_id: "device-1".to_string(),
            overrides: HashMap::new(),
            stop_on_failure: true,
            interval_seconds: Some(1),
        };

        assert!(store.create(base.clone()).is_err());
        assert!(store.get("../../credentials").is_none());
        assert!(!store.delete("not-a-uuid"));
        let _ = std::fs::remove_dir_all(dir);
    }
}
