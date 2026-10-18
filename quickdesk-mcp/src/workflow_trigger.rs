use std::collections::HashMap;
use std::path::PathBuf;
use std::str::FromStr;

use chrono::{DateTime, Duration, Utc};
use cron::Schedule;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use uuid::Uuid;

const MIN_INTERVAL_SECONDS: u64 = 60;
const MAX_INTERVAL_SECONDS: u64 = 365 * 24 * 60 * 60;

pub const SUPPORTED_EVENT_TYPES: &[&str] = &[
    "device.online.changed",
    "device.offline.sustained",
    "device.metric.alert",
    "device.access.granted",
    "device.access.denied",
    "device.access_approval.requested",
    "device.access_approval.decided",
    "remote_session.started",
    "remote_session.ended",
    "recording.created",
    "recording.updated",
    "recording.deleted",
    "scheduled_task.failed",
];

#[derive(Clone, Debug)]
pub struct WorkflowTriggerSpec {
    pub name: String,
    pub workflow_id: String,
    pub device_id: String,
    pub overrides: HashMap<String, Value>,
    pub stop_on_failure: bool,
    pub interval_seconds: Option<u64>,
    pub cron_expression: Option<String>,
    pub event_type: Option<String>,
    pub event_device_id: Option<String>,
    pub event_filter: Option<Value>,
    pub callback_url: Option<String>,
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
    #[serde(default)]
    pub cron_expression: Option<String>,
    #[serde(default)]
    pub event_type: Option<String>,
    #[serde(default)]
    pub event_device_id: Option<String>,
    #[serde(default)]
    pub event_filter: Option<Value>,
    #[serde(default)]
    pub callback_url: Option<String>,
    pub enabled: bool,
    pub next_run_at: Option<String>,
    pub last_run_at: Option<String>,
    pub last_run_id: String,
    pub last_status: String,
    pub last_error: String,
    #[serde(default)]
    pub last_event_id: Option<String>,
    #[serde(default)]
    pub last_server_rev: i64,
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
        let cron_expression = spec
            .cron_expression
            .as_deref()
            .map(str::trim)
            .filter(|value| !value.is_empty())
            .map(str::to_string);
        if spec.interval_seconds.is_some() && cron_expression.is_some() {
            return Err(
                "trigger can use only one of interval_seconds or cron_expression".to_string(),
            );
        }
        if let Some(expression) = cron_expression.as_deref() {
            validate_cron_expression(expression)?;
        }
        let event_type = spec
            .event_type
            .as_deref()
            .map(str::trim)
            .filter(|value| !value.is_empty())
            .map(str::to_string);
        if spec.interval_seconds.is_none() && cron_expression.is_none() && event_type.is_none() {
            return Err(
                "trigger needs interval_seconds, cron_expression or event_type".to_string(),
            );
        }
        if let Some(event_type) = event_type.as_deref() {
            if !SUPPORTED_EVENT_TYPES.contains(&event_type) {
                return Err("trigger event_type is not supported".to_string());
            }
        }
        if let Some(filter) = spec.event_filter.as_ref() {
            validate_event_filter(filter, 0)?;
        }
        let event_device_id = if event_type.is_some() {
            spec.event_device_id
                .as_deref()
                .map(str::trim)
                .filter(|value| !value.is_empty())
                .map(str::to_string)
                .or_else(|| {
                    event_type
                        .as_deref()
                        .is_some_and(event_is_device_scoped)
                        .then(|| device_id.to_string())
                })
        } else {
            None
        };

