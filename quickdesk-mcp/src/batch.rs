// Batch skill execution across many devices, with one acknowledgement per device.
//
// `skill_exec` runs one tool on one host. The batch prompt that already exists
// only tells a model to loop over devices itself, which means the model decides
// what "done" means for each one and the operator gets prose instead of a count.
// This is the execution half: issue the same tool to a set of devices
// concurrently, wait for each acknowledgement, and report per-device outcomes.

use serde_json::{json, Value};

/// One device's outcome in a batch run.
#[derive(Debug, Clone, PartialEq)]
pub struct BatchDeviceOutcome {
    pub device_id: String,
    pub ok: bool,
    /// The host's answer, or why there is none.
    pub detail: String,
}

/// The batch as a whole, in a shape an operator can act on.
#[derive(Debug, Clone, PartialEq)]
pub struct BatchReport {
    pub outcomes: Vec<BatchDeviceOutcome>,
}

impl BatchReport {
    pub fn succeeded(&self) -> usize {
        self.outcomes.iter().filter(|o| o.ok).count()
    }

    pub fn failed(&self) -> usize {
        self.outcomes.len() - self.succeeded()
    }

    /// The wire shape. Every device is listed, whichever way it went: a summary
    /// that only names the failures hides how much of the fleet was never
    /// reached at all.
    pub fn to_json(&self) -> Value {
        json!({
            "total": self.outcomes.len(),
            "succeeded": self.succeeded(),
            "failed": self.failed(),
            "devices": self.outcomes.iter().map(|o| json!({
                "device_id": o.device_id,
                "ok": o.ok,
                "detail": o.detail,
            })).collect::<Vec<_>>(),
        })
    }
}

/// Normalises what a caller sent into the device list the batch will run on.
///
/// Rejects an empty list rather than returning an empty success: "ran on zero
/// devices" reported as `succeeded: 0, failed: 0` reads like a fleet where
/// nothing went wrong, when the truth is nothing was attempted.
pub fn normalize_batch_devices(raw: &[String]) -> Result<Vec<String>, String> {
    let mut seen = std::collections::HashSet::new();
    let mut out = Vec::new();
    for entry in raw {
        let id = entry.trim();
        if id.is_empty() {
            continue;
        }
        // A duplicate would run the same command twice on one machine and count
        // it as two acknowledgements.
        if seen.insert(id.to_string()) {
            out.push(id.to_string());
        }
    }
    if out.is_empty() {
        return Err("a batch needs at least one device id".to_string());
    }
    Ok(out)
}

/// Bounds one batch so a mistaken call cannot hold a worker for an unbounded
/// time or open that many sessions at once.
pub const MAX_BATCH_DEVICES: usize = 50;

pub fn validate_batch_size(count: usize) -> Result<(), String> {
    if count == 0 {
        return Err("a batch needs at least one device id".to_string());
    }
    if count > MAX_BATCH_DEVICES {
        return Err(format!(
            "a batch covers at most {MAX_BATCH_DEVICES} devices, got {count}"
        ));
    }
    Ok(())
}

/// Runs one tool across a set of devices concurrently and collects one
/// acknowledgement per device.
///
/// Concurrency is the point: a batch over fifty hosts run one at a time is
/// fifty round trips the operator waits through, and the whole reason to batch
/// is that the same command is going to many machines at once. The WS client
/// already multiplexes by request id, so the requests genuinely overlap rather
/// than queueing behind each other.
///
/// A device that fails does not fail the batch. One unreachable host must not
/// hide the result of the other forty-nine, which is what a sequential
/// fail-fast would do.
pub async fn run_batch<F, Fut>(
    devices: Vec<String>,
    per_device: F,
) -> BatchReport
where
    // 'static + Send so the futures can outlive this call on their own tasks,
    // and F is shared by every task rather than moved into the first one.
    F: Fn(String) -> Fut + Send + Sync + 'static,
    Fut: std::future::Future<Output = BatchDeviceOutcome> + Send + 'static,
{
    // Arc so every spawned task can call the same closure rather than the first
    // one consuming it.
    let per_device = std::sync::Arc::new(per_device);
    let mut handles = Vec::with_capacity(devices.len());
    for device_id in devices {
        let per_device = std::sync::Arc::clone(&per_device);
        handles.push(tokio::spawn(async move {
            // A panic in one device's work must not take the batch down with it:
            // the other acknowledgements are still worth having. Awaiting the
            // future inside the spawned task is what isolates it.
            per_device(device_id).await
        }));
    }

    let mut outcomes = Vec::with_capacity(handles.len());
    for handle in handles {
        match handle.await {
            Ok(outcome) => outcomes.push(outcome),
            Err(_) => {
                // The join itself failed, which means the task was cancelled or
                // the runtime is shutting down. Recording it as a failure keeps
                // the count honest rather than silently dropping a device.
                outcomes.push(BatchDeviceOutcome {
                    device_id: String::new(),
                    ok: false,
                    detail: "the batch task did not complete".to_string(),
                });
            }
        }
    }
    BatchReport { outcomes }
}


