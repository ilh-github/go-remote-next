// Cross-device command chaining: run a step on one set of machines, then run
// the next step on another set only if the first went the way the caller asked.
//
// `batch_skill_exec` answers "the same command, everywhere, at once". This
// answers the other question, which a batch cannot: "do X on the servers, and
// only if that worked, do Y on the desktops". An operator orchestrating a change
// across tiers currently has to read the batch report, decide, and issue the
// second call by hand - which means the decision lives in a conversation rather
// than in the automation, and a fleet-wide change is only as repeatable as the
// person who remembers the second half.

use serde_json::{json, Value};

/// One step of a chain: which machines, which tool, and what has to be true of
/// the previous step for this one to run.
#[derive(Debug, Clone, PartialEq)]
pub struct ChainStep {
    /// Devices this step runs on. Empty means "every device the previous step
    /// succeeded on", which is the shape that makes a chain useful without
    /// restating the fleet.
    pub device_ids: Vec<String>,
    pub tool: String,
    pub args: Value,
    /// What the previous step must have produced for this one to run at all.
    pub when: StepCondition,
}

/// When a step runs, given how the previous one went.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum StepCondition {
    /// Run regardless. Only meaningful on the first step, where there is
    /// nothing to condition on.
    Always,
    /// Run only if every device in the previous step succeeded.
    OnAllSucceeded,
    /// Run if at least one device succeeded, and only on those.
    OnAnySucceeded,
    /// Run even if the previous step failed outright - the cleanup case.
    OnFailure,
}

/// What one step actually did, in a shape the next step and the operator can
/// both read.
#[derive(Debug, Clone, PartialEq)]
pub struct StepOutcome {
    pub tool: String,
    /// Devices this step ran on, in the order the caller listed them.
    pub device_ids: Vec<String>,
    /// Devices that acknowledged. Empty when the step was skipped.
    pub succeeded: Vec<String>,
    /// Devices that did not, each with why.
    pub failures: Vec<(String, String)>,
    /// True when the step did not run because its condition was not met.
    pub skipped: bool,
    pub skip_reason: String,
}

impl StepOutcome {
    pub fn ran(tool: &str, device_ids: Vec<String>) -> Self {
        Self {
            tool: tool.to_string(),
            device_ids,
            succeeded: Vec::new(),
            failures: Vec::new(),
            skipped: false,
            skip_reason: String::new(),
        }
    }

    /// A step that did not run, with the reason it did not.
    pub fn skipped(tool: &str, reason: &str) -> Self {
        Self {
            tool: tool.to_string(),
            device_ids: Vec::new(),
            succeeded: Vec::new(),
            failures: Vec::new(),
            skipped: true,
            skip_reason: reason.to_string(),
        }
    }

    /// Whether the step's condition for the next one is satisfied.
    pub fn all_succeeded(&self) -> bool {
        !self.skipped && self.failures.is_empty() && !self.succeeded.is_empty()
    }

    pub fn any_succeeded(&self) -> bool {
        !self.skipped && !self.succeeded.is_empty()
    }
}

/// The chain as a whole: every step in order, including the ones that did not
/// run. A report that only lists the steps that ran hides the fact that the
/// chain stopped, which is the single most important thing to know about a
/// chain that did not finish.
#[derive(Debug, Clone, PartialEq)]
pub struct ChainReport {
    pub steps: Vec<StepOutcome>,
}

impl ChainReport {
    /// Devices the last step that actually ran succeeded on. This is what the
    /// caller asked for when it said "do X, then Y wherever X worked".
    pub fn final_succeeded(&self) -> &[String] {
        self.steps
            .iter()
            .rev()
            .find(|step| !step.skipped)
            .map(|step| step.succeeded.as_slice())
            .unwrap_or(&[])
    }

    /// True when every step ran and none was skipped for a failed condition.
    pub fn completed(&self) -> bool {
        !self.steps.is_empty() && self.steps.iter().all(|step| !step.skipped)
    }

