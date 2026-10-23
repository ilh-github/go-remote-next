//! Cross-machine execution claims for queued workflow runs.
//!
//! The local-file claim that shipped first is only meaningful between processes
//! on one machine: `create_new` on a path is atomic for the local filesystem and
//! says nothing about a worker on another host that cannot see that directory at
//! all. Two workers on two machines would each open their own claim file, both
//! believe they own the run, and both drive the same remote desktop.
//!
//! `DistributedClaimStore` puts the claim in a SQLite database instead, which is
//! the only shared-state primitive this binary already depends on. A single
//! `INSERT` is the mutual exclusion: exactly one worker can insert a row for a
//! run id, and the loser gets a constraint violation rather than a second
//! claim. The heartbeat column is what makes a crashed worker recoverable -
//! without it a claim held by a process that died would never be released, and
//! the run would sit unclaimed forever.
//!
//! A store with no database path is a deliberate no-op that reports "not
//! claimed": the single-machine deployment keeps using the file claim it already
//! has, and nothing changes for it.
use std::path::Path;
use std::time::{SystemTime, UNIX_EPOCH};

use rusqlite::{params, Connection};

/// How long a claim may go without a heartbeat before another worker may take
/// it over. Deliberately longer than a normal step: a worker in the middle of a
/// slow tool call must not lose its run to a worker that merely polled faster.
pub const CLAIM_STALE_AFTER_SECONDS: i64 = 300;

// Hand-written rather than `thiserror`: the crate is built offline, and one
// dependency for three variants is not worth adding to that constraint.
#[derive(Debug)]
pub enum ClaimError {
    Open(String),
    Schema(String),
    Query(String),
}

impl std::fmt::Display for ClaimError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            ClaimError::Open(detail) => write!(f, "cannot open claim database: {detail}"),
            ClaimError::Schema(detail) => {
                write!(f, "cannot initialise claim schema: {detail}")
            }
            ClaimError::Query(detail) => write!(f, "claim query failed: {detail}"),
        }
    }
}

impl std::error::Error for ClaimError {}
/// One claim on one run, as stored.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ClaimRecord {
    pub run_id: String,
    /// Identifies the worker holding it, so a takeover can tell "my own stale
    /// claim" from "somebody else's dead one".
    pub worker_id: String,
    pub heartbeat_at: i64,
}

// SQLite-backed execution claims, safe across machines when the database sits
// on storage they share.
#[derive(Clone)]
pub struct DistributedClaimStore {
    db_path: Option<std::path::PathBuf>,
}

impl DistributedClaimStore {
    /// A store with no database. Every operation reports "not claimed", which is
    /// what lets a deployment opt out without a second code path.
    #[allow(dead_code)]
    pub fn disabled() -> Self {
        Self { db_path: None }
    }

    pub fn open(path: impl AsRef<Path>) -> Result<Self, ClaimError> {
        let path = path.as_ref().to_path_buf();
        if let Some(parent) = path.parent() {
            if !parent.as_os_str().is_empty() {
                std::fs::create_dir_all(parent).map_err(|error| {
                    ClaimError::Open(format!("{}: {error}", parent.display()))
                })?;
            }
        }
        let store = Self { db_path: Some(path) };
        store.with_connection(|conn| store.init_schema(conn))?;
        Ok(store)
    }

    pub fn is_enabled(&self) -> bool {
        self.db_path.is_some()
    }

