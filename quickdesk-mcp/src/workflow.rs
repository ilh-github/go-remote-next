use chrono::Utc;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::collections::HashMap;
use std::path::PathBuf;
use std::sync::{Arc, Mutex};
use tracing::info;
use uuid::Uuid;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WorkflowStep {
    pub seq: u32,
    pub tool_name: String,
    pub arguments: Value,
    pub delay_ms: u64,
    pub result_summary: String,
    pub success: bool,
    pub parameterized_keys: Vec<String>,
    #[serde(default)]
    pub executor: StepExecutor,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq, Default)]
#[serde(rename_all = "snake_case")]
pub enum StepExecutor {
    RemoteApi,
    #[default]
    SkillHost,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Workflow {
    #[serde(default = "default_workflow_format_version")]
    pub format_version: u32,
    pub id: String,
    pub name: String,
    pub description: String,
    pub device_id: String,
    pub steps: Vec<WorkflowStep>,
    pub tags: Vec<String>,
    pub created_at: String,
    pub updated_at: String,
}

struct RecordingState {
    workflow_id: String,
    name: String,
    device_id: String,
    steps: Vec<WorkflowStep>,
    started_at: String,
    last_step_time: std::time::Instant,
}

pub struct WorkflowStore {
    storage_dir: PathBuf,
    runs_dir: PathBuf,
    active_recordings: Arc<Mutex<HashMap<String, RecordingState>>>,
}

impl WorkflowStore {
    pub fn new() -> Result<Self, String> {
        let dir = Self::storage_dir();
        Self::from_storage_dir(dir)
    }

    fn from_storage_dir(dir: PathBuf) -> Result<Self, String> {
        std::fs::create_dir_all(&dir).map_err(|e| format!("cannot create workflow dir: {e}"))?;
        let runs_dir = dir.with_file_name("workflow-runs");
        std::fs::create_dir_all(&runs_dir)
            .map_err(|e| format!("cannot create workflow run dir: {e}"))?;
        info!("WorkflowStore at {}", dir.display());
        Ok(Self {
            storage_dir: dir,
            runs_dir,
            active_recordings: Arc::new(Mutex::new(HashMap::new())),
        })
    }

    #[cfg(test)]
    pub(crate) fn with_storage_dir(dir: PathBuf) -> Self {
        Self::from_storage_dir(dir).expect("test workflow store")
    }

    fn storage_dir() -> PathBuf {
        let base = dirs_data_dir().unwrap_or_else(|| PathBuf::from("."));
        base.join(".quickdesk").join("workflows")
    }

    pub fn start_recording(&self, name: &str, device_id: &str) -> Result<String, String> {
        let mut recordings = self.active_recordings.lock().unwrap();
        if recordings.contains_key(device_id) {
            return Err("a recording is already active for this device".to_string());
        }

        let id = Uuid::new_v4().to_string();
        let state = RecordingState {
            workflow_id: id.clone(),
            name: name.to_string(),
            device_id: device_id.to_string(),
            steps: Vec::new(),
            started_at: Utc::now().to_rfc3339(),
            last_step_time: std::time::Instant::now(),
        };
        recordings.insert(device_id.to_string(), state);
        info!("recording started: id={id} name={name} device_id={device_id}");
        Ok(id)
    }

    pub fn is_recording(&self, device_id: &str) -> bool {
        self.active_recordings
            .lock()
            .unwrap()
            .contains_key(device_id)
    }

    pub fn record_step(
        &self,
        device_id: &str,
        tool_name: &str,
        arguments: &Value,
        result_summary: &str,
        success: bool,
        executor: StepExecutor,
    ) {
        let mut recordings = self.active_recordings.lock().unwrap();
        if let Some(state) = recordings.get_mut(device_id) {
            let now = std::time::Instant::now();
            let delay = now.duration_since(state.last_step_time).as_millis() as u64;
            state.last_step_time = now;

            let seq = state.steps.len() as u32 + 1;
            let param_keys = detect_parameterizable(arguments);
            state.steps.push(WorkflowStep {
                seq,
                tool_name: tool_name.to_string(),
                arguments: arguments.clone(),
                delay_ms: delay,
                result_summary: truncate(result_summary, 512),
                success,
                parameterized_keys: param_keys,
                executor,
            });
        }
    }