    pub fn to_json(&self) -> Value {
        json!({
            "completed": self.completed(),
            "steps": self.steps.iter().map(|step| json!({
                "tool": step.tool,
                "devices": step.device_ids,
                "succeeded": step.succeeded,
                "failed": step.failures.iter().map(|(id, why)| json!({
                    "device_id": id, "reason": why,
                })).collect::<Vec<_>>(),
                "skipped": step.skipped,
                "skip_reason": step.skip_reason,
            })).collect::<Vec<_>>(),
            "final_succeeded": self.final_succeeded(),
        })
    }
}

/// Bounds a chain so a mistaken call cannot hold a worker forever.
pub const MAX_CHAIN_STEPS: usize = 20;

/// Normalises and validates a chain before any of it runs.
///
/// Every step is checked up front: a chain that discovers a malformed third step
/// only after running the first two has already acted on the fleet, and rolling
/// back is not something this system can do.
pub fn normalize_chain(steps: Vec<ChainStep>) -> Result<Vec<ChainStep>, String> {
    if steps.is_empty() {
        return Err("a chain needs at least one step".to_string());
    }
    if steps.len() > MAX_CHAIN_STEPS {
        return Err(format!(
            "a chain runs at most {MAX_CHAIN_STEPS} steps, got {}",
            steps.len()
        ));
    }
    let mut out = Vec::with_capacity(steps.len());
    for (index, mut step) in steps.into_iter().enumerate() {
        step.tool = step.tool.trim().to_string();
        if step.tool.is_empty() {
            return Err(format!("step {} has no tool name", index + 1));
        }
        let mut seen = std::collections::HashSet::new();
        let mut devices = Vec::new();
        for id in step.device_ids {
            let id = id.trim();
            if id.is_empty() || !seen.insert(id.to_string()) {
                continue;
            }
            devices.push(id.to_string());
        }
        // The first step cannot be conditioned on a previous outcome, because
        // there is not one. Silently treating it as Always would make a
        // mis-ordered chain look like it ran.
        if index == 0 && step.when != StepCondition::Always {
            return Err("the first step of a chain cannot be conditioned on a previous step".to_string());
        }
        step.device_ids = devices;
        out.push(step);
    }
    Ok(out)
}

/// Decides the devices a step runs on, given the previous step's outcome.
///
/// An empty device list means "wherever the previous step succeeded", which is
/// what makes a chain express "do X, then Y where X worked" without restating
/// the fleet in both steps.
pub fn chain_step_devices(
    step: &ChainStep,
    previous: Option<&StepOutcome>,
) -> Result<Vec<String>, String> {
    if !step.device_ids.is_empty() {
        return Ok(step.device_ids.clone());
    }
    match previous {
        None => Err("the first step of a chain must name its devices".to_string()),
        Some(prev) => {
            if prev.skipped {
                return Err(format!(
                    "the previous step did not run ({}), so there is no device to follow",
                    prev.skip_reason
                ));
            }
            if prev.succeeded.is_empty() {
                return Err("the previous step succeeded nowhere, so there is no device to follow"
                    .to_string());
            }
            Ok(prev.succeeded.clone())
        }
    }
}

/// Whether a step runs at all, given the previous outcome and its condition.
pub fn chain_step_should_run(when: StepCondition, previous: Option<&StepOutcome>) -> bool {
    match when {
        StepCondition::Always => true,
        StepCondition::OnAllSucceeded => previous.map(|p| p.all_succeeded()).unwrap_or(false),
        StepCondition::OnAnySucceeded => previous.map(|p| p.any_succeeded()).unwrap_or(false),
        // The cleanup case: the previous step failing is exactly when this runs.
        StepCondition::OnFailure => previous.map(|p| !p.all_succeeded()).unwrap_or(false),
    }
}

