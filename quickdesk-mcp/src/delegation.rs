use std::collections::HashMap;
use std::path::PathBuf;

use serde::{Deserialize, Serialize};
use serde_json::Value;
use uuid::Uuid;

/// One Owner-issued delegation grant.
///
/// The HTTP trigger is authenticated by a single shared bearer token, which
/// means every holder can replay every workflow against every device the
/// process allows. That is fine for one business system and wrong the moment
/// a second one - or a contractor - needs access: the owner has no way to say
/// "this caller may only run these workflows on these machines".
///
/// A grant is the owner's answer to that. It names a caller, bounds it to a
/// set of workflows and devices, optionally caps what a single run may
/// execute, and expires. The `subject` is what `submitter_key` already
/// computes for the fair queue, so the same operator identity that decides
/// queue order also decides authority.
#[derive(Clone, Debug, Serialize, Deserialize, PartialEq, Eq)]
pub struct DelegationGrant {
    pub id: String,
    /// Caller identity this grant applies to, e.g. `operator:alice`.
    pub subject: String,
    /// Human label the owner uses to remember what this was for.
    #[serde(default)]
    pub label: String,
    /// Workflow IDs the subject may run. Empty means "all workflows".
    #[serde(default)]
    pub workflow_ids: Vec<String>,
    /// Devices the subject may run against. Empty means "all allowed
    /// devices" - it never widens the process-level allow list, only narrows
    /// it.
    #[serde(default)]
    pub device_ids: Vec<String>,
    /// Maximum steps one run may execute. 0 means "no extra cap".
    #[serde(default)]
    pub max_steps: u32,
    /// Tool names the subject may not use in any run. The trust layer's own
    /// blocked list still applies on top of this.
    #[serde(default)]
    pub denied_tools: Vec<String>,
    pub created_at: String,
    /// Grants expire; a grant past this instant no longer authorises.
    pub expires_at: String,
    /// Revocation timestamp. A revoked grant is refused even before it
    /// expires, so the owner can react without waiting for the TTL.
    #[serde(default)]
    pub revoked_at: Option<String>,
}

impl DelegationGrant {
    fn new(
        subject: &str,
        label: &str,
        workflow_ids: Vec<String>,
        device_ids: Vec<String>,
        max_steps: u32,
        denied_tools: Vec<String>,
        expires_at: &str,
    ) -> Self {
        Self {
            id: Uuid::new_v4().to_string(),
            subject: subject.trim().to_string(),
            label: label.trim().to_string(),
            workflow_ids: normalize_list(workflow_ids),
            device_ids: normalize_list(device_ids),
            max_steps,
            denied_tools: normalize_list(denied_tools),
            created_at: chrono::Utc::now().to_rfc3339(),
            expires_at: expires_at.trim().to_string(),
            revoked_at: None,
        }
    }

    /// Reports whether this grant still authorises its subject.
    ///
    /// An unparsable expiry is treated as expired: a grant whose TTL cannot be
    /// read must not silently become permanent.
    fn is_live_at(&self, now: chrono::DateTime<chrono::Utc>) -> bool {
        if self.revoked_at.is_some() {
            return false;
        }
        match chrono::DateTime::parse_from_rfc3339(&self.expires_at) {
            Ok(expiry) => expiry.with_timezone(&chrono::Utc) > now,
            Err(_) => false,
        }
    }

    fn permits_workflow(&self, workflow_id: &str) -> bool {
        self.workflow_ids.is_empty()
            || self.workflow_ids.iter().any(|id| id == workflow_id)
    }

    fn permits_device(&self, device_id: &str) -> bool {
        self.device_ids.is_empty() || self.device_ids.iter().any(|id| id == device_id)
    }
}

fn normalize_list(values: Vec<String>) -> Vec<String> {
    let mut seen = Vec::new();
    for value in values {
        let value = value.trim();
        if !value.is_empty() && !seen.iter().any(|existing| existing == value) {
            seen.push(value.to_string());
        }
    }
    seen
}

/// The outcome of checking one request against the owner's grants.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum DelegationDecision {
    /// A live grant authorises this exact request.
    Allowed { grant_id: String, max_steps: u32, denied_tools: Vec<String> },
    /// The subject has grants, but none covers this request.
    Denied { reason: String },
}

/// Store of delegation grants, persisted as one JSON file so a restart does
/// not silently widen what a caller may do.
#[derive(Clone, Debug)]
pub struct DelegationStore {
    path: PathBuf,
}

impl DelegationStore {
    pub fn new() -> Result<Self, String> {
        let base = delegation_data_dir().unwrap_or_else(|| PathBuf::from("."));
        Self::from_path(base.join(".quickdesk").join("delegation-grants.json"))
    }

    /// Builds a store over an explicit path. Used by tests and by an operator
    /// who wants the grant file somewhere other than the default location.
    pub(crate) fn from_path(path: PathBuf) -> Result<Self, String> {
        if let Some(parent) = path.parent() {
            std::fs::create_dir_all(parent)
                .map_err(|error| format!("cannot create delegation dir: {error}"))?;
        }
        Ok(Self { path })
    }