        let now_text = now.to_rfc3339();
        let next_run_at = if let Some(seconds) = spec.interval_seconds {
            Some((now + Duration::seconds(seconds as i64)).to_rfc3339())
        } else if let Some(expression) = cron_expression.as_deref() {
            Some(next_cron_after(expression, now)?.to_rfc3339())
        } else {
            None
        };
        let trigger = WorkflowTrigger {
            id: Uuid::new_v4().to_string(),
            name: name.to_string(),
            workflow_id: spec.workflow_id.trim().to_string(),
            device_id: device_id.to_string(),
            overrides: spec.overrides,
            stop_on_failure: spec.stop_on_failure,
            interval_seconds: spec.interval_seconds,
            cron_expression,
            event_type,
            event_device_id,
            event_filter: spec.event_filter,
            callback_url: spec.callback_url,
            enabled: true,
            next_run_at,
            last_run_at: None,
            last_run_id: String::new(),
            last_status: String::new(),
            last_error: String::new(),
            last_event_id: None,
            last_server_rev: 0,
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
            if !trigger.enabled
                || (trigger.interval_seconds.is_none() && trigger.cron_expression.is_none())
            {
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
            let next_run_at = if let Some(interval) = trigger.interval_seconds {
                (now + Duration::seconds(interval as i64)).to_rfc3339()
            } else {
                let Some(expression) = trigger.cron_expression.as_deref() else {
                    continue;
                };
                match next_cron_after(expression, now) {
                    Ok(next) => next.to_rfc3339(),
                    Err(error) => {
                        trigger.last_status = "failed".to_string();
                        trigger.last_error = error;
                        let _ = self.save(&trigger);
                        continue;
                    }
                }
            };
            trigger.next_run_at = Some(next_run_at);
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

    pub fn claim_event(
        &self,
        id: &str,
        event_id: Option<&str>,
        server_rev: i64,
        at: DateTime<Utc>,
    ) -> Result<bool, String> {
        let mut trigger = self
            .get(id)
            .ok_or_else(|| "workflow trigger not found".to_string())?;
        if !trigger.enabled || trigger.event_type.is_none() {
            return Ok(false);
        }
        if let Some(event_id) = event_id {
            if trigger.last_event_id.as_deref() == Some(event_id) {
                return Ok(false);
            }
        } else if server_rev <= 0 || server_rev <= trigger.last_server_rev {
            return Ok(false);
        }

        trigger.last_event_id = event_id.map(str::to_string);
        if server_rev > 0 {
            trigger.last_server_rev = server_rev;
        }
        trigger.updated_at = at.to_rfc3339();
        self.save(&trigger)?;
        Ok(true)
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

fn validate_cron_expression(expression: &str) -> Result<(), String> {
    Schedule::from_str(expression)
        .map(|_| ())
        .map_err(|error| format!("invalid cron_expression: {error}"))
}

fn next_cron_after(expression: &str, after: DateTime<Utc>) -> Result<DateTime<Utc>, String> {
    let schedule = Schedule::from_str(expression)
        .map_err(|error| format!("invalid cron_expression: {error}"))?;
    schedule
        .after(&after)
        .next()
        .ok_or_else(|| "cron_expression has no future run".to_string())
}

fn truncate(value: &str, max: usize) -> String {
    if value.len() <= max {
        value.to_string()
    } else {
        format!("{}...", &value[..max])
    }
}

fn validate_event_filter(value: &Value, depth: usize) -> Result<(), String> {
    if depth > 4 {
        return Err("trigger event_filter is nested too deeply".to_string());
    }
    let Some(object) = value.as_object() else {
        return Err("trigger event_filter must be an object".to_string());
    };
    if object.len() > 64 {
        return Err("trigger event_filter has too many keys".to_string());
    }
    for (key, nested) in object {
        let normalized = key.trim().to_ascii_lowercase();
        if normalized.is_empty() || key.len() > 128 {
            return Err("trigger event_filter contains an invalid key".to_string());
        }
        if normalized.contains("access_code")
            || normalized.contains("password")
            || normalized.contains("secret")
            || normalized.contains("token")
        {
            return Err("trigger event_filter cannot match sensitive fields".to_string());
        }
        if nested.as_str().is_some_and(|text| text.len() > 4096) {
            return Err("trigger event_filter string is too long".to_string());
        }
        if nested.is_object() {
            validate_event_filter(nested, depth + 1)?;
        }
    }
    Ok(())
}

fn event_is_device_scoped(event_type: &str) -> bool {
    event_type != "scheduled_task.failed"
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
                    cron_expression: None,
                    event_type: None,
                    event_device_id: None,
                    event_filter: None,
                    callback_url: None,
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
                    cron_expression: None,
                    event_type: None,
                    event_device_id: None,
                    event_filter: None,
                    callback_url: None,
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
    fn claims_a_due_cron_once_and_advances_to_the_next_slot() {
        let dir = std::env::temp_dir().join(format!("quickdesk-trigger-store-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&dir).unwrap();
        let store = WorkflowTriggerStore::with_storage_dir(dir.clone());
        let created = Utc.with_ymd_and_hms(2026, 10, 17, 15, 7, 0).unwrap();
        let trigger = store
            .create_at(
                WorkflowTriggerSpec {
                    name: "every fifteen minutes".to_string(),
                    workflow_id: Uuid::new_v4().to_string(),
                    device_id: "device-1".to_string(),
                    overrides: HashMap::new(),
                    stop_on_failure: true,
                    interval_seconds: None,
                    cron_expression: Some("0 */15 * * * *".to_string()),
                    event_type: None,
                    event_device_id: None,
                    event_filter: None,
                    callback_url: None,
                },
                created,
            )
            .unwrap();
        assert_eq!(
            trigger.next_run_at,
            Some(
                Utc.with_ymd_and_hms(2026, 10, 17, 15, 15, 0)
                    .unwrap()
                    .to_rfc3339()
            )
        );

        let first = store.claim_due(Utc.with_ymd_and_hms(2026, 10, 17, 15, 15, 0).unwrap());
        assert_eq!(first.len(), 1);
        assert!(store
            .claim_due(Utc.with_ymd_and_hms(2026, 10, 17, 15, 16, 0).unwrap())
            .is_empty());
        assert_eq!(
            store.get(&trigger.id).unwrap().next_run_at,
            Some(
                Utc.with_ymd_and_hms(2026, 10, 17, 15, 30, 0)
                    .unwrap()
                    .to_rfc3339()
            )
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
            cron_expression: None,
            event_type: None,
            event_device_id: None,
            event_filter: None,
            callback_url: None,
        };

        assert!(store.create(base.clone()).is_err());
        assert!(store.get("../../credentials").is_none());
        assert!(!store.delete("not-a-uuid"));
        let _ = std::fs::remove_dir_all(dir);
    }

    #[test]
    fn event_trigger_defaults_to_target_device_and_claims_one_server_rev() {
        let dir = std::env::temp_dir().join(format!("quickdesk-trigger-store-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&dir).unwrap();
        let store = WorkflowTriggerStore::with_storage_dir(dir.clone());
        let now = Utc.with_ymd_and_hms(2026, 10, 17, 15, 0, 0).unwrap();
        let trigger = store
            .create_at(
                WorkflowTriggerSpec {
                    name: "device denied".to_string(),
                    workflow_id: Uuid::new_v4().to_string(),
                    device_id: "target-device".to_string(),
                    overrides: HashMap::new(),
                    stop_on_failure: true,
                    interval_seconds: None,
                    cron_expression: None,
                    event_type: Some("device.access.denied".to_string()),
                    event_device_id: None,
                    event_filter: Some(json!({"reason": "policy"})),
                    callback_url: None,
                },
                now,
            )
            .unwrap();

        assert_eq!(trigger.event_device_id.as_deref(), Some("target-device"));
        assert!(trigger.next_run_at.is_none());

        assert!(store
            .claim_event(&trigger.id, Some("evt-1"), 42, now)
            .unwrap());
        let loaded = store.get(&trigger.id).unwrap();
        assert_eq!(loaded.last_event_id.as_deref(), Some("evt-1"));
        assert_eq!(loaded.last_server_rev, 42);

        assert!(!store
            .claim_event(&trigger.id, Some("evt-1"), 42, now)
            .unwrap());
        assert!(store
            .claim_event(&trigger.id, Some("evt-2"), 1, now)
            .unwrap());
        assert!(!store.claim_event(&trigger.id, None, 1, now).unwrap());
        assert!(store.claim_event(&trigger.id, None, 2, now).unwrap());

        let loaded = store.get(&trigger.id).unwrap();
        assert!(loaded.last_event_id.is_none());
        assert_eq!(loaded.last_server_rev, 2);

        assert!(store
            .claim_event(&trigger.id, Some("evt-3"), 2, now)
            .unwrap());

        let task_trigger = store
            .create_at(
                WorkflowTriggerSpec {
                    name: "scheduled failure".to_string(),
                    workflow_id: Uuid::new_v4().to_string(),
                    device_id: "target-device".to_string(),
                    overrides: HashMap::new(),
                    stop_on_failure: true,
                    interval_seconds: None,
                    cron_expression: None,
                    event_type: Some("scheduled_task.failed".to_string()),
                    event_device_id: None,
                    event_filter: Some(json!({"kind": "sign_out_all_sessions"})),
                    callback_url: None,
                },
                now,
            )
            .unwrap();
        assert!(task_trigger.event_device_id.is_none());

        let _ = std::fs::remove_dir_all(dir);
    }

    #[test]
    fn rejects_unsupported_event_types_and_sensitive_event_filters() {
        let dir = std::env::temp_dir().join(format!("quickdesk-trigger-store-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&dir).unwrap();
        let store = WorkflowTriggerStore::with_storage_dir(dir.clone());
        let base = WorkflowTriggerSpec {
            name: "event trigger".to_string(),
            workflow_id: Uuid::new_v4().to_string(),
            device_id: "target-device".to_string(),
            overrides: HashMap::new(),
            stop_on_failure: true,
            interval_seconds: None,
            cron_expression: None,
            event_type: Some("device.access.denied".to_string()),
            event_device_id: None,
            event_filter: Some(json!({"reason": "policy"})),
            callback_url: None,
        };

        assert!(store.create(base.clone()).is_ok());

        let mut unsupported = base.clone();
        unsupported.event_type = Some("device.access_code.changed".to_string());
        assert!(store.create(unsupported).is_err());

        let mut non_object_filter = base.clone();
        non_object_filter.event_filter = Some(json!("policy"));
        assert!(store.create(non_object_filter).is_err());

        let mut sensitive_filter = base;
        sensitive_filter.event_filter = Some(json!({"access_code": "012345"}));
        assert!(store.create(sensitive_filter).is_err());

        let _ = std::fs::remove_dir_all(dir);
    }
}
