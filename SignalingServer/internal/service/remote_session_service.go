package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RemoteSessionService owns the control plane's record of who is holding a
// signaling session against which device, and when it ended.
//
// It is deliberately the same fact the device access log already records at the
// moment a session starts - but as a row with a lifecycle instead of an
// immutable timeline entry, so "is anyone in this machine right now" and "how
// long was that session" become answerable without reconstructing them from a
// log that never recorded the end.
type RemoteSessionService struct {
	db     *gorm.DB
	outbox EventOutboxWriter
}

func NewRemoteSessionService(db *gorm.DB) *RemoteSessionService {
	return &RemoteSessionService{db: db}
}

// errRemoteSessionExists is the in-transaction signal that the insert lost to a
// row that is already there. It never reaches a caller: Open turns it into the
// existing row. It has to be an error rather than a flag because a Create that
// failed leaves the transaction unusable on Postgres, so the transaction has to
// be rolled back before the existing row can be read.
var errRemoteSessionExists = errors.New("remote session already exists")

func (s *RemoteSessionService) WithOutbox(outbox EventOutboxWriter) *RemoteSessionService {
	s.outbox = outbox
	return s
}

// OpenRemoteSessionInput describes a session that just authenticated.
type OpenRemoteSessionInput struct {
	SessionID string
	DeviceID  string
	ClientID  string
	ClientIP  string
	// InstanceID is the signaling-server process admitting the session. It is
	// what lets a later sweep tell this process's sessions apart from another
	// live instance's, so a restart closes only what really died.
	InstanceID string
}

// Open records a newly established control session.
//
// It is idempotent on the session id: bookkeeping must never be the reason a
// signaling handshake fails, so a duplicate (a retry, or a race between two
// paths that both noticed the same session) returns the row that is already
// there rather than an error.
func (s *RemoteSessionService) Open(ctx context.Context, in OpenRemoteSessionInput) (*models.RemoteSession, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("remote session service is not configured")
	}
	sessionID := strings.TrimSpace(in.SessionID)
	deviceID := strings.TrimSpace(in.DeviceID)
	if sessionID == "" || deviceID == "" {
		return nil, errors.New("session_id and device_id are required")
	}
	var row models.RemoteSession
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row = models.RemoteSession{
			SessionID:  sessionID,
			UserID:     ownerOfDeviceWithDB(ctx, tx, deviceID),
			DeviceID:   deviceID,
			ClientID:   strings.TrimSpace(in.ClientID),
			ClientIP:   strings.TrimSpace(in.ClientIP),
			InstanceID: strings.TrimSpace(in.InstanceID),
			StartedAt:  time.Now().UTC(),
		}
		if err := tx.Create(&row).Error; err != nil {
			if isUniqueViolation(err) {
				return errRemoteSessionExists
			}
			return err
		}
		row.Decorate()
		return s.enqueueRemoteSessionEvent(ctx, tx, EventRemoteSessionStarted, &row)
	})
	if errors.Is(err, errRemoteSessionExists) {
		// A failed Create aborts the transaction on Postgres, so the row that is
		// already there is read back outside it. Reporting the error instead
		// would work on SQLite, which does not abort, and would fail every
		// retried handshake on Postgres.
		if err := s.db.WithContext(ctx).
			Where("session_id = ?", sessionID).
			First(&row).Error; err != nil {
			return nil, err
		}
		row.Decorate()
		return &row, nil
	}
	if err != nil {
		return nil, err
	}
	row.Decorate()
	return &row, nil
}

// ownerOf resolves the device's owner. An unbound device (or a device row that
// has not been created yet) yields 0, and the session is recorded with no owner
// rather than dropped: the signaling side legitimately knows about devices the
// control plane has not bound to an account.
func (s *RemoteSessionService) ownerOf(ctx context.Context, deviceID string) uint {
	return ownerOfDeviceWithDB(ctx, s.db, deviceID)
}

func ownerOfDeviceWithDB(ctx context.Context, db *gorm.DB, deviceID string) uint {
	var device models.Device
	if err := db.WithContext(ctx).
		Select("user_id").
		Where("device_id = ?", deviceID).
		First(&device).Error; err != nil {
		return 0
	}
	if device.UserID == nil {
		return 0
	}
	return *device.UserID
}