    fn load(&self) -> Result<Vec<DelegationGrant>, String> {
        if !self.path.exists() {
            return Ok(Vec::new());
        }
        let data = std::fs::read_to_string(&self.path)
            .map_err(|error| format!("read delegation grants: {error}"))?;
        serde_json::from_str(&data).map_err(|error| format!("parse delegation grants: {error}"))
    }

    fn save(&self, grants: &[DelegationGrant]) -> Result<(), String> {
        let json = serde_json::to_vec_pretty(grants)
            .map_err(|error| format!("encode delegation grants: {error}"))?;
        std::fs::write(&self.path, json)
            .map_err(|error| format!("write delegation grants: {error}"))
    }

    pub fn list(&self) -> Result<Vec<DelegationGrant>, String> {
        self.load()
    }

    /// Issues a grant for `subject`. The subject must already be an identity
    /// the owner recognises - typically an operator token - because a grant
    /// for an unknown caller would authorise nobody.
/// Issues a grant for `subject`. The subject must already be an identity
/// the owner recognises - typically an operator token - because a grant
/// for an unknown caller would authorise nobody.
// The parameter count is the grant's own fields: subject, label, workflow
// scope, device scope, step cap and denied tools plus the expiry. Grouping
// them into a struct would only move the same six values one layer out.
#[allow(clippy::too_many_arguments)]
    pub fn grant(
        &self,
        subject: &str,
        label: &str,
        workflow_ids: Vec<String>,
        device_ids: Vec<String>,
        max_steps: u32,
        denied_tools: Vec<String>,
        expires_at: &str,
    ) -> Result<DelegationGrant, String> {
        let subject = subject.trim();
        if subject.is_empty() {
            return Err("subject is required".to_string());
        }
        let grant = DelegationGrant::new(
            subject,
            label,
            workflow_ids,
            device_ids,
            max_steps,
            denied_tools,
            expires_at,
        );
        let mut grants = self.load()?;
        grants.push(grant.clone());
        self.save(&grants)?;
        Ok(grant)
    }

    /// Revokes by id. Returns whether a grant was actually changed, so a
    /// caller cannot tell "revoked" from "already revoked" apart and assume
    /// the second one failed.
    pub fn revoke(&self, grant_id: &str) -> Result<bool, String> {
        let mut grants = self.load()?;
        let now = chrono::Utc::now().to_rfc3339();
        let mut changed = false;
        for grant in grants.iter_mut() {
            if grant.id == grant_id && grant.revoked_at.is_none() {
                grant.revoked_at = Some(now.clone());
                changed = true;
            }
        }
        if changed {
            self.save(&grants)?;
        }
        Ok(changed)
    }

    /// Decides whether `subject` may run `workflow_id` against `device_id`.
    ///
    /// Every live grant that could apply is examined so the caller can be
    /// told which one authorised the run - that value goes into the audit
    /// trail, and an owner reviewing it needs to know which grant to revoke.
    pub fn authorize(
        &self,
        subject: &str,
        workflow_id: &str,
        device_id: &str,
    ) -> DelegationDecision {
        let now = chrono::Utc::now();
        let Ok(grants) = self.load() else {
            return DelegationDecision::Denied {
                reason: "delegation grants cannot be read".to_string(),
            };
        };
        let mut denied_reason: Option<String> = None;
        for grant in grants.iter().filter(|grant| grant.subject == subject) {
            if !grant.is_live_at(now) {
                denied_reason = Some("the delegation grant for this caller is expired or revoked".to_string());
                continue;
            }
            if !grant.permits_workflow(workflow_id) {
                denied_reason = Some(format!(
                    "delegation grant does not cover workflow '{workflow_id}'"
                ));
                continue;
            }
            if !grant.permits_device(device_id) {
                denied_reason = Some(format!(
                    "delegation grant does not cover device '{device_id}'"
                ));
                continue;
            }
            return DelegationDecision::Allowed {
                grant_id: grant.id.clone(),
                max_steps: grant.max_steps,
                denied_tools: grant.denied_tools.clone(),
            };
        }
        denied_reason.map_or_else(
            || {
                DelegationDecision::Denied {
                    reason: format!("no delegation grant covers caller '{subject}'"),
                }
            },
            |reason| DelegationDecision::Denied { reason },
        )
    }
}

fn delegation_data_dir() -> Option<PathBuf> {
    #[cfg(target_os = "windows")]
    {
        std::env::var("USERPROFILE").ok().map(PathBuf::from)
    }
    #[cfg(not(target_os = "windows"))]
    {
        std::env::var("HOME").ok().map(PathBuf::from)
    }
}

/// Reads an overrides map and reports the step cap a grant imposes.
///
/// A grant that caps steps must not be bypassable by the caller asking for a
/// longer run, so the cap is applied as a ceiling rather than a default.
pub fn apply_step_cap(overrides: &HashMap<String, Value>, max_steps: u32) -> HashMap<String, Value> {
    if max_steps == 0 {
        return overrides.clone();
    }
    let mut capped = overrides.clone();
    let requested = capped
        .get("max_steps")
        .and_then(Value::as_u64)
        .unwrap_or(max_steps as u64);
    let effective = requested.min(max_steps as u64);
    capped.insert(
        "max_steps".to_string(),
        Value::from(effective),
    );
    capped
}