/// Why a step was skipped, phrased for the operator rather than for the code.
pub fn chain_skip_reason(when: StepCondition, previous: Option<&StepOutcome>) -> String {
    if previous.is_none() {
        return String::new();
    }
    match when {
        StepCondition::Always => String::new(),
        StepCondition::OnAllSucceeded => {
            let failed = previous.map(|p| p.failures.len()).unwrap_or(0);
            format!("{failed} device(s) in the previous step did not succeed")
        }
        StepCondition::OnAnySucceeded => "the previous step succeeded on no device".to_string(),
        StepCondition::OnFailure => "the previous step succeeded everywhere".to_string(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::batch::{run_batch, BatchDeviceOutcome};

    fn step(tool: &str, devices: &[&str], when: StepCondition) -> ChainStep {
        ChainStep {
            device_ids: devices.iter().map(|s| s.to_string()).collect(),
            tool: tool.to_string(),
            args: json!({}),
            when,
        }
    }

    fn outcome(tool: &str, succeeded: &[&str], failed: &[(&str, &str)]) -> StepOutcome {
        StepOutcome {
            tool: tool.to_string(),
            device_ids: succeeded
                .iter()
                .chain(failed.iter().map(|(id, _)| id))
                .map(|s| s.to_string())
                .collect(),
            succeeded: succeeded.iter().map(|s| s.to_string()).collect(),
            failures: failed
                .iter()
                .map(|(id, why)| (id.to_string(), why.to_string()))
                .collect(),
            skipped: false,
            skip_reason: String::new(),
        }
    }

    fn skipped(tool: &str, reason: &str) -> StepOutcome {
        StepOutcome {
            tool: tool.to_string(),
            device_ids: Vec::new(),
            succeeded: Vec::new(),
            failures: Vec::new(),
            skipped: true,
            skip_reason: reason.to_string(),
        }
    }

    #[test]
    fn a_chain_needs_at_least_one_step() {
        assert!(normalize_chain(vec![]).is_err());
    }

    // A chain that discovers a malformed third step after running the first two
    // has already acted on the fleet, and there is no rolling back. So every
    // step is checked before any of them runs.
    #[test]
    fn a_malformed_step_is_refused_before_anything_runs() {
        let err = normalize_chain(vec![
            step("patch", &["srv-1"], StepCondition::Always),
            step("   ", &["ws-1"], StepCondition::OnAllSucceeded),
        ])
        .unwrap_err();
        assert!(err.contains("step 2"), "error was: {err}");
    }

    #[test]
    fn a_chain_is_bounded() {
        let many: Vec<ChainStep> = (0..(MAX_CHAIN_STEPS + 1))
            .map(|i| step(&format!("tool-{i}"), &["srv-1"], StepCondition::Always))
            .collect();
        assert!(normalize_chain(many).is_err());
    }

    // The first step has nothing to condition on. Accepting a condition there
    // would make a mis-ordered chain look like it ran.
    #[test]
    fn the_first_step_cannot_be_conditioned() {
        let err = normalize_chain(vec![step("patch", &["srv-1"], StepCondition::OnAllSucceeded)])
            .unwrap_err();
        assert!(err.contains("first step"), "error was: {err}");
    }

    #[test]
    fn device_lists_are_trimmed_and_deduplicated() {
        let steps = normalize_chain(vec![step(
            "patch",
            &["srv-1", " srv-1 ", "srv-2", "  "],
            StepCondition::Always,
        )])
        .unwrap();
        assert_eq!(steps[0].device_ids, vec!["srv-1", "srv-2"]);
    }

    // An empty device list means "wherever the previous step succeeded" - the
    // shape that lets a chain say "do X, then Y where X worked".
    #[test]
    fn an_empty_device_list_follows_the_previous_successes() {
        let next = step("restart", &[], StepCondition::OnAllSucceeded);
        let prev = outcome("patch", &["srv-1", "srv-2"], &[]);
        assert_eq!(
            chain_step_devices(&next, Some(&prev)).unwrap(),
            vec!["srv-1", "srv-2"]
        );
    }

    #[test]
    fn the_first_step_must_name_its_devices() {
        let first = step("patch", &[], StepCondition::Always);
        assert!(chain_step_devices(&first, None).is_err());
    }

    // Following a step that succeeded nowhere would silently produce an empty
    // run, which reads as "ran on zero machines and nothing went wrong".
    #[test]
    fn following_a_step_that_succeeded_nowhere_is_refused() {
        let next = step("restart", &[], StepCondition::OnAnySucceeded);
        let prev = outcome("patch", &[], &[("srv-1", "host offline")]);
        assert!(chain_step_devices(&next, Some(&prev)).is_err());
    }

    #[test]
    fn a_skipped_step_cannot_be_followed() {
        let next = step("restart", &[], StepCondition::Always);
        let prev = skipped("patch", "the previous step failed");
        let err = chain_step_devices(&next, Some(&prev)).unwrap_err();
        assert!(err.contains("did not run"), "error was: {err}");
    }

    #[test]
    fn on_all_succeeded_requires_every_device() {
        let all = outcome("patch", &["srv-1", "srv-2"], &[]);
        let some = outcome("patch", &["srv-1"], &[("srv-2", "timeout")]);
        assert!(chain_step_should_run(StepCondition::OnAllSucceeded, Some(&all)));
        assert!(!chain_step_should_run(StepCondition::OnAllSucceeded, Some(&some)));
        assert!(!chain_step_should_run(StepCondition::OnAllSucceeded, None));
    }

    #[test]
    fn on_any_succeeded_needs_one_device_and_runs_only_there() {
        let some = outcome("patch", &["srv-1"], &[("srv-2", "timeout")]);
        assert!(chain_step_should_run(StepCondition::OnAnySucceeded, Some(&some)));
        let none = outcome("patch", &[], &[("srv-1", "offline")]);
        assert!(!chain_step_should_run(StepCondition::OnAnySucceeded, Some(&none)));
    }

    // The cleanup case: the previous step failing is exactly when this runs.
    #[test]
    fn on_failure_runs_when_the_previous_step_did_not_fully_succeed() {
        let some = outcome("patch", &["srv-1"], &[("srv-2", "timeout")]);
        assert!(chain_step_should_run(StepCondition::OnFailure, Some(&some)));
        let all = outcome("patch", &["srv-1"], &[]);
        assert!(!chain_step_should_run(StepCondition::OnFailure, Some(&all)));
    }

    #[test]
    fn skip_reasons_name_what_happened() {
        let some = outcome("patch", &["srv-1"], &[("srv-2", "timeout")]);
        let reason = chain_skip_reason(StepCondition::OnAllSucceeded, Some(&some));
        assert!(reason.contains('1'), "reason was: {reason}");
        let none = outcome("patch", &[], &[("srv-1", "offline")]);
        assert!(chain_skip_reason(StepCondition::OnAnySucceeded, Some(&none)).contains("no device"));
    }

    // A chain that stopped has to say so. A report listing only the steps that
    // ran hides the most important fact about a chain that did not finish.
    #[test]
    fn the_report_distinguishes_a_stopped_chain_from_a_finished_one() {
        let finished = ChainReport {
            steps: vec![
                outcome("patch", &["srv-1"], &[]),
                outcome("restart", &["srv-1"], &[]),
            ],
        };
        assert!(finished.completed());
        assert_eq!(finished.final_succeeded(), &["srv-1"]);

        let stopped = ChainReport {
            steps: vec![
                outcome("patch", &["srv-1"], &[("srv-2", "timeout")]),
                skipped("restart", "1 device(s) in the previous step did not succeed"),
            ],
        };
        assert!(!stopped.completed());
        // The final answer is what the last step that ran achieved, not the
        // empty list the skipped step would otherwise contribute.
        assert_eq!(stopped.final_succeeded(), &["srv-1"]);
    }

    #[test]
    fn the_wire_shape_reports_every_step_including_skipped_ones() {
        let report = ChainReport {
            steps: vec![
                outcome("patch", &["srv-1"], &[("srv-2", "timeout")]),
                skipped("restart", "1 device(s) did not succeed"),
            ],
        };
        let json = report.to_json();
        assert_eq!(json["completed"], false);
        let steps = json["steps"].as_array().unwrap();
        assert_eq!(steps.len(), 2, "a skipped step must still be reported");
        assert_eq!(steps[1]["skipped"], true);
        assert_eq!(steps[0]["failed"][0]["device_id"], "srv-2");
    }

    // A step that ran on nothing is not a step that succeeded everywhere: the
    // two are different facts and a chain must not conflate them.
    #[test]
    fn a_step_that_ran_nowhere_is_not_a_success() {
        let empty = outcome("patch", &[], &[]);
        assert!(!empty.all_succeeded());
        assert!(!empty.any_succeeded());
    }

// The executor: the half that actually runs a chain.
//
// The rules above decide what *should* happen. These tests decide whether it
// does - and specifically whether a step is skipped when its condition is not
// met, which is the difference between a chain and a batch with extra steps.

use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::Arc;

/// A stand-in for one host's answer. Records what it was asked so a test can
/// assert the chain did not call a step it should have skipped.
#[derive(Clone, Default)]
struct FakeHost {
    /// (tool, device) pairs this host was asked about, in call order.
    pub calls: Arc<std::sync::Mutex<Vec<(String, String)>>>,
    /// Devices whose tool call fails.
    pub failing: Vec<String>,
}

impl FakeHost {
    fn new(failing: &[&str]) -> Self {
        Self {
            calls: Arc::new(std::sync::Mutex::new(Vec::new())),
            failing: failing.iter().map(|s| s.to_string()).collect(),
        }
    }

    async fn call(&self, tool: &str, device_id: &str) -> Result<String, String> {
        self.calls
            .lock()
            .unwrap()
            .push((tool.to_string(), device_id.to_string()));
        if self.failing.iter().any(|f| f == device_id) {
            return Err(format!("{device_id} refused {tool}"));
        }
        Ok(format!("{tool} ok on {device_id}"))
    }

    fn calls_for(&self, tool: &str) -> Vec<String> {
        self.calls
            .lock()
            .unwrap()
            .iter()
            .filter(|(t, _)| t == tool)
            .map(|(_, d)| d.clone())
            .collect()
    }
}

/// Runs one step against the fake host, reusing the batch's per-device
/// concurrency. This is the seam the real executor uses too: the chain decides
/// which devices and whether to run at all, the batch does the fan-out.
async fn run_step_on(
    host: &FakeHost,
    step: &ChainStep,
    devices: &[String],
) -> StepOutcome {
    let devices = devices.to_vec();
    // run_batch needs a 'static closure, so the two values it reads are taken
    // out of the borrowed arguments before the closure rather than inside it.
    let host = host.clone();
    let tool = step.tool.clone();
    let report = run_batch(devices.clone(), move |device_id| {
        let host = host.clone();
        let tool = tool.clone();
        async move {
            match host.call(&tool, &device_id).await {
                Ok(detail) => BatchDeviceOutcome {
                    device_id,
                    ok: true,
                    detail,
                },
                Err(why) => BatchDeviceOutcome {
                    device_id,
                    ok: false,
                    detail: why,
                },
            }
        }
    })
    .await;

    let mut out = StepOutcome::ran(&step.tool, devices);
    for entry in report.outcomes {
        if entry.ok {
            out.succeeded.push(entry.device_id);
        } else {
            out.failures.push((entry.device_id, entry.detail));
        }
    }
    out
}

/// The chain executor under test.
async fn run_chain_on(host: &FakeHost, steps: Vec<ChainStep>) -> Result<ChainReport, String> {
    let steps = normalize_chain(steps)?;
    let mut outcomes: Vec<StepOutcome> = Vec::with_capacity(steps.len());
    let mut previous: Option<StepOutcome> = None;

    for step in &steps {
        if !chain_step_should_run(step.when, previous.as_ref()) {
            let reason = chain_skip_reason(step.when, previous.as_ref());
            outcomes.push(skipped(&step.tool, &reason));
            // A skipped step is not a previous outcome to follow: the next
            // step's condition is evaluated against the last step that ran.
            continue;
        }
        let devices = chain_step_devices(step, previous.as_ref())?;
        let outcome = run_step_on(host, step, &devices).await;
        previous = Some(outcome.clone());
        outcomes.push(outcome);
    }

    Ok(ChainReport { steps: outcomes })
}

#[tokio::test]
async fn a_two_step_chain_runs_the_second_step_on_the_first_steps_devices() {
    let host = FakeHost::new(&[]);
    let report = run_chain_on(
        &host,
        vec![
            step("patch", &["srv-1", "srv-2"], StepCondition::Always),
            step("restart", &[], StepCondition::OnAllSucceeded),
        ],
    )
    .await
    .unwrap();

    assert_eq!(report.steps.len(), 2);
    assert!(report.completed());
    assert_eq!(report.final_succeeded(), &["srv-1", "srv-2"]);
    // The second step ran on exactly what the first step named.
    assert_eq!(host.calls_for("restart"), vec!["srv-1", "srv-2"]);
}

// The whole point of the feature: when the first step fails somewhere, the
// conditioned step does not run at all - not "runs and fails", not "runs on
// the survivors". Skipping is a decision the chain makes, and it has to be
// visible in the report.
#[tokio::test]
async fn a_conditioned_step_is_skipped_when_the_previous_step_fails() {
    let host = FakeHost::new(&["srv-2"]);
    let report = run_chain_on(
        &host,
        vec![
            step("patch", &["srv-1", "srv-2"], StepCondition::Always),
            step("restart", &[], StepCondition::OnAllSucceeded),
        ],
    )
    .await
    .unwrap();

    assert!(!report.completed(), "a chain with a skipped step did not finish");
    assert!(report.steps[1].skipped);
    assert!(report.steps[1].skip_reason.contains('1'));
    // The decisive assertion: nothing was sent for the second tool.
    assert!(
        host.calls_for("restart").is_empty(),
        "the skipped step still ran: {:?}",
        host.calls_for("restart")
    );
    // And the first step's partial success is still reported, because that is
    // what the operator needs to act on.
    assert_eq!(report.steps[0].succeeded, vec!["srv-1"]);
    assert_eq!(report.steps[0].failures.len(), 1);
}

// OnAnySucceeded is the tiered rollout shape: whatever worked gets the next
// step, whatever failed does not.
#[tokio::test]
async fn on_any_succeeded_runs_the_next_step_only_on_the_survivors() {
    let host = FakeHost::new(&["srv-2"]);
    let report = run_chain_on(
        &host,
        vec![
            step("patch", &["srv-1", "srv-2"], StepCondition::Always),
            step("restart", &[], StepCondition::OnAnySucceeded),
        ],
    )
    .await
    .unwrap();

    assert!(report.completed());
    assert_eq!(report.final_succeeded(), &["srv-1"]);
    assert_eq!(host.calls_for("restart"), vec!["srv-1"]);
}

// The cleanup case: the second step runs *because* the first one failed, which
// is the only condition that must fire on failure.
#[tokio::test]
async fn on_failure_runs_the_next_step_when_the_previous_one_failed() {
    let host = FakeHost::new(&["srv-1"]);
    let report = run_chain_on(
        &host,
        vec![
            step("patch", &["srv-1", "srv-2"], StepCondition::Always),
            step("rollback", &["srv-1", "srv-2"], StepCondition::OnFailure),
        ],
    )
    .await
    .unwrap();

    assert!(report.completed());
    // The rollback is attempted on both machines, including the one that failed
    // the patch - that is the point of a cleanup step. The host still refuses
    // srv-1, so only one succeeds, but the attempt itself is the assertion:
    // a rollback that skipped the failed machine would leave it patched.
    assert_eq!(host.calls_for("rollback"), vec!["srv-1", "srv-2"]);
    assert_eq!(report.steps[1].succeeded, vec!["srv-2"]);
    assert_eq!(report.steps[1].failures.len(), 1);
}

// A chain whose second step is skipped stops there: the third step's condition
// is evaluated against the last step that actually ran, not against a step that
// did not happen. Otherwise a skip would cascade into conditions that look
// satisfied by nothing.
#[tokio::test]
async fn a_skipped_step_does_not_become_the_previous_outcome() {
    let host = FakeHost::new(&["srv-2"]);
    let report = run_chain_on(
        &host,
        vec![
            step("patch", &["srv-1", "srv-2"], StepCondition::Always),
            step("restart", &[], StepCondition::OnAllSucceeded),
            step("verify", &[], StepCondition::OnAnySucceeded),
        ],
    )
    .await
    .unwrap();

    // Step 2 skipped, so step 3's condition is judged against step 1, which
    // did succeed somewhere - so step 3 runs on the survivors.
    assert!(report.steps[1].skipped);
    assert!(!report.steps[2].skipped, "step 3 was skipped because step 2 was");
    assert_eq!(host.calls_for("verify"), vec!["srv-1"]);
}

// An explicit device list on a later step overrides the "follow the survivors"
// default, because sometimes the next step really does go somewhere else.
#[tokio::test]
async fn an_explicit_device_list_overrides_the_follow_default() {
    let host = FakeHost::new(&[]);
    let report = run_chain_on(
        &host,
        vec![
            step("patch", &["srv-1"], StepCondition::Always),
            step("restart", &["ws-1", "ws-2"], StepCondition::OnAllSucceeded),
        ],
    )
    .await
    .unwrap();

    assert!(report.completed());
    assert_eq!(host.calls_for("restart"), vec!["ws-1", "ws-2"]);
    assert_eq!(report.final_succeeded(), &["ws-1", "ws-2"]);
}

// Nothing runs until the whole chain validates. A chain that acts on the fleet
// and then discovers step three was malformed has already acted.
#[tokio::test]
async fn a_malformed_chain_runs_nothing() {
    let host = FakeHost::new(&[]);
    let err = run_chain_on(
        &host,
        vec![
            step("patch", &["srv-1"], StepCondition::Always),
            step("", &["srv-2"], StepCondition::OnAllSucceeded),
        ],
    )
    .await
    .unwrap_err();

    assert!(err.contains("step 2"), "error was: {err}");
    assert!(
        host.calls.lock().unwrap().is_empty(),
        "a malformed chain still ran: {:?}",
        host.calls.lock().unwrap()
    );
}

// Devices within a step run concurrently, the same as a batch: a chain is not a
// reason to make each step slower than the batch it replaced.
#[tokio::test]
async fn devices_within_a_step_overlap() {
    // The barrier counts the arrivals it expects: two devices means two. A
    // barrier waiting for a participant that never arrives deadlocks the test
    // rather than failing it, which is how this test hung the suite the first
    // time it was written.
    let barrier = Arc::new(tokio::sync::Barrier::new(2));
    let arrived = Arc::new(AtomicUsize::new(0));

    let devices = vec!["srv-1".to_string(), "srv-2".to_string()];
    let report = run_batch(devices.clone(), {
        let barrier = barrier.clone();
        let arrived = arrived.clone();
        move |device_id| {
            let barrier = barrier.clone();
            let arrived = arrived.clone();
            async move {
                arrived.fetch_add(1, Ordering::SeqCst);
                // A sequential implementation deadlocks here rather than merely
                // running slowly.
                barrier.wait().await;
                BatchDeviceOutcome {
                    device_id,
                    ok: true,
                    detail: "done".to_string(),
                }
            }
        }
    })
    .await;

    assert_eq!(report.succeeded(), 2);
    assert_eq!(arrived.load(Ordering::SeqCst), 2);
}
}