func (s *RemoteSessionService) enqueueRemoteSessionEvent(
	ctx context.Context,
	tx *gorm.DB,
	eventType string,
	row *models.RemoteSession,
) error {
	if s.outbox == nil || row == nil || row.UserID == 0 || eventType == "" {
		return nil
	}
	data := map[string]interface{}{
		"session_id":       row.SessionID,
		"device_id":        row.DeviceID,
		"client_id":        row.ClientID,
		"client_ip":        row.ClientIP,
		"started_at":       row.StartedAt,
		"active":           row.Active,
		"duration_seconds": row.DurationSeconds,
	}
	if row.EndedAt != nil {
		data["ended_at"] = *row.EndedAt
		data["end_reason"] = row.EndReason
	}
	_, err := s.outbox.EnqueueTx(ctx, tx, Event{
		Type:     eventType,
		UserID:   row.UserID,
		DeviceID: row.DeviceID,
		Data:     data,
	})
	return err
}

// Close marks a session finished and reports whether this call is the one that
// ended it. Closing twice is not an error - a reconnect, a sweep and a socket
// teardown can all race - but only the first call reports true, so a caller can
// publish exactly one "it ended" event.
func (s *RemoteSessionService) Close(ctx context.Context, sessionID string, reason string) (*models.RemoteSession, bool, error) {
	if s == nil || s.db == nil {
		return nil, false, errors.New("remote session service is not configured")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, false, errors.New("session_id is required")
	}
	var row models.RemoteSession
	changed := false
	now := time.Now().UTC()
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.RemoteSession{}).
			Where("session_id = ? AND ended_at IS NULL", sessionID).
			Updates(map[string]interface{}{
				"ended_at":   now,
				"end_reason": strings.TrimSpace(reason),
				"updated_at": now,
			})
		if result.Error != nil {
			return result.Error
		}
		if err := tx.Where("session_id = ?", sessionID).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		changed = result.RowsAffected > 0
		if !changed {
			return nil
		}
		// A session that ended on its own has nothing left to revoke, so its
		// recorded command goes with it - in the same transaction, because a
		// command that outlived its session would be re-examined by the deliverer
		// forever and would eventually be delivered against a socket that is
		// already gone.
		if err := tx.Where("session_id = ?", sessionID).
			Delete(&models.SessionRevokeCommand{}).Error; err != nil {
			return err
		}
		row.Decorate()
		return s.enqueueRemoteSessionEvent(ctx, tx, EventRemoteSessionEnded, &row)
	})
	if err != nil {
		return nil, false, err
	}
	if row.SessionID == "" {
		return nil, false, nil
	}
	row.Decorate()
	return &row, changed, nil
}

