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
    /// Optional pre-condition verified before the step is replayed. A missing or
    /// stale screen state means the replay would click the wrong thing, so the
    /// step is reported as pending instead of being executed.
    #[serde(default)]
    pub expect: Option<StepExpectation>,
}

/// A pre-condition on the remote screen, checked before a replay step runs.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct StepExpectation {
    /// `screen_hash` or `screen_contains`.
    pub kind: String,
    /// The expected hash, or the text that must be on screen.
    pub value: String,
    /// How many times the check is retried before the step is parked.
    #[serde(default = "default_expectation_retries")]
    pub retries: u32,
}

fn default_expectation_retries() -> u32 {
    1
}

/// Outcome of checking one pre-condition against the observed screen.
#[derive(Debug, Clone, PartialEq)]
pub enum ExpectationOutcome {
    Matched,
    Mismatched { expected: String, observed: String },
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
    /// Where parked runs leave their handover. Sits beside the run reports so
    /// a store and its resume points always move together.
    resume_points: PathBuf,
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
        let resume_dir = dir.with_file_name("workflow-resume");
        std::fs::create_dir_all(&resume_dir)
            .map_err(|e| format!("cannot create workflow resume dir: {e}"))?;
        info!("WorkflowStore at {}", dir.display());
        Ok(Self {
            storage_dir: dir,
            runs_dir,
            resume_points: resume_dir.join("resume-points.json"),
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
                expect: None,
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

    /// Attach or clear the screen pre-condition on one step.
    ///
    /// An empty or missing `kind` clears the expectation so a workflow recorded
    /// without one keeps replaying unguarded. Returns the updated step.
    pub fn set_step_expectation(
        &self,
        workflow_id: &str,
        seq: u32,
        kind: Option<&str>,
        value: Option<&str>,
        retries: Option<u32>,
    ) -> Result<WorkflowStep, String> {
        let mut wf = self
            .get_workflow(workflow_id)
            .ok_or_else(|| "workflow not found".to_string())?;

        let position = wf
            .steps
            .iter()
            .position(|step| step.seq == seq)
            .ok_or_else(|| format!("step {seq} not found in workflow {workflow_id}"))?;

        let kind = kind.unwrap_or("").trim();
        let value = value.unwrap_or("").trim();
        wf.steps[position].expect = if kind.is_empty() {
            None
        } else {
            if !matches!(kind, "screen_hash" | "screen_contains") {
                return Err(format!(
                    "unsupported expectation kind: {kind} (use screen_hash or screen_contains)"
                ));
            }
            if value.is_empty() {
                return Err("an expectation needs a value".to_string());
            }
            Some(StepExpectation {
                kind: kind.to_string(),
                value: value.to_string(),
                retries: retries.unwrap_or(1).clamp(1, 10),
            })
        };

        self.save_workflow(&wf)?;
        Ok(wf.steps[position].clone())
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
                expect: step.expect.clone(),
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

    /// Leaves a handover for a parked run.
    ///
    /// Called when a replay parks on a screen mismatch. A failure here is not
    /// fatal to the run - the report is already saved - but losing the
    /// handover costs the operator their position, so it is reported.
    pub fn record_resume_point(
        &self,
        point: &crate::recovery::ResumePoint,
    ) -> Result<(), String> {
        let mut points = self.load_resume_points()?;
        points.retain(|existing| existing.run_id != point.run_id);
        points.push(point.clone());
        self.write_resume_points(&points)
    }

    /// The most recent handover, for an operator asking where a parked run
    /// stopped.
    pub fn latest_resume_point(&self) -> Option<crate::recovery::ResumePoint> {
        let mut points = self.load_resume_points().ok()?;
        points.sort_by(|left, right| left.run_id.cmp(&right.run_id));
        points.pop()
    }

    /// Drops the handover for a run that has been resumed or abandoned.
    pub fn clear_resume_point(&self, run_id: &str) {
        let Ok(mut points) = self.load_resume_points() else {
            return;
        };
        let before = points.len();
        points.retain(|existing| existing.run_id != run_id);
        if points.len() != before {
            let _ = self.write_resume_points(&points);
        }
    }

    fn load_resume_points(&self) -> Result<Vec<crate::recovery::ResumePoint>, String> {
        if !self.resume_points.exists() {
            return Ok(Vec::new());
        }
        let data = std::fs::read_to_string(&self.resume_points)
            .map_err(|error| format!("read resume points: {error}"))?;
        serde_json::from_str(&data).map_err(|error| format!("parse resume points: {error}"))
    }

    fn write_resume_points(&self, points: &[crate::recovery::ResumePoint]) -> Result<(), String> {
        let json = serde_json::to_string_pretty(points)
            .map_err(|error| format!("encode resume points: {error}"))?;
        std::fs::write(&self.resume_points, json)
            .map_err(|error| format!("write resume points: {error}"))
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
    pub expect: Option<StepExpectation>,
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
    #[serde(default)]
    pub cancelled: bool,
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
    #[serde(default)]
    pub cancelled: bool,
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
            cancelled: report.cancelled,
        }
    }
}

impl WorkflowRunReport {
    #[allow(clippy::too_many_arguments)]
    pub fn from_step_results_with_id(
        run_id: String,
        workflow_id: &str,
        device_id: &str,
        started_at: &str,
        completed_at: &str,
        total_steps: u32,
        steps: Vec<StepRunResult>,
        stopped_early: bool,
        cancelled: bool,
    ) -> Self {
        let succeeded_steps = steps.iter().filter(|step| step.success).count() as u32;
        let failed_steps = steps.iter().filter(|step| !step.success).count() as u32;
        let skipped_steps = total_steps.saturating_sub(steps.len() as u32);
        Self {
            run_id,
            workflow_id: workflow_id.to_string(),
            device_id: device_id.to_string(),
            started_at: started_at.to_string(),
            completed_at: completed_at.to_string(),
            total_steps,
            succeeded_steps,
            failed_steps,
            skipped_steps,
            success: failed_steps == 0 && skipped_steps == 0 && !cancelled,
            stopped_early,
            cancelled,
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
    /// The step was not executed because its screen expectation did not match.
    #[serde(default)]
    pub pending: bool,
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

/// Check one pre-condition against what is currently on the remote screen.
///
/// `observed_hash` and `observed_text` describe the same screen; only the field
/// the expectation needs is consulted. A malformed expectation is reported as a
/// mismatch so the replay parks the step instead of guessing.
pub fn evaluate_expectation(
    expect: &StepExpectation,
    observed_hash: &str,
    observed_text: &str,
) -> ExpectationOutcome {
    let expected = expect.value.trim();
    if expected.is_empty() {
        return ExpectationOutcome::Mismatched {
            expected: String::new(),
            observed: describe_observation(expect, observed_hash, observed_text),
        };
    }

    let matched = match expect.kind.as_str() {
        "screen_hash" => !observed_hash.is_empty()
            && observed_hash.eq_ignore_ascii_case(expected),
        "screen_contains" => {
            !observed_text.is_empty() && observed_text.contains(expected)
        }
        _ => false,
    };

    if matched {
        ExpectationOutcome::Matched
    } else {
        ExpectationOutcome::Mismatched {
            expected: expected.to_string(),
            observed: describe_observation(expect, observed_hash, observed_text),
        }
    }
}

fn describe_observation(expect: &StepExpectation, observed_hash: &str, observed_text: &str) -> String {
    match expect.kind.as_str() {
        "screen_contains" => {
            if observed_text.is_empty() {
                String::new()
            } else {
                truncate(observed_text, 120)
            }
        }
        _ => observed_hash.to_string(),
    }
}

/// Human-readable reason used in run reports when a step is parked.
pub fn expectation_mismatch_reason(
    seq: u32,
    outcome: &ExpectationOutcome,
    attempt: u32,
    retries: u32,
) -> String {
    match outcome {
        ExpectationOutcome::Matched => String::new(),
        ExpectationOutcome::Mismatched { expected, observed } => format!(
            "step {seq} screen state did not match after {attempt}/{retries} checks \
             (expected {expected:?}, observed {observed:?})"
        ),
    }
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
            cancelled: false,
            steps: vec![
                StepRunResult {
                    seq: 1,
                    tool_name: "mouse_click".to_string(),
                    executor: StepExecutor::RemoteApi,
                    success: true,
                    pending: false,
                    result: serde_json::json!({"ok": true}),
                    error: String::new(),
                    duration_ms: 12,
                },
                StepRunResult {
                    seq: 2,
                    tool_name: "run_command".to_string(),
                    executor: StepExecutor::SkillHost,
                    success: false,
                    pending: false,
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
                pending: false,
                result: serde_json::json!({"ok": true}),
                error: String::new(),
                duration_ms: 10,
            },
            StepRunResult {
                seq: 2,
                tool_name: "run_command".to_string(),
                executor: StepExecutor::SkillHost,
                success: false,
                pending: false,
                result: serde_json::Value::Null,
                error: "exit 1".to_string(),
                duration_ms: 20,
            },
        ];

        let report = WorkflowRunReport::from_step_results_with_id(
            Uuid::new_v4().to_string(),
            "workflow-1",
            "device-1",
            "2026-10-17T14:00:00Z",
            "2026-10-17T14:00:01Z",
            3,
            results,
            true,
            false,
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
            let report = WorkflowRunReport::from_step_results_with_id(
                Uuid::new_v4().to_string(),
                workflow_id,
                "device-1",
                "2026-10-17T09:00:00Z",
                completed_at,
                0,
                Vec::new(),
                false,
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

    fn expectation(kind: &str, value: &str) -> StepExpectation {
        StepExpectation {
            kind: kind.to_string(),
            value: value.to_string(),
            retries: 1,
        }
    }

    #[test]
    fn screen_expectations_match_hash_and_text() {
        let hash = expectation("screen_hash", "abc123");
        assert_eq!(
            evaluate_expectation(&hash, "ABC123", "anything"),
            ExpectationOutcome::Matched,
            "hash comparison is case-insensitive",
        );
        assert_eq!(
            evaluate_expectation(&hash, "", ""),
            ExpectationOutcome::Mismatched {
                expected: "abc123".to_string(),
                observed: String::new(),
            },
            "an empty observation never matches",
        );

        let text = expectation("screen_contains", "Save");
        assert_eq!(
            evaluate_expectation(&text, "ignored", "Do you want to Save changes?"),
            ExpectationOutcome::Matched,
        );
        assert_eq!(
            evaluate_expectation(&text, "ignored", "nothing here"),
            ExpectationOutcome::Mismatched {
                expected: "Save".to_string(),
                observed: "nothing here".to_string(),
            },
        );
    }

    #[test]
    fn malformed_expectations_are_reported_as_mismatches() {
        let unknown = expectation("screen_pixel", "10,20");
        assert_eq!(
            evaluate_expectation(&unknown, "abc", "text"),
            ExpectationOutcome::Mismatched {
                expected: "10,20".to_string(),
                observed: "abc".to_string(),
            },
            "an unsupported kind must not silently pass",
        );

        let empty = expectation("screen_contains", "   ");
        let outcome = evaluate_expectation(&empty, "abc", "text");
        assert_eq!(
            outcome,
            ExpectationOutcome::Mismatched {
                expected: String::new(),
                observed: "text".to_string(),
            },
        );

        let reason = expectation_mismatch_reason(3, &outcome, 2, 3);
        assert!(reason.contains("step 3"), "{reason}");
        assert!(reason.contains("2/3"), "{reason}");
    }

    #[test]
    fn replay_plan_carries_step_expectations() {
        let dir = std::env::temp_dir().join(format!("quickdesk-workflow-test-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&dir).unwrap();
        let store = WorkflowStore::with_storage_dir(dir.clone());

        let id = store.start_recording("guarded", "device-1").unwrap();
        store.record_step(
            "device-1",
            "mouse_click",
            &serde_json::json!({"x": 5, "y": 6}),
            "clicked",
            true,
            StepExecutor::RemoteApi,
        );
        store.stop_recording("device-1", "", &[]).unwrap();

        let mut workflow = store.get_workflow(&id).unwrap();
        workflow.steps[0].expect = Some(expectation("screen_contains", "Confirm"));
        store.save_workflow(&workflow).unwrap();

        let plan = store.build_replay_steps(&id, &HashMap::new()).unwrap();
        assert_eq!(plan[0].expect, Some(expectation("screen_contains", "Confirm")));

        let _ = std::fs::remove_dir_all(dir);
    }

    #[test]
    fn legacy_workflows_without_expectations_replay_unguarded() {
        let dir = std::env::temp_dir().join(format!("quickdesk-workflow-test-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&dir).unwrap();
        let store = WorkflowStore::with_storage_dir(dir.clone());

        let id = store.start_recording("legacy", "device-1").unwrap();
        store.record_step(
            "device-1",
            "run_command",
            &serde_json::json!({"command": "echo ok"}),
            "ok",
            true,
            StepExecutor::SkillHost,
        );
        store.stop_recording("device-1", "", &[]).unwrap();

        let plan = store.build_replay_steps(&id, &HashMap::new()).unwrap();
        assert_eq!(plan[0].expect, None);

        let _ = std::fs::remove_dir_all(dir);
    }

    #[test]
    fn step_expectations_can_be_set_validated_and_cleared() {
        let dir = std::env::temp_dir().join(format!("quickdesk-workflow-test-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&dir).unwrap();
        let store = WorkflowStore::with_storage_dir(dir.clone());

        let id = store.start_recording("guarded", "device-1").unwrap();
        store.record_step(
            "device-1",
            "mouse_click",
            &serde_json::json!({"x": 1, "y": 2}),
            "clicked",
            true,
            StepExecutor::RemoteApi,
        );
        store.stop_recording("device-1", "", &[]).unwrap();

        let step = store
            .set_step_expectation(&id, 1, Some("screen_contains"), Some(" Confirm "), Some(3))
            .unwrap();
        assert_eq!(
            step.expect,
            Some(StepExpectation {
                kind: "screen_contains".to_string(),
                value: "Confirm".to_string(),
                retries: 3,
            }),
        );

        // Reload from disk: the guard must survive a restart.
        let reloaded = store.get_workflow(&id).unwrap();
        assert!(reloaded.steps[0].expect.is_some());

        assert!(
            store
                .set_step_expectation(&id, 1, Some("screen_pixel"), Some("1,2"), None)
                .is_err(),
            "an unsupported kind must be rejected",
        );
        assert!(
            store
                .set_step_expectation(&id, 1, Some("screen_hash"), Some("  "), None)
                .is_err(),
            "an expectation without a value must be rejected",
        );
        assert!(
            store
                .set_step_expectation(&id, 9, Some("screen_hash"), Some("abc"), None)
                .is_err(),
            "an unknown step must be rejected",
        );

        let cleared = store.set_step_expectation(&id, 1, None, None, None).unwrap();
        assert_eq!(cleared.expect, None);

        let _ = std::fs::remove_dir_all(dir);
    }
}