    pub fn stop_recording(
        &self,
        device_id: &str,
        description: &str,
        tags: &[String],
    ) -> Result<Workflow, String> {
        let mut recordings = self.active_recordings.lock().unwrap();
        let state = recordings
            .remove(device_id)
            .ok_or_else(|| "no active recording for this device".to_string())?;

        if state.steps.is_empty() {
            return Err("no steps were recorded".to_string());
        }

        let now = Utc::now().to_rfc3339();
        let wf = Workflow {
            format_version: default_workflow_format_version(),
            id: state.workflow_id,
            name: state.name,
            description: description.to_string(),
            device_id: state.device_id,
            steps: state.steps,
            tags: tags.to_vec(),
            created_at: state.started_at,
            updated_at: now,
        };

        self.save_workflow(&wf)?;
        info!("recording stopped: id={} steps={}", wf.id, wf.steps.len());
        Ok(wf)
    }

    pub fn list_workflows(&self) -> Vec<WorkflowSummary> {
        let mut results = Vec::new();
        let entries = match std::fs::read_dir(&self.storage_dir) {
            Ok(e) => e,
            Err(_) => return results,
        };

        for entry in entries.flatten() {
            let path = entry.path();
            if path.extension().map(|e| e == "json").unwrap_or(false) {
                if let Ok(data) = std::fs::read_to_string(&path) {
                    if let Ok(wf) = serde_json::from_str::<Workflow>(&data) {
                        results.push(WorkflowSummary {
                            id: wf.id,
                            name: wf.name,
                            description: wf.description,
                            device_id: wf.device_id,
                            step_count: wf.steps.len() as u32,
                            tags: wf.tags,
                            created_at: wf.created_at,
                        });
                    }
                }
            }
        }

        results.sort_by(|a, b| b.created_at.cmp(&a.created_at));
        results
    }

    pub fn get_workflow(&self, id: &str) -> Option<Workflow> {
        let path = self.workflow_path(id)?;
        let data = std::fs::read_to_string(&path).ok()?;
        serde_json::from_str(&data).ok()
    }

    pub fn delete_workflow(&self, id: &str) -> bool {
        let Some(path) = self.workflow_path(id) else {
            return false;
        };
        std::fs::remove_file(&path).is_ok()
    }

    pub fn build_replay_steps(
        &self,
        workflow_id: &str,
        overrides: &HashMap<String, Value>,
    ) -> Result<Vec<ReplayStep>, String> {
        let wf = self
            .get_workflow(workflow_id)
            .ok_or_else(|| "workflow not found".to_string())?;

        let mut steps = Vec::new();
        for step in &wf.steps {
            let arguments = substitute_parameters(&step.arguments, overrides);
            steps.push(ReplayStep {
                seq: step.seq,
                tool_name: step.tool_name.clone(),
                arguments: apply_top_level_overrides(arguments, overrides),
                executor: step.executor,
            });
        }
        Ok(steps)
    }

    fn save_workflow(&self, wf: &Workflow) -> Result<(), String> {
        let path = self
            .workflow_path(&wf.id)
            .ok_or_else(|| "invalid workflow id".to_string())?;
        let data = serde_json::to_string_pretty(wf).map_err(|e| format!("serialize: {e}"))?;
        std::fs::write(&path, data).map_err(|e| format!("write: {e}"))
    }

    fn workflow_path(&self, id: &str) -> Option<PathBuf> {
        validate_workflow_id(id).ok()?;
        Some(self.storage_dir.join(format!("{id}.json")))
    }

    pub fn save_run_report(&self, report: &WorkflowRunReport) -> Result<(), String> {
        validate_workflow_id(&report.run_id)?;
        let data = serde_json::to_string_pretty(report)
            .map_err(|e| format!("serialize workflow run: {e}"))?;
        let path = self.runs_dir.join(format!("{}.json", report.run_id));
        std::fs::write(path, data).map_err(|e| format!("write workflow run: {e}"))
    }