// CloseOrphanedSessions finishes the open sessions whose instance is no longer
// alive, and reports how many it closed.
//
// Closing *every* open row is only correct when there is one instance; with
// two, a restart on one machine would stamp the other's live sessions as
// finished. So the caller passes the set of instances that are currently alive,
// and only rows outside that set are closed.
//
// An empty live set is refused rather than treated as "everybody is dead":
// that is what an unreachable or unconfigured liveness store looks like, and
// answering it by closing every session in the fleet would turn a Redis blip
// into an outage. A running instance always has at least itself in the set.
//
// Rows with no instance id are closed too. They were written before this column
// existed, so nothing is holding them: a row without an owner instance cannot be
// anybody's live session.
func (s *RemoteSessionService) CloseOrphanedSessions(
	ctx context.Context,
	liveInstanceIDs []string,
	reason string,
) (int64, error) {
	if s == nil || s.db == nil {
		return 0, errors.New("remote session service is not configured")
	}
	seen := make(map[string]bool, len(liveInstanceIDs))
	live := make([]string, 0, len(liveInstanceIDs))
	for _, id := range liveInstanceIDs {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		live = append(live, id)
	}
	if len(live) == 0 {
		return 0, errors.New("no live instances were reported; refusing to close every open session")
	}
	now := time.Now().UTC()
	closed := int64(0)
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rows []models.RemoteSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("ended_at IS NULL").
			Where("instance_id IS NULL OR instance_id = '' OR instance_id NOT IN ?", live).
			Find(&rows).Error; err != nil {
			return err
		}
		for i := range rows {
			rows[i].EndedAt = &now
			rows[i].EndReason = strings.TrimSpace(reason)
			rows[i].UpdatedAt = now
			if err := tx.Save(&rows[i]).Error; err != nil {
				return err
			}
			closed++
			rows[i].Decorate()
			if err := s.enqueueRemoteSessionEvent(ctx, tx, EventRemoteSessionEnded, &rows[i]); err != nil {
				return err
			}
			// The same reason Close clears it: a command for a session that is
			// over has nothing left to revoke, and Pending loads every
			// undelivered command on every pass - leaving the row behind makes
			// the delivery loop re-examine it forever. Orphaned sessions are
			// where this matters most, because the instance that held the socket
			// is exactly the one that was never going to receive the command.
			if err := tx.Where("session_id = ?", rows[i].SessionID).
				Delete(&models.SessionRevokeCommand{}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return closed, nil
}

// PurgeEndedSessions deletes finished rows older than retentionDays and reports
// how many went.
//
// QD-207's fourth stage. Open, close and the orphan sweep are all here; what was
// missing was any bound on the table, so every session ever admitted kept its row
// forever. The audit trail already made this argument in its own config: a table
// with no natural terminal state only grows, and retention is the only thing that
// bounds it. `remote_sessions` is that table.
//
// Two boundaries the delete must not cross. It never touches a row whose
// `ended_at` is null, because that session is still open and erasing it would
// make the control plane forget somebody is in there. And the window is measured
// from `ended_at`, not `started_at`: a session that ran for two months and
// finished yesterday is recent history, and measuring from the start would delete
// it.
//
// retentionDays <= 0 keeps everything, the same convention the audit and
// instance-log sweeps use, so a deployment that has not configured retention does
// not get an empty table.
func (s *RemoteSessionService) PurgeEndedSessions(ctx context.Context, retentionDays int) (int64, error) {
	if s == nil || s.db == nil {
		return 0, errors.New("remote session service is not configured")
	}
	if retentionDays <= 0 {
		return 0, nil
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
	res := s.db.WithContext(ctx).
		Where("ended_at IS NOT NULL AND ended_at < ?", cutoff).
		Delete(&models.RemoteSession{})
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}

// RemoteSessionFilter narrows a listing. Zero values mean "no filter", except
// Active, where nil is the zero value and means "either".
type RemoteSessionFilter struct {
	UserID   uint
	DeviceID string
	// Active nil lists both states, true only open sessions, false only finished.
	Active   *bool
	Limit    int
	OffsetID uint
}

// List returns newest first. Rows are keyed by descending id, which is the
// order they were opened in.
func (s *RemoteSessionService) List(ctx context.Context, filter RemoteSessionFilter) ([]models.RemoteSession, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("remote session service is not configured")
	}
	query := s.db.WithContext(ctx).Model(&models.RemoteSession{})
	if filter.UserID != 0 {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if deviceID := strings.TrimSpace(filter.DeviceID); deviceID != "" {
		query = query.Where("device_id = ?", deviceID)
	}
	if filter.Active != nil {
		if *filter.Active {
			query = query.Where("ended_at IS NULL")
		} else {
			query = query.Where("ended_at IS NOT NULL")
		}
	}
	if filter.OffsetID != 0 {
		query = query.Where("id < ?", filter.OffsetID)
	}
	// The bound is only here so a caller cannot ask for an unbounded scan; it is
	// deliberately above the API's own page cap (maxCursorLimit) because the
	// handler asks for one row past the page to learn whether a next page exists.
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > maxRemoteSessionListLimit {
		limit = maxRemoteSessionListLimit
	}
	var out []models.RemoteSession
	if err := query.Order("id DESC").Limit(limit).Find(&out).Error; err != nil {
		return nil, err
	}
	if out == nil {
		out = []models.RemoteSession{}
	}
	for i := range out {
		out[i].Decorate()
	}
	return out, nil
}

// maxRemoteSessionListLimit is the service's own ceiling, kept above the API's
// page cap so the handler's "one extra row" probe is never truncated.
const maxRemoteSessionListLimit = 500

// Get loads one session by its signaling id.
func (s *RemoteSessionService) Get(ctx context.Context, sessionID string) (*models.RemoteSession, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("remote session service is not configured")
	}
	var row models.RemoteSession
	if err := s.db.WithContext(ctx).Where("session_id = ?", strings.TrimSpace(sessionID)).First(&row).Error; err != nil {
		return nil, err
	}
	row.Decorate()
	return &row, nil
}

// GetForUser loads one session the caller owns.
//
// A session belonging to somebody else reports "not found" rather than
// "forbidden": the caller learns nothing about whether that session id exists,
// which is the same choice the device-scoped routes make.
func (s *RemoteSessionService) GetForUser(ctx context.Context, userID uint, sessionID string) (*models.RemoteSession, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("remote session service is not configured")
	}
	if userID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	var row models.RemoteSession
	if err := s.db.WithContext(ctx).
		Where("session_id = ? AND user_id = ?", strings.TrimSpace(sessionID), userID).
		First(&row).Error; err != nil {
		return nil, err
	}
	row.Decorate()
	return &row, nil
}