    fn init_schema(&self, conn: &Connection) -> Result<(), ClaimError> {
        conn.execute_batch(
            "CREATE TABLE IF NOT EXISTS workflow_execution_claims (
                 run_id       TEXT PRIMARY KEY,
                 worker_id    TEXT NOT NULL,
                 heartbeat_at INTEGER NOT NULL
             );
             CREATE INDEX IF NOT EXISTS idx_workflow_claims_heartbeat
                 ON workflow_execution_claims(heartbeat_at);",
        )
        .map_err(|error| ClaimError::Schema(error.to_string()))
    }

    /// Takes the claim for `run_id`, or reports that another worker holds it.
    ///
    /// A claim whose heartbeat has gone stale is taken over, because a worker
    /// that stopped reporting is not coming back for this run. The takeover is
    /// an `UPDATE` guarded on the heartbeat, not a blind overwrite: two workers
    /// that both see a stale row must not both win.
    pub fn try_claim(
        &self,
        run_id: &str,
        worker_id: &str,
    ) -> Result<bool, ClaimError> {
        if self.db_path.is_none() {
            return Ok(false);
        }
        let run_id = run_id.trim();
        if run_id.is_empty() {
            return Ok(false);
        }
        let now = now_seconds();
        let stale_cutoff = now - CLAIM_STALE_AFTER_SECONDS;
        self.with_connection(|conn| {
            // Insert first: the primary key is the mutual exclusion, so the
            // common case is settled without a second round trip.
            let inserted = conn
                .execute(
                    "INSERT OR IGNORE INTO workflow_execution_claims
                         (run_id, worker_id, heartbeat_at) VALUES (?1, ?2, ?3)",
                    params![run_id, worker_id, now],
                )
                .map_err(|error| ClaimError::Query(error.to_string()))?;
            if inserted > 0 {
                return Ok(true);
            }

            // A row already exists. It is ours to keep if it is ours and fresh.
            let existing = conn
                .query_row(
                    "SELECT worker_id, heartbeat_at FROM workflow_execution_claims
                         WHERE run_id = ?1",
                    params![run_id],
                    |row| {
                        Ok((
                            row.get::<_, String>(0)?,
                            row.get::<_, i64>(1)?,
                        ))
                    },
                )
                .map_err(|error| ClaimError::Query(error.to_string()))?;
            if existing.0 == worker_id {
                Self::touch(conn, run_id, worker_id, now)?;
                return Ok(true);
            }
            if existing.1 > stale_cutoff {
                return Ok(false);
            }

            // Takeover. The heartbeat guard is what makes it safe: only the
            // worker that actually observed the stale value wins, so a race
            // between two of them resolves to one.
            let taken = conn
                .execute(
                    "UPDATE workflow_execution_claims
                         SET worker_id = ?1, heartbeat_at = ?2
                         WHERE run_id = ?3 AND heartbeat_at <= ?4",
                    params![worker_id, now, run_id, stale_cutoff],
                )
                .map_err(|error| ClaimError::Query(error.to_string()))?;
            Ok(taken > 0)
        })
    }

    fn touch(
        conn: &Connection,
        run_id: &str,
        worker_id: &str,
        now: i64,
    ) -> Result<(), ClaimError> {
        conn.execute(
            "UPDATE workflow_execution_claims SET heartbeat_at = ?1
                 WHERE run_id = ?2 AND worker_id = ?3",
            params![now, run_id, worker_id],
        )
        .map_err(|error| ClaimError::Query(error.to_string()))?;
        Ok(())
    }

    /// Refreshes the heartbeat so a long-running run is not mistaken for a dead
    /// worker.
    pub fn heartbeat(&self, run_id: &str, worker_id: &str) -> Result<(), ClaimError> {
        let Some(_) = self.db_path.as_ref() else {
            return Ok(());
        };
        self.with_connection(|conn| Self::touch(conn, run_id, worker_id, now_seconds()))
    }

    /// Releases the claim. Only the holder may release it, so a worker that
    /// lost a takeover cannot delete the row that now belongs to somebody else.
    pub fn release(&self, run_id: &str, worker_id: &str) -> Result<(), ClaimError> {
        let Some(_) = self.db_path.as_ref() else {
            return Ok(());
        };
        self.with_connection(|conn| {
            conn.execute(
                "DELETE FROM workflow_execution_claims WHERE run_id = ?1 AND worker_id = ?2",
                params![run_id, worker_id],
            )
            .map_err(|error| ClaimError::Query(error.to_string()))?;
            Ok(())
        })
    }

    /// Who holds the run right now, if anybody.
    ///
    /// Part of the store's public surface rather than an internal helper: the
    /// question "why is my run not starting" is answered by this, and an
    /// operator should not have to open the database to ask it.
    #[allow(dead_code)]
    pub fn holder(&self, run_id: &str) -> Result<Option<ClaimRecord>, ClaimError> {
        let Some(_) = self.db_path.as_ref() else {
            return Ok(None);
        };
        self.with_connection(|conn| {
            conn.query_row(
                "SELECT run_id, worker_id, heartbeat_at FROM workflow_execution_claims
                     WHERE run_id = ?1",
                params![run_id],
                |row| {
                    Ok(ClaimRecord {
                        run_id: row.get(0)?,
                        worker_id: row.get(1)?,
                        heartbeat_at: row.get(2)?,
                    })
                },
            )
            .map(Some)
            .or_else(|error| match error {
                rusqlite::Error::QueryReturnedNoRows => Ok(None),
                other => Err(ClaimError::Query(other.to_string())),
            })
        })
    }

    /// Ages a claim's heartbeat so it reads as a dead worker's.
    ///
    /// Exposed for tests only: a takeover path that can only be reached by
    /// waiting five minutes is a path nobody verifies, and a test that sleeps is
    /// a test that gets skipped.
    #[cfg(test)]
    pub(crate) fn age_claim_for_test(&self, run_id: &str, heartbeat_at: i64) -> Result<(), ClaimError> {
        self.with_connection(|conn| {
            conn.execute(
                "UPDATE workflow_execution_claims SET heartbeat_at = ?1 WHERE run_id = ?2",
                params![heartbeat_at, run_id],
            )
            .map_err(|error| ClaimError::Query(error.to_string()))?;
            Ok(())
        })
    }

    fn with_connection<T>(
        &self,
        operation: impl FnOnce(&Connection) -> Result<T, ClaimError>,
    ) -> Result<T, ClaimError> {
        let path = self
            .db_path
            .as_ref()
            .expect("with_connection requires an enabled store");
        let conn = Connection::open(path).map_err(|error| ClaimError::Open(error.to_string()))?;
        // A shared database is written by several processes, so the busy timeout
        // is what keeps a claim attempt from failing on a lock another worker
        // holds for the length of one statement.
        conn.busy_timeout(std::time::Duration::from_secs(5))
            .map_err(|error| ClaimError::Open(error.to_string()))?;
        operation(&conn)
    }
}