    pub fn get_run_report(&self, run_id: &str) -> Option<WorkflowRunReport> {
        validate_workflow_id(run_id).ok()?;
        let data = std::fs::read_to_string(self.runs_dir.join(format!("{run_id}.json"))).ok()?;
        serde_json::from_str(&data).ok()
    }

    pub fn list_run_reports(
        &self,
        workflow_id: Option<&str>,
        limit: usize,
    ) -> Vec<WorkflowRunSummary> {
        if limit == 0 {
            return Vec::new();
        }

        let mut reports: Vec<WorkflowRunReport> = Vec::new();
        let Ok(entries) = std::fs::read_dir(&self.runs_dir) else {
            return Vec::new();
        };

        for entry in entries.flatten() {
            let path = entry.path();
            if !path.extension().map(|ext| ext == "json").unwrap_or(false) {
                continue;
            }
            let Ok(data) = std::fs::read_to_string(&path) else {
                continue;
            };
            let Ok(report) = serde_json::from_str::<WorkflowRunReport>(&data) else {
                continue;
            };
            if workflow_id.is_some_and(|id| report.workflow_id != id) {
                continue;
            }
            reports.push(report);
        }

        reports.sort_by(|a, b| {
            b.completed_at
                .cmp(&a.completed_at)
                .then_with(|| b.run_id.cmp(&a.run_id))
        });
        reports
            .into_iter()
            .take(limit)
            .map(WorkflowRunSummary::from)
            .collect()
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WorkflowSummary {
    pub id: String,
    pub name: String,
    pub description: String,
    pub device_id: String,
    pub step_count: u32,
    pub tags: Vec<String>,
    pub created_at: String,
}

#[derive(Debug, Clone, PartialEq)]
pub struct ReplayStep {
    pub seq: u32,
    pub tool_name: String,
    pub arguments: Value,
    pub executor: StepExecutor,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct WorkflowRunReport {
    pub run_id: String,
    pub workflow_id: String,
    pub device_id: String,
    pub started_at: String,
    pub completed_at: String,
    pub total_steps: u32,
    pub succeeded_steps: u32,
    pub failed_steps: u32,
    pub skipped_steps: u32,
    pub success: bool,
    pub stopped_early: bool,
    pub steps: Vec<StepRunResult>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct WorkflowRunSummary {
    pub run_id: String,
    pub workflow_id: String,
    pub device_id: String,
    pub started_at: String,
    pub completed_at: String,
    pub total_steps: u32,
    pub succeeded_steps: u32,
    pub failed_steps: u32,
    pub skipped_steps: u32,
    pub success: bool,
    pub stopped_early: bool,
}

impl From<WorkflowRunReport> for WorkflowRunSummary {
    fn from(report: WorkflowRunReport) -> Self {
        Self {
            run_id: report.run_id,
            workflow_id: report.workflow_id,
            device_id: report.device_id,
            started_at: report.started_at,
            completed_at: report.completed_at,
            total_steps: report.total_steps,
            succeeded_steps: report.succeeded_steps,
            failed_steps: report.failed_steps,
            skipped_steps: report.skipped_steps,
            success: report.success,
            stopped_early: report.stopped_early,
        }
    }
}

impl WorkflowRunReport {
    pub fn from_step_results(
        workflow_id: &str,
        device_id: &str,
        started_at: &str,
        completed_at: &str,
        total_steps: u32,
        steps: Vec<StepRunResult>,
        stopped_early: bool,
    ) -> Self {
        let succeeded_steps = steps.iter().filter(|step| step.success).count() as u32;
        let failed_steps = steps.iter().filter(|step| !step.success).count() as u32;
        let skipped_steps = total_steps.saturating_sub(steps.len() as u32);
        Self {
            run_id: Uuid::new_v4().to_string(),
            workflow_id: workflow_id.to_string(),
            device_id: device_id.to_string(),
            started_at: started_at.to_string(),
            completed_at: completed_at.to_string(),
            total_steps,
            succeeded_steps,
            failed_steps,
            skipped_steps,
            success: failed_steps == 0 && skipped_steps == 0,
            stopped_early,
            steps,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct StepRunResult {
    pub seq: u32,
    pub tool_name: String,
    pub executor: StepExecutor,
    pub success: bool,
    pub result: Value,
    pub error: String,
    pub duration_ms: u64,
}

fn detect_parameterizable(args: &Value) -> Vec<String> {
    let mut keys = Vec::new();
    if let Value::Object(map) = args {
        for (k, v) in map {
            match v {
                Value::String(s) if looks_like_path(s) => keys.push(k.clone()),
                Value::String(s) if s.len() > 100 => keys.push(k.clone()),
                _ => {}
            }
        }
    }
    keys
}

fn looks_like_path(s: &str) -> bool {
    s.starts_with('/') || s.starts_with("C:\\") || s.starts_with("D:\\") || s.contains("\\")
}

fn truncate(s: &str, max: usize) -> String {
    if s.len() > max {
        format!("{}...", &s[..max])
    } else {
        s.to_string()
    }
}

fn default_workflow_format_version() -> u32 {
    1
}

fn substitute_parameters(value: &Value, overrides: &HashMap<String, Value>) -> Value {
    match value {
        Value::String(text) => {
            if let Some(key) = text
                .strip_prefix("{{")
                .and_then(|rest| rest.strip_suffix("}}"))
            {
                if let Some(replacement) = overrides.get(key.trim()) {
                    return replacement.clone();
                }
            }

            let mut rendered = text.clone();
            for (key, replacement) in overrides {
                let placeholder = format!("{{{{{key}}}}}");
                if !rendered.contains(&placeholder) {
                    continue;
                }
                let replacement = match replacement {
                    Value::String(s) => s.clone(),
                    other => other.to_string(),
                };
                rendered = rendered.replace(&placeholder, &replacement);
            }
            Value::String(rendered)
        }
        Value::Array(items) => Value::Array(
            items
                .iter()
                .map(|item| substitute_parameters(item, overrides))
                .collect(),
        ),
        Value::Object(map) => Value::Object(
            map.iter()
                .map(|(key, value)| (key.clone(), substitute_parameters(value, overrides)))
                .collect(),
        ),
        other => other.clone(),
    }
}

fn apply_top_level_overrides(value: Value, overrides: &HashMap<String, Value>) -> Value {
    let Value::Object(mut map) = value else {
        return value;
    };

    for (key, replacement) in overrides {
        if map.contains_key(key) {
            map.insert(key.clone(), replacement.clone());
        }
    }

    Value::Object(map)
}

fn validate_workflow_id(id: &str) -> Result<(), String> {
    Uuid::parse_str(id)
        .map(|_| ())
        .map_err(|_| "invalid workflow id".to_string())
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

    #[test]
    fn substitutes_nested_placeholders_and_preserves_override_types() {
        let arguments = serde_json::json!({
            "path": "{{target}}",
            "nested": {
                "count": "{{count}}",
                "label": "save {{name}}"
            },
            "list": ["{{name}}", 7]
        });
        let overrides = HashMap::from([
            ("target".to_string(), serde_json::json!("/tmp/new")),
            ("count".to_string(), serde_json::json!(3)),
            ("name".to_string(), serde_json::json!("report")),
        ]);

        let substituted = substitute_parameters(&arguments, &overrides);

        assert_eq!(
            substituted,
            serde_json::json!({
                "path": "/tmp/new",
                "nested": {
                    "count": 3,
                    "label": "save report"
                },
                "list": ["report", 7]
            })
        );
    }

    #[test]
    fn rejects_workflow_ids_that_can_escape_the_storage_directory() {
        assert!(validate_workflow_id("../../credentials").is_err());
        assert!(validate_workflow_id("not-a-uuid").is_err());
        assert!(validate_workflow_id("b7ca0a2f-0d4c-4f0c-9dbf-4a92e8a67d10").is_ok());
    }

    #[test]
    fn records_which_transport_each_step_must_replay_through() {
        let dir = std::env::temp_dir().join(format!("quickdesk-workflow-test-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&dir).unwrap();
        let store = WorkflowStore::with_storage_dir(dir.clone());

        let id = store.start_recording("mixed workflow", "device-1").unwrap();
        store.record_step(
            "device-1",
            "mouse_click",
            &serde_json::json!({"x": 1, "y": 2}),
            "clicked",
            true,
            StepExecutor::RemoteApi,
        );
        store.record_step(
            "device-1",
            "run_command",
            &serde_json::json!({"command": "echo ok"}),
            "ok",
            true,
            StepExecutor::SkillHost,
        );
        let workflow = store.stop_recording("device-1", "", &[]).unwrap();

        assert_eq!(workflow.id, id);
        assert_eq!(workflow.steps[0].executor, StepExecutor::RemoteApi);
        assert_eq!(workflow.steps[1].executor, StepExecutor::SkillHost);

        let _ = std::fs::remove_dir_all(dir);
    }

    #[test]
    fn replay_plan_uses_nested_overrides_and_recorded_transport() {
        let dir = std::env::temp_dir().join(format!("quickdesk-workflow-test-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&dir).unwrap();
        let store = WorkflowStore::with_storage_dir(dir.clone());

        let id = store.start_recording("parameterized", "device-1").unwrap();
        store.record_step(
            "device-1",
            "keyboard_type",
            &serde_json::json!({"text": "open {{path}}"}),
            "typed",
            true,
            StepExecutor::RemoteApi,
        );
        store.stop_recording("device-1", "", &[]).unwrap();
        let overrides = HashMap::from([(
            "path".to_string(),
            serde_json::json!("C:\\reports\\daily.txt"),
        )]);

        let plan = store.build_replay_steps(&id, &overrides).unwrap();

        assert_eq!(plan[0].executor, StepExecutor::RemoteApi);
        assert_eq!(
            plan[0].arguments,
            serde_json::json!({"text": "open C:\\reports\\daily.txt"})
        );

        let _ = std::fs::remove_dir_all(dir);
    }

    #[test]
    fn replay_plan_keeps_legacy_top_level_argument_overrides() {
        let dir = std::env::temp_dir().join(format!("quickdesk-workflow-test-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&dir).unwrap();
        let store = WorkflowStore::with_storage_dir(dir.clone());

        let id = store
            .start_recording("legacy overrides", "device-1")
            .unwrap();
        store.record_step(
            "device-1",
            "keyboard_type",
            &serde_json::json!({"text": "old"}),
            "typed",
            true,
            StepExecutor::RemoteApi,
        );
        store.stop_recording("device-1", "", &[]).unwrap();
        let overrides = HashMap::from([("text".to_string(), serde_json::json!("new"))]);

        let plan = store.build_replay_steps(&id, &overrides).unwrap();

        assert_eq!(plan[0].arguments, serde_json::json!({"text": "new"}));

        let _ = std::fs::remove_dir_all(dir);
    }

    #[test]
    fn persists_replay_reports_with_step_failures() {
        let dir = std::env::temp_dir().join(format!("quickdesk-workflow-test-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&dir).unwrap();
        let store = WorkflowStore::with_storage_dir(dir.clone());
        let run_id = Uuid::new_v4().to_string();
        let report = WorkflowRunReport {
            run_id: run_id.clone(),
            workflow_id: "b7ca0a2f-0d4c-4f0c-9dbf-4a92e8a67d10".to_string(),
            device_id: "device-1".to_string(),
            started_at: "2026-10-17T14:00:00Z".to_string(),
            completed_at: "2026-10-17T14:00:02Z".to_string(),
            total_steps: 2,
            succeeded_steps: 1,
            failed_steps: 1,
            skipped_steps: 0,
            success: false,
            stopped_early: true,
            steps: vec![
                StepRunResult {
                    seq: 1,
                    tool_name: "mouse_click".to_string(),
                    executor: StepExecutor::RemoteApi,
                    success: true,
                    result: serde_json::json!({"ok": true}),
                    error: String::new(),
                    duration_ms: 12,
                },
                StepRunResult {
                    seq: 2,
                    tool_name: "run_command".to_string(),
                    executor: StepExecutor::SkillHost,
                    success: false,
                    result: serde_json::Value::Null,
                    error: "exit 1".to_string(),
                    duration_ms: 30,
                },
            ],
        };

        store.save_run_report(&report).unwrap();
        let loaded = store.get_run_report(&run_id).unwrap();

        assert_eq!(loaded, report);
        let _ = std::fs::remove_dir_all(dir);
    }

    #[test]
    fn public_workflow_lookup_cannot_read_outside_the_storage_directory() {
        let base = std::env::temp_dir().join(format!("quickdesk-workflow-test-{}", Uuid::new_v4()));
        let workflows = base.join("workflows");
        std::fs::create_dir_all(&workflows).unwrap();
        let store = WorkflowStore::with_storage_dir(workflows);

        let id = store.start_recording("safe", "device-1").unwrap();
        store.record_step(
            "device-1",
            "keyboard_type",
            &serde_json::json!({"text": "ok"}),
            "typed",
            true,
            StepExecutor::RemoteApi,
        );
        let workflow = store.stop_recording("device-1", "", &[]).unwrap();
        std::fs::write(
            base.join("evil.json"),
            serde_json::to_string(&workflow).unwrap(),
        )
        .unwrap();

        assert!(store.get_workflow("../evil").is_none());
        assert_eq!(store.get_workflow(&id).unwrap().name, "safe");

        let _ = std::fs::remove_dir_all(base);
    }

    #[test]
    fn run_report_summarizes_success_failure_and_skipped_steps() {
        let results = vec![
            StepRunResult {
                seq: 1,
                tool_name: "mouse_click".to_string(),
                executor: StepExecutor::RemoteApi,
                success: true,
                result: serde_json::json!({"ok": true}),
                error: String::new(),
                duration_ms: 10,
            },
            StepRunResult {
                seq: 2,
                tool_name: "run_command".to_string(),
                executor: StepExecutor::SkillHost,
                success: false,
                result: serde_json::Value::Null,
                error: "exit 1".to_string(),
                duration_ms: 20,
            },
        ];

        let report = WorkflowRunReport::from_step_results(
            "workflow-1",
            "device-1",
            "2026-10-17T14:00:00Z",
            "2026-10-17T14:00:01Z",
            3,
            results,
            true,
        );

        assert_eq!(report.succeeded_steps, 1);
        assert_eq!(report.failed_steps, 1);
        assert_eq!(report.skipped_steps, 1);
        assert!(!report.success);
        assert!(report.stopped_early);
    }

    #[test]
    fn lists_newest_workflow_runs_with_filter_and_limit() {
        let dir = std::env::temp_dir().join(format!("quickdesk-workflow-test-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&dir).unwrap();
        let store = WorkflowStore::with_storage_dir(dir.clone());
        let workflow_a = Uuid::new_v4().to_string();
        let workflow_b = Uuid::new_v4().to_string();

        for (workflow_id, completed_at) in [
            (&workflow_a, "2026-10-17T10:00:00Z"),
            (&workflow_a, "2026-10-17T12:00:00Z"),
            (&workflow_b, "2026-10-17T11:00:00Z"),
        ] {
            let report = WorkflowRunReport::from_step_results(
                workflow_id,
                "device-1",
                "2026-10-17T09:00:00Z",
                completed_at,
                0,
                Vec::new(),
                false,
            );
            store.save_run_report(&report).unwrap();
        }

        let runs = store.list_run_reports(Some(&workflow_a), 1);

        assert_eq!(runs.len(), 1);
        assert_eq!(runs[0].workflow_id, workflow_a);
        assert_eq!(runs[0].completed_at, "2026-10-17T12:00:00Z");

        let _ = std::fs::remove_dir_all(dir);
    }

    #[test]
    fn loads_pre_versioned_workflows_with_skill_host_defaults() {
        let legacy = serde_json::json!({
            "id": "b7ca0a2f-0d4c-4f0c-9dbf-4a92e8a67d10",
            "name": "legacy",
            "description": "",
            "device_id": "device-1",
            "steps": [{
                "seq": 1,
                "tool_name": "run_command",
                "arguments": {"command": "echo ok"},
                "delay_ms": 0,
                "result_summary": "ok",
                "success": true,
                "parameterized_keys": []
            }],
            "tags": [],
            "created_at": "2026-10-17T10:00:00Z",
            "updated_at": "2026-10-17T10:00:01Z"
        });

        let workflow: Workflow = serde_json::from_value(legacy).unwrap();

        assert_eq!(workflow.format_version, 1);
        assert_eq!(workflow.steps[0].executor, StepExecutor::SkillHost);
    }
}