#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn a_batch_needs_at_least_one_device() {
        // "Ran on zero devices" reported as 0 succeeded / 0 failed reads like a
        // fleet where nothing went wrong. The truth is nothing was attempted,
        // and those are different facts.
        assert!(normalize_batch_devices(&[]).is_err());
        assert!(normalize_batch_devices(&["   ".to_string()]).is_err());
        assert!(validate_batch_size(0).is_err());
    }

    #[test]
    fn duplicates_collapse_so_one_machine_is_not_two_acknowledgements() {
        let devices = normalize_batch_devices(&[
            "dev-1".to_string(),
            " dev-1 ".to_string(),
            "dev-2".to_string(),
        ])
        .expect("three entries with one duplicate");
        assert_eq!(devices, vec!["dev-1".to_string(), "dev-2".to_string()]);
    }

    #[test]
    fn the_size_bound_refuses_a_fleet_the_caller_did_not_mean() {
        assert!(validate_batch_size(MAX_BATCH_DEVICES).is_ok());
        assert!(validate_batch_size(MAX_BATCH_DEVICES + 1).is_err());
    }

    #[test]
    fn the_report_counts_both_ways_and_lists_every_device() {
        let report = BatchReport {
            outcomes: vec![
                BatchDeviceOutcome {
                    device_id: "dev-1".to_string(),
                    ok: true,
                    detail: "exit 0".to_string(),
                },
                BatchDeviceOutcome {
                    device_id: "dev-2".to_string(),
                    ok: false,
                    detail: "host offline".to_string(),
                },
                BatchDeviceOutcome {
                    device_id: "dev-3".to_string(),
                    ok: false,
                    detail: "tool not found".to_string(),
                },
            ],
        };
        assert_eq!(report.succeeded(), 1);
        assert_eq!(report.failed(), 2);

        let wire = report.to_json();
        assert_eq!(wire["total"], json!(3));
        assert_eq!(wire["succeeded"], json!(1));
        assert_eq!(wire["failed"], json!(2));
        // A summary that only names the failures hides how much of the fleet
        // was never reached, so every device is listed whichever way it went.
        assert_eq!(wire["devices"].as_array().map(|a| a.len()), Some(3));
        assert_eq!(wire["devices"][1]["device_id"], json!("dev-2"));
        assert_eq!(wire["devices"][1]["ok"], json!(false));
    }

    // Concurrency has to be observable, or the batch could quietly serialise and
    // nobody would notice until a fifty-host run took fifty round trips. Each
    // device waits on a barrier that only opens once every device has arrived,
    // so a serial implementation would deadlock rather than merely be slow.
    #[tokio::test]
    async fn devices_run_concurrently_not_one_at_a_time() {
        let count = 6;
        let barrier = std::sync::Arc::new(tokio::sync::Barrier::new(count));
        let devices: Vec<String> = (0..count).map(|i| format!("dev-{i}")).collect();

        let report = run_batch(devices, {
            let barrier = std::sync::Arc::clone(&barrier);
            move |device_id| {
                let barrier = std::sync::Arc::clone(&barrier);
                async move {
                    // Every device must arrive before any of them may proceed.
                    barrier.wait().await;
                    BatchDeviceOutcome {
                        device_id,
                        ok: true,
                        detail: "ok".to_string(),
                    }
                }
            }
        })
        .await;

        assert_eq!(report.succeeded(), count);
        assert_eq!(report.failed(), 0);
    }

    // One unreachable host must not hide the other results: the operator still
    // needs to know what happened on the machines that did answer.
    #[tokio::test]
    async fn a_failing_device_does_not_fail_the_batch() {
        let devices = vec![
            "dev-ok".to_string(),
            "dev-offline".to_string(),
            "dev-ok-2".to_string(),
        ];
        let report = run_batch(devices, |device_id| async move {
            if device_id == "dev-offline" {
                return BatchDeviceOutcome {
                    device_id,
                    ok: false,
                    detail: "host offline".to_string(),
                };
            }
            BatchDeviceOutcome {
                device_id,
                ok: true,
                detail: "ok".to_string(),
            }
        })
        .await;

        assert_eq!(report.succeeded(), 2);
        assert_eq!(report.failed(), 1);
        assert_eq!(report.outcomes.len(), 3, "every device is accounted for");
    }

    // Order must follow the caller's list rather than completion order, so a
    // report can be diffed against the request that produced it.
    #[tokio::test]
    async fn the_report_keeps_the_requested_device_order() {
        let devices = vec![
            "dev-a".to_string(),
            "dev-b".to_string(),
            "dev-c".to_string(),
        ];
        let report = run_batch(devices.clone(), |device_id| async move {
            // Finish in the opposite order to prove the report is not assembled
            // from whichever acknowledgement happened to land first.
            let delay = match device_id.as_str() {
                "dev-a" => 60,
                "dev-b" => 30,
                _ => 0,
            };
            tokio::time::sleep(std::time::Duration::from_millis(delay)).await;
            BatchDeviceOutcome {
                device_id,
                ok: true,
                detail: "ok".to_string(),
            }
        })
        .await;

        let ids: Vec<&str> = report.outcomes.iter().map(|o| o.device_id.as_str()).collect();
        assert_eq!(ids, vec!["dev-a", "dev-b", "dev-c"]);
    }

    #[test]
    fn an_all_success_batch_reports_no_failures() {
        let report = BatchReport {
            outcomes: vec![BatchDeviceOutcome {
                device_id: "dev-1".to_string(),
                ok: true,
                detail: "ok".to_string(),
            }],
        };
        let wire = report.to_json();
        assert_eq!(wire["failed"], json!(0));
        assert_eq!(wire["succeeded"], json!(1));
    }
}