/// A held claim, kept alive by a background heartbeat and released on drop.
///
/// Without the heartbeat a run whose steps take longer than
/// `CLAIM_STALE_AFTER_SECONDS` would look like a dead worker's, and another
/// machine would take the run over underneath it - two workers driving one
/// desktop again, just delayed. Without the release-on-drop a worker that
/// finished normally would leave the row behind until it aged out, and the run
/// would sit unclaimable for five minutes for no reason.
pub struct ClaimLease {
    store: DistributedClaimStore,
    run_id: String,
    worker_id: String,
    cancel: tokio_util::sync::CancellationToken,
    task: Option<tokio::task::JoinHandle<()>>,
}

impl ClaimLease {
    /// Takes the lease for `run_id`, or `None` when another worker holds it.
    pub async fn acquire(
        store: &DistributedClaimStore,
        run_id: &str,
        worker_id: &str,
    ) -> Result<Option<Self>, ClaimError> {
        if !store.is_enabled() {
            return Ok(None);
        }
        if !store.try_claim(run_id, worker_id)? {
            return Ok(None);
        }
        let cancel = tokio_util::sync::CancellationToken::new();
        let heartbeat_store = store.clone();
        let heartbeat_run = run_id.to_string();
        let heartbeat_worker = worker_id.to_string();
        let heartbeat_cancel = cancel.clone();
        // Well inside the stale window: the point is that a live worker never
        // looks dead, not that the row is refreshed as late as possible.
        let period = std::time::Duration::from_secs(CLAIM_STALE_AFTER_SECONDS as u64 / 3);
        let task = tokio::spawn(async move {
            let mut ticker = tokio::time::interval(period);
            ticker.tick().await;
            loop {
                tokio::select! {
                    _ = heartbeat_cancel.cancelled() => return,
                    _ = ticker.tick() => {
                        if let Err(error) =
                            heartbeat_store.heartbeat(&heartbeat_run, &heartbeat_worker)
                        {
                            tracing::warn!(
                                "cannot heartbeat claim {}: {error}",
                                heartbeat_run
                            );
                            return;
                        }
                    }
                }
            }
        });
        Ok(Some(Self {
            store: store.clone(),
            run_id: run_id.to_string(),
            worker_id: worker_id.to_string(),
            cancel,
            task: Some(task),
        }))
    }
}

