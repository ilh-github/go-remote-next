package database

import (
	"context"
	"log"
	"strings"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// The statements below are what make the shipped-log search indexable. They are
// deliberately separate and idempotent, because the two halves can fail
// independently on a managed Postgres:
//
//   - the trigram index accelerates the `q=` substring search that has always
//     existed, and needs the pg_trgm extension;
//   - the generated tsvector column plus its GIN index back the newer `match=`
//     full-text search, and needs nothing but a recent server.
//
// Neither is created by GORM's AutoMigrate (it manages columns it knows about,
// not expression indexes or generated columns), so they are ensured explicitly
// on startup. A deployment where either fails keeps working: the queries are
// still correct, they just fall back to a sequential scan, and the failure is
// logged once at boot rather than silently.
const (
	createTrigramExtension = `CREATE EXTENSION IF NOT EXISTS pg_trgm`
	createTrigramIndex     = `CREATE INDEX IF NOT EXISTS idx_instance_log_entries_line_trgm
		ON instance_log_entries USING gin (line gin_trgm_ops)`
	createSearchVectorColumn = `ALTER TABLE instance_log_entries
		ADD COLUMN IF NOT EXISTS line_tsv tsvector
		GENERATED ALWAYS AS (to_tsvector('simple', line)) STORED`
	createSearchVectorIndex = `CREATE INDEX IF NOT EXISTS idx_instance_log_entries_line_tsv
		ON instance_log_entries USING gin (line_tsv)`
)

// EnsureLogSearchIndexes creates the search indexes if they are not there yet.
// It reports what it managed to do, and never returns an error: a missing index
// is a performance problem, and refusing to start over one would turn it into
// an outage.
func EnsureLogSearchIndexes(ctx context.Context, db *gorm.DB) {
	if db == nil {
		return
	}
	// The extension has to exist before the operator class `gin_trgm_ops` can be
	// named, so the two are one unit.
	if err := db.WithContext(ctx).Exec(createTrigramExtension).Error; err != nil {
		log.Printf("Warning: pg_trgm is unavailable (%v); substring log search stays unindexed", err)
	} else if err := db.WithContext(ctx).Exec(createTrigramIndex).Error; err != nil {
		log.Printf("Warning: could not create the trigram index on instance_log_entries.line: %v", err)
	} else {
		// `q=` keeps its substring semantics; the index only makes it fast, so
		// the message says "substring", not "search".
		log.Println("Log search index: substring (q=) queries are index-backed by pg_trgm")
	}

	if err := db.WithContext(ctx).Exec(createSearchVectorColumn).Error; err != nil {
		log.Printf("Warning: could not add instance_log_entries.line_tsv (%v); match= queries would fail", err)
		return
	}
	if err := db.WithContext(ctx).Exec(createSearchVectorIndex).Error; err != nil {
		log.Printf("Warning: could not index instance_log_entries.line_tsv: %v", err)
		return
	}
	log.Println("Log search index: full-text (match=) queries are index-backed")
}

// createAuditActorIdentity is the partial unique index behind `LogActorOnce`.
//
// AutoMigrate cannot express a partial index, and the "accepted only once"
// promise is worthless without one: the service checks with a count and then
// inserts, so two concurrent reports of the same transfer both see zero and both
// write. This is what makes the second insert fail instead.
//
// Partial on actor_user_id > 0 because admin rows are a different story - an
// admin legitimately repeats the same action on the same resource, and a unique
// constraint there would silently drop those rows.
//
// Also partial on a non-empty resource_id: a row that names no resource has no
// stable identity to be unique on, and several genuinely distinct events carry an
// empty one. The two callers that rely on this guarantee - the file-transfer and
// clipboard reports - always supply a transfer id.
var createAuditActorIdentity = `CREATE UNIQUE INDEX IF NOT EXISTS ` + models.AuditActorIdentity + `
	ON audit_logs (actor_user_id, action, resource_type, resource_id)
	WHERE actor_user_id > 0
	  AND action IN ('` + strings.Join(models.AuditOnceActions, "','") + `')
	  AND resource_id <> ''`

// EnsureAuditActorIdentity creates the partial unique index if it is not there
// yet, and reports whether it is in place.
//
// Unlike the log-search indexes this one is not merely a performance concern:
// without it the audit trail can gain a second row for one transfer id whenever
// two reports overlap. It still does not abort startup, because a server that
// boots without it is strictly better than one that does not boot - but the
// caller is told, so a deployment is not left believing a guarantee it does not
// have.
func EnsureAuditActorIdentity(ctx context.Context, db *gorm.DB) bool {
	if db == nil {
		return false
	}
	if err := db.WithContext(ctx).Exec(createAuditActorIdentity).Error; err != nil {
		log.Printf("Warning: could not create %s (%v); a retried report can produce a second audit row",
			models.AuditActorIdentity, err)
		return false
	}
	return true
}

// createAuditActorIdentitySQLite is the same statement for SQLite. SQLite does
// support partial indexes, so the shape is identical rather than a workaround -
// which matters, because two different shapes are two different guarantees and
// the one that gets tested should be the one that ships.
var createAuditActorIdentitySQLite = createAuditActorIdentity

// EnsureAuditActorIdentitySQLite applies the index on a SQLite handle. Separate
// from the Postgres entry point so a caller picks the driver explicitly rather
// than relying on a fallback that would leave a deployment running a different
// shape than the one that was tested.
func EnsureAuditActorIdentitySQLite(ctx context.Context, db *gorm.DB) bool {
	if db == nil {
		return false
	}
	if err := db.WithContext(ctx).Exec(createAuditActorIdentitySQLite).Error; err != nil {
		log.Printf("Warning: could not create %s on SQLite (%v); a retried report can produce a second audit row",
			models.AuditActorIdentity, err)
		return false
	}
	return true
}
