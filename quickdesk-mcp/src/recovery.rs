use std::collections::HashMap;

use serde::{Deserialize, Serialize};

/// Where a parked run can be resumed from.
///
/// A parked step is the point where the replay stopped touching the remote
/// machine: the screen did not match what the recording expected, so the run
/// refuses to keep clicking pixels that may now belong to something else.
/// That is the right call for an unattended run and the wrong one for a
/// supervised one - an operator who has just fixed the screen wants to carry
/// on from where the replay stopped, not start over.
///
/// `ResumePoint` is that handover: the step the run parked on, the overrides
/// it was running with, and enough context for the operator to see what the
/// replay expected versus what it saw.
#[derive(Clone, Debug, Serialize, Deserialize, PartialEq)]
pub struct ResumePoint {
    pub run_id: String,
    pub workflow_id: String,
    pub device_id: String,
    /// 1-based step the run parked on. Resuming starts here.
    pub parked_step: u32,
    /// Total steps the parked run had, so "is there anything left" is answered
    /// from the run itself rather than by re-reading the workflow - which may
    /// have been edited or deleted since.
    #[serde(default)]
    pub total_steps: u32,
    /// Why the run parked, taken from the parked step's error text.
    pub reason: String,
    /// The overrides the parked run was executing with, so a resume does not
    /// silently drop them.
    #[serde(default)]
    pub overrides: HashMap<String, serde_json::Value>,
    #[serde(default)]
    pub stop_on_failure: bool,
}

impl ResumePoint {
    /// Builds a resume point from a parked run's report.
    ///
    /// Returns `None` when the run did not park, because there is nothing to
    /// hand over: a run that failed outright, was cancelled, or completed
    /// has no resumable position.
    pub fn from_report(
        run_id: &str,
        workflow_id: &str,
        device_id: &str,
        overrides: HashMap<String, serde_json::Value>,
        stop_on_failure: bool,
        total_steps: u32,
        steps: &[super::workflow::StepRunResult],
    ) -> Option<Self> {
        // Only a pending step is a park. A step that failed outright is a
        // real failure: the replay ran the tool and it did not work, so
        // resuming from there would repeat the same failure rather than give
        // the operator a fresh position to work from.
        let parked = steps
            .iter()
            .find(|step| step.pending)
            ?;
        Some(Self {
            run_id: run_id.to_string(),
            workflow_id: workflow_id.to_string(),
            device_id: device_id.to_string(),
            parked_step: parked.seq,
            total_steps,
            reason: parked.error.clone(),
            overrides,
            stop_on_failure,
        })
    }

    /// Reports whether resuming from this point would run any step at all.
    ///
    /// A point parked on the final step has nothing left: the operator would
    /// restart the workflow rather than resume it, and saying so is more
    /// useful than accepting a resume that immediately completes.
    pub fn has_remaining_steps(&self, total_steps: u32) -> bool {
        if self.total_steps > 0 {
            return self.parked_step < self.total_steps;
        }
        self.parked_step < total_steps.max(1)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::workflow::{StepExecutor, StepRunResult};

    fn step(seq: u32, success: bool, pending: bool) -> StepRunResult {
        StepRunResult {
            seq,
            tool_name: format!("tool-{seq}"),
            executor: StepExecutor::SkillHost,
            success,
            pending,
            result: serde_json::Value::Null,
            error: if pending {
                "step 3 screen state did not match".to_string()
            } else {
                String::new()
            },
            duration_ms: 0,
        }
    }

    // The core guarantee: a run parked on a screen mismatch hands over the
    // step it stopped on, so an operator can continue from there.
    #[test]
    fn a_parked_run_yields_a_resume_point() {
        let steps = vec![step(1, true, false), step(2, true, false), step(3, false, true)];

        let point = ResumePoint::from_report(
            "run-1",
            "wf-1",
            "device-1",
            HashMap::new(),
            true,
            steps.len() as u32,
            &steps,
        )
        .expect("a parked run must yield a resume point");

        assert_eq!(point.parked_step, 3);
        assert_eq!(point.workflow_id, "wf-1");
        assert!(point.reason.contains("did not match"));
    }

    // A run that failed outright has no resumable position: the last step is a
    // real failure, not a park.
    #[test]
    fn a_failed_run_has_no_resume_point() {
        let steps = vec![step(1, true, false), step(2, false, false)];

        assert!(ResumePoint::from_report(
            "run-2",
            "wf-1",
            "device-1",
            HashMap::new(),
            true,
            steps.len() as u32,
            &steps,
        )
        .is_none());
    }

    // The overrides travel with the point: dropping them would make a resume
    // run different steps than the run that parked.
    #[test]
    fn overrides_travel_with_the_resume_point() {
        let steps = vec![step(1, false, true)];
        let mut overrides = HashMap::new();
        overrides.insert("target".to_string(), serde_json::json!("window-2"));

        let point = ResumePoint::from_report(
            "run-3",
            "wf-1",
            "device-1",
            overrides.clone(),
            false,
            steps.len() as u32,
            &steps,
        )
        .expect("a parked run must yield a resume point");

        assert_eq!(point.overrides, overrides);
        assert!(!point.stop_on_failure);
    }

    // A point parked on the last step has nothing to resume.
    #[test]
    fn parking_on_the_last_step_leaves_nothing_to_resume() {
        let steps = vec![step(1, true, false), step(2, false, true)];
        let point = ResumePoint::from_report(
            "run-4",
            "wf-1",
            "device-1",
            HashMap::new(),
            true,
            steps.len() as u32,
            &steps,
        )
        .expect("a parked run must yield a resume point");

        // The run's own total decides: parked on step 2 of 2 leaves nothing,
        // and the caller's argument is not consulted once the run recorded one.
        assert!(!point.has_remaining_steps(2));
        assert!(!point.has_remaining_steps(5));
    }

    // A point parked mid-run has something left, which is the case an
    // operator actually resumes from.
    #[test]
    fn parking_mid_run_leaves_steps_to_resume() {
        let steps = vec![step(1, true, false), step(2, false, true)];
        let point = ResumePoint::from_report(
            "run-6",
            "wf-1",
            "device-1",
            HashMap::new(),
            true,
            6,
            &steps,
        )
        .expect("a parked run must yield a resume point");

        assert!(point.has_remaining_steps(0));
    }

    // An empty step list cannot produce a point; the caller must not treat
    // that as "resume from the beginning".
    #[test]
    fn an_empty_step_list_has_no_resume_point() {
        assert!(ResumePoint::from_report(
            "run-5",
            "wf-1",
            "device-1",
            HashMap::new(),
            true,
            0,
            &[],
        )
        .is_none());
    }
}
