package models

import "time"

// AuditOnceActions are the actions whose rows carry a permanent identity: one
// row per transfer id, whichever way it went and however often it is retried.
//
// The list is the scope of the unique index, not just documentation. Media reads
// (`LogActorDeduped`, recording.download and friends) deliberately write several
// rows for the same resource over a time window - that is what a window is for -
// so a unique index covering every actor row would forbid exactly that. Scoping
// the index to the two transfer actions keeps both behaviours.
var AuditOnceActions = []string{"file.transfer", "clipboard.sync"}

// AuditActorIdentity is the unique index that makes `LogActorOnce` hold.
//
// Without it the "accepted only once" promise is a count followed by an insert,
// and two concurrent reports of the same transfer both see zero and both write.
// A retry is exactly when two reports overlap, so that is the path that matters.
//
// It is partial three ways, and each one is load-bearing:
//
//   - actor_user_id > 0, because admin rows are a different story - an admin
//     legitimately repeats the same action on the same resource, and a unique
//     constraint there would silently drop those rows.
//   - action IN AuditOnceActions, because the media-read audit deliberately
//     writes several rows per resource over a time window.
//   - resource_id <> ”, because a row that names no resource has no stable
//     identity to be unique on, and several genuinely distinct events carry an
//     empty one.
//
// AutoMigrate cannot express a partial index, which is why the DDL lives in
// database.EnsureAuditActorIdentity alongside the log-search indexes, and is
// mirrored in migrations/001_init.sql for installs that bootstrap from the file.
const AuditActorIdentity = "uni_audit_logs_actor_identity"

type AuditLog struct {
	ID uint `gorm:"primaryKey" json:"id"`
	// AdminID and AdminUsername identify the administrator actor. They stay
	// zero for actions taken by an end user.
	AdminID       uint   `gorm:"index" json:"admin_id"`
	AdminUsername string `gorm:"size:100;index" json:"admin_username"`
	// ActorUserID identifies the end-user actor when the action came from a
	// user rather than an admin. Two columns rather than one "actor" because
	// the two live in different tables and callers filter on one or the other;
	// overloading admin_id with a user id would make "which admin did this?"
	// unanswerable.
	ActorUserID  uint      `gorm:"index;not null;default:0" json:"actor_user_id"`
	Action       string    `gorm:"size:50;index" json:"action"`
	ResourceType string    `gorm:"size:50" json:"resource_type"`
	ResourceID   string    `gorm:"size:100" json:"resource_id"`
	Details      string    `gorm:"type:text" json:"details"`
	IP           string    `gorm:"size:50" json:"ip"`
	CreatedAt    time.Time `json:"created_at"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}