impl Drop for ClaimLease {
    fn drop(&mut self) {
        self.cancel.cancel();
        if let Some(task) = self.task.take() {
            task.abort();
        }
        // Only the holder may release, which is what keeps a worker that lost a
        // takeover from deleting the row that now belongs to somebody else.
        if let Err(error) = self.store.release(&self.run_id, &self.worker_id) {
            tracing::warn!(
                "cannot release claim {} on drop: {error}",
                self.run_id
            );
        }
    }
}

fn now_seconds() -> i64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|elapsed| elapsed.as_secs() as i64)
        .unwrap_or_default()
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::Arc;

    fn store(name: &str) -> DistributedClaimStore {
        let dir = std::env::temp_dir().join(format!("qd-claim-{}-{}", name, uuid::Uuid::new_v4()));
        DistributedClaimStore::open(dir.join("claims.db")).expect("open claim store")
    }

    // The whole reason this module exists: two workers, one run, one winner.
    // With the file claim this was unenforceable across machines because neither
    // worker could see the other's claim file at all.
    #[test]
    fn only_one_worker_claims_a_run() {
        let store = store("exclusive");
        assert!(store.try_claim("run-1", "worker-a").expect("claim a"));
        assert!(
            !store.try_claim("run-1", "worker-b").expect("claim b"),
            "a second worker must not claim a run that is already held"
        );
        assert_eq!(
            store.holder("run-1").expect("holder").map(|row| row.worker_id),
            Some("worker-a".to_string())
        );
    }

    // A worker that lost a takeover must not be able to release the row that now
    // belongs to the winner.
    #[test]
    fn only_the_holder_may_release() {
        let store = store("release");
        assert!(store.try_claim("run-1", "worker-a").expect("claim"));
        store.release("run-1", "worker-b").expect("foreign release");
        assert!(
            store.holder("run-1").expect("holder").is_some(),
            "a foreign release deleted somebody else's claim"
        );
        store.release("run-1", "worker-a").expect("own release");
        assert!(store.holder("run-1").expect("holder").is_none());
    }

    // Re-claiming a run one already holds is not a conflict: the worker is still
    // working on it, and refusing would look exactly like losing it to a rival.
    #[test]
    fn a_worker_may_reclaim_its_own_run() {
        let store = store("reclaim");
        assert!(store.try_claim("run-1", "worker-a").expect("first"));
        assert!(
            store.try_claim("run-1", "worker-a").expect("second"),
            "a worker lost its own live claim"
        );
    }

    // A dead worker's claim must be recoverable, or the run sits unclaimed
    // forever. The heartbeat is the liveness signal, and the takeover is guarded
    // on the value that was read so two workers racing on the same stale row
    // cannot both win.
    #[test]
    fn a_stale_claim_is_taken_over_by_exactly_one_worker() {
        let store = store("stale");
        assert!(store.try_claim("run-1", "worker-a").expect("claim"));
        let stale = now_seconds() - CLAIM_STALE_AFTER_SECONDS - 1;
        store
            .with_connection(|conn| {
                conn.execute(
                    "UPDATE workflow_execution_claims SET heartbeat_at = ?1 WHERE run_id = ?2",
                    params![stale, "run-1"],
                )
                .map_err(|error| ClaimError::Query(error.to_string()))
            })
            .expect("age the claim");

        assert!(
            store.try_claim("run-1", "worker-b").expect("takeover b"),
            "a stale claim was not recoverable"
        );
        assert!(
            !store.try_claim("run-1", "worker-c").expect("takeover c"),
            "two workers took over the same stale claim"
        );
        assert_eq!(
            store.holder("run-1").expect("holder").map(|row| row.worker_id),
            Some("worker-b".to_string())
        );
    }

    // A heartbeat keeps a live claim alive past the stale window, which is what
    // lets a long step finish without another worker stealing the run.
    #[test]
    fn a_heartbeat_keeps_a_claim_alive() {
        let store = store("heartbeat");
        assert!(store.try_claim("run-1", "worker-a").expect("claim"));
        let aged = now_seconds() - CLAIM_STALE_AFTER_SECONDS + 30;
        store
            .with_connection(|conn| {
                conn.execute(
                    "UPDATE workflow_execution_claims SET heartbeat_at = ?1 WHERE run_id = ?2",
                    params![aged, "run-1"],
                )
                .map_err(|error| ClaimError::Query(error.to_string()))
            })
            .expect("age the claim");
        store.heartbeat("run-1", "worker-a").expect("heartbeat");

        assert!(
            !store.try_claim("run-1", "worker-b").expect("rival claim"),
            "a heartbeating claim was treated as dead"
        );
    }

    // Different runs are different claims; a busy worker on one must not block
    // another worker on an unrelated one.
    #[test]
    fn claims_are_per_run() {
        let store = store("per-run");
        assert!(store.try_claim("run-1", "worker-a").expect("claim 1"));
        assert!(store.try_claim("run-2", "worker-b").expect("claim 2"));
        assert_eq!(
            store.holder("run-2").expect("holder").map(|row| row.worker_id),
            Some("worker-b".to_string())
        );
    }

    // An empty run id is not a run, and claiming it would create a row that can
    // never be matched to work.
    #[test]
    fn an_empty_run_id_is_refused() {
        let store = store("empty");
        assert!(!store.try_claim("   ", "worker-a").expect("claim"));
        assert!(store.holder("").expect("holder").is_none());
    }

    // A deployment with no shared database keeps its existing behaviour: the
    // caller falls back to the local file claim, and nothing here interferes.
    // The lease releases on drop. Without that a worker that finished normally
    // would leave its row behind until it aged out, and the run would sit
    // unclaimable for five minutes for no reason at all.
    #[tokio::test]
    async fn a_lease_releases_when_it_is_dropped() {
        let store = store("lease-drop");
        let lease = ClaimLease::acquire(&store, "run-1", "worker-a")
            .await
            .expect("acquire")
            .expect("a lease");
        assert!(store.holder("run-1").expect("holder").is_some());

        drop(lease);
        assert!(
            store.holder("run-1").expect("holder").is_none(),
            "dropping the lease left the claim behind"
        );
    }

    // A second worker must wait for the lease, which is the same guarantee as the
    // raw claim but through the path the worker actually uses.
    #[tokio::test]
    async fn a_second_worker_cannot_take_the_leased_run() {
        let store = store("lease-exclusive");
        let lease = ClaimLease::acquire(&store, "run-1", "worker-a")
            .await
            .expect("acquire a")
            .expect("a lease");
        let second = ClaimLease::acquire(&store, "run-1", "worker-b")
            .await
            .expect("acquire b");
        assert!(second.is_none(), "two workers held the same lease");

        drop(lease);
        let third = ClaimLease::acquire(&store, "run-1", "worker-c")
            .await
            .expect("acquire c");
        assert!(third.is_some(), "the run was not reclaimable after the lease ended");
    }

    // A disabled store yields no lease rather than an error, which is what keeps
    // the single-machine deployment on the code path it already had.
    #[tokio::test]
    async fn a_disabled_store_yields_no_lease() {
        let store = DistributedClaimStore::disabled();
        let lease = ClaimLease::acquire(&store, "run-1", "worker-a")
            .await
            .expect("acquire");
        assert!(lease.is_none());
    }

    // Real concurrency, not a simulated one. Several workers racing on the same
    // database at the same instant is the situation the store exists for, and a
    // sequential test would pass even if the INSERT were not atomic. SQLite is
    // what makes this hold across processes as well as threads: the statement is
    // serialised by the database, not by this process's memory.
    #[test]
    fn concurrent_claims_on_one_database_yield_exactly_one_winner() {
        let store = store("concurrent");
        let winners = Arc::new(std::sync::atomic::AtomicUsize::new(0));
        let mut handles = Vec::new();
        for index in 0..8 {
            let store = store.clone();
            let winners = Arc::clone(&winners);
            handles.push(std::thread::spawn(move || {
                if store
                    .try_claim("run-race", &format!("worker-{index}"))
                    .expect("claim")
                {
                    winners.fetch_add(1, std::sync::atomic::Ordering::SeqCst);
                }
            }));
        }
        for handle in handles {
            handle.join().expect("worker thread");
        }
        assert_eq!(
            winners.load(std::sync::atomic::Ordering::SeqCst),
            1,
            "more than one worker won the same run"
        );
        assert!(
            store
                .holder("run-race")
                .expect("holder")
                .is_some_and(|row| !row.worker_id.is_empty()),
            "the winner must be recorded"
        );
    }

    // Cross-process, not just cross-thread. The threads test above proves the
    // INSERT is atomic inside this process; this proves the *database* is what
    // enforces it, by having a genuinely separate process - node, opening the
    // same file with its own SQLite - try to take a run this process holds.
    //
    // That is the deployment the store exists for: a second worker on a second
    // machine is another process, and nothing about this process's memory
    // protects against it.
    #[test]
    fn a_separate_process_cannot_take_a_held_run() {
        let store = store("cross-process");
        store.try_claim("run-cross", "worker-a").expect("claim");

        // node is the second process; `node:sqlite` is built in from Node 22, so
        // the probe needs no dependency of its own.
        let probe = r#"
const { DatabaseSync } = require('node:sqlite');
const db = new DatabaseSync(process.argv[2], { readOnly: false });
db.exec('PRAGMA busy_timeout = 5000;');
const before = db.prepare('SELECT COUNT(*) AS n FROM workflow_execution_claims').get().n;
const inserted = db.prepare(
  'INSERT OR IGNORE INTO workflow_execution_claims (run_id, worker_id, heartbeat_at) VALUES (?, ?, ?)'
).run('run-cross', 'worker-b', Math.floor(Date.now() / 1000)).changes;
const after = db.prepare('SELECT COUNT(*) AS n FROM workflow_execution_claims').get().n;
console.log(JSON.stringify({ before, inserted, after }));
"#;
        let dir = std::env::temp_dir().join(format!("qd-claim-probe-{}", uuid::Uuid::new_v4()));
        std::fs::create_dir_all(&dir).expect("probe dir");
        let script = dir.join("probe.cjs");
        std::fs::write(&script, probe).expect("write probe");
        let db_path = store
            .db_path
            .as_ref()
            .expect("an enabled store has a path")
            .clone();

        let output = std::process::Command::new("node")
            .arg(&script)
            .arg(&db_path)
            .output();
        let Ok(output) = output else {
            // No node on this machine: the threads test already covers the
            // atomicity, and skipping beats failing for an environment reason.
            eprintln!("skipping cross-process probe: node is unavailable");
            let _ = std::fs::remove_dir_all(&dir);
            return;
        };
        assert!(
            output.status.success(),
            "the probe process failed: {}",
            String::from_utf8_lossy(&output.stderr)
        );
        let stdout = String::from_utf8_lossy(&output.stdout);
        let report: serde_json::Value =
            serde_json::from_str(stdout.trim()).expect("probe printed JSON");
        assert_eq!(report["before"], 1, "this process must hold the run first");
        assert_eq!(
            report["inserted"], 0,
            "a separate process inserted a row for a run that is already held"
        );
        assert_eq!(report["after"], 1, "the held run must still have exactly one row");
        assert_eq!(
            store.holder("run-cross").expect("holder").map(|row| row.worker_id),
            Some("worker-a".to_string()),
            "the holder must not have changed"
        );

        let _ = std::fs::remove_dir_all(&dir);
    }

    #[test]
    fn a_disabled_store_claims_nothing() {
        let store = DistributedClaimStore::disabled();
        assert!(!store.is_enabled());
        assert!(!store.try_claim("run-1", "worker-a").expect("claim"));
        assert!(store.holder("run-1").expect("holder").is_none());
        store.heartbeat("run-1", "worker-a").expect("heartbeat");
        store.release("run-1", "worker-a").expect("release");
    }
}
