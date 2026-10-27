package service

import (
	"context"
	"errors"
	"log"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// SessionRevokeCommandStore is the durable half of a cross-instance revoke.
//
// The revoke path reads a session row and then publishes a command to the
// instance holding that session's socket. Pub/sub is fire-and-forget, so a
// process that dies in between leaves the command nowhere: the operator's kick
// is gone, the socket is still open, and the only thing that eventually closes
// it is the presence-lease sweep about a minute later. Recording the command
// first means the decision outlives the process that made it.
//
// The store deliberately does not deliver anything. It answers three questions -
// what is still owed, what was attempted, and what is no longer relevant - and
// the delivery loop lives next to the channel it uses.
type SessionRevokeCommandStore struct {
	db *gorm.DB
}

func NewSessionRevokeCommandStore(db *gorm.DB) *SessionRevokeCommandStore {
	return &SessionRevokeCommandStore{db: db}
}

// Record writes the command before it is delivered. It is idempotent per
// session: a second revoke for a session that already has an undelivered command
// returns that command rather than queueing a duplicate, because two commands
// that close the same socket is one close plus one row nobody can explain.
func (s *SessionRevokeCommandStore) Record(
	ctx context.Context, sessionID, instanceID, reason string,
) (*models.SessionRevokeCommand, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("session revoke command store is not configured")
	}
	sessionID = trimSpace(sessionID)
	instanceID = trimSpace(instanceID)
	if sessionID == "" {
		return nil, errors.New("session_id is required")
	}
	if instanceID == "" {
		// A command that cannot be addressed cannot be delivered, and recording
		// one would only produce a row the delivery loop keeps re-examining.
		return nil, errors.New("instance_id is required to address a revoke command")
	}
	now := time.Now().UTC()

	var existing models.SessionRevokeCommand
	err := s.db.WithContext(ctx).Where("session_id = ?", sessionID).First(&existing).Error
	switch {
	case err == nil:
		if !existing.Delivered {
			return &existing, nil
		}
		// The previous command for this session was delivered, so this is a new
		// decision about a session that is somehow still open. Reuse the row and
		// reset it rather than fighting the unique index.
		existing.Delivered = false
		existing.DeliveredAt = nil
		existing.InstanceID = instanceID
		existing.Reason = reason
		existing.Attempts = 0
		existing.ClaimExpiresAt = nil
		existing.LastError = ""
		existing.UpdatedAt = now
		if err := s.db.WithContext(ctx).Save(&existing).Error; err != nil {
			return nil, err
		}
		return &existing, nil
	case !errors.Is(err, gorm.ErrRecordNotFound):
		return nil, err
	}

	row := models.SessionRevokeCommand{
		SessionID:  sessionID,
		InstanceID: instanceID,
		Reason:     reason,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// Pending lists the commands still owed, in the order they were recorded.
//
// Two filters, both about not acting on something stale:
//   - A command whose session has already ended is dropped. Delivering it would
//     close nothing and would read as though the revoke had worked, when the
//     session had ended for an unrelated reason.
//   - A command whose delivery attempt is still in flight is left alone, so two
//     loops cannot deliver the same command twice.
func (s *SessionRevokeCommandStore) Pending(ctx context.Context) ([]models.SessionRevokeCommand, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("session revoke command store is not configured")
	}
	now := time.Now().UTC()
	var rows []models.SessionRevokeCommand
	err := s.db.WithContext(ctx).
		Where("delivered = ?", false).
		Where("claim_expires_at IS NULL OR claim_expires_at < ?", now).
		Order("id asc").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}

	// Drop the ones whose session is already over. The session table is the only
	// thing that knows that, so this is a join rather than a column on the
	// command row: a command that outlived its session must stop being owed, and
	// keeping the two in step by hand would be a second source of truth.
	sessionIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		sessionIDs = append(sessionIDs, row.SessionID)
	}
	var open []string
	if err := s.db.WithContext(ctx).Model(&models.RemoteSession{}).
		Where("session_id IN ? AND ended_at IS NULL", sessionIDs).
		Pluck("session_id", &open).Error; err != nil {
		return nil, err
	}
	stillOpen := make(map[string]bool, len(open))
	for _, id := range open {
		stillOpen[id] = true
	}

	out := make([]models.SessionRevokeCommand, 0, len(rows))
	for _, row := range rows {
		if stillOpen[row.SessionID] {
			out = append(out, row)
		}
	}
	return out, nil
}

// Claim marks a delivery attempt as in flight so a concurrent loop skips it. The
// lease is what makes a crashed attempt recoverable: without it, an attempt that
// died mid-delivery would never be retried, which is the same lost command the
// store exists to prevent.
func (s *SessionRevokeCommandStore) Claim(ctx context.Context, id uint, lease time.Duration) error {
	if s == nil || s.db == nil {
		return errors.New("session revoke command store is not configured")
	}
	if lease <= 0 {
		return errors.New("claim lease must be positive")
	}
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Model(&models.SessionRevokeCommand{}).
		Where("id = ? AND delivered = ?", id, false).
		Updates(map[string]interface{}{
			"attempts":         gorm.Expr("attempts + 1"),
			"claim_expires_at": now.Add(lease),
			"updated_at":       now,
		}).Error
}

// MarkDelivered records that the command reached the owning instance. It keeps
// the row: a command that was delivered but whose socket close was never
// confirmed has to stay distinguishable from one nobody attempted.
func (s *SessionRevokeCommandStore) MarkDelivered(ctx context.Context, id uint) error {
	if s == nil || s.db == nil {
		return errors.New("session revoke command store is not configured")
	}
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Model(&models.SessionRevokeCommand{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"delivered":        true,
			"delivered_at":     now,
			"claim_expires_at": nil,
			"updated_at":       now,
		}).Error
}

// MarkDeliveredBySession marks the command for one session delivered and reports
// how many rows changed. Zero means there was nothing to mark - the session
// ended while the command was in flight and something already cleared it, which
// is the outcome the caller wanted anyway.
func (s *SessionRevokeCommandStore) MarkDeliveredBySession(ctx context.Context, sessionID string) (int64, error) {
	if s == nil || s.db == nil {
		return 0, errors.New("session revoke command store is not configured")
	}
	sessionID = trimSpace(sessionID)
	if sessionID == "" {
		return 0, errors.New("session_id is required")
	}
	now := time.Now().UTC()
	res := s.db.WithContext(ctx).Model(&models.SessionRevokeCommand{}).
		Where("session_id = ? AND delivered = ?", sessionID, false).
		Updates(map[string]interface{}{
			"delivered":        true,
			"delivered_at":     now,
			"claim_expires_at": nil,
			"updated_at":       now,
		})
	return res.RowsAffected, res.Error
}

// NoteFailure records why an attempt did not land, and releases the claim so the
// next loop tries again. The error is kept because "this command has failed four
// times" and "this command has failed once" call for different reactions.
func (s *SessionRevokeCommandStore) NoteFailure(ctx context.Context, id uint, reason string) error {
	if s == nil || s.db == nil {
		return errors.New("session revoke command store is not configured")
	}
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Model(&models.SessionRevokeCommand{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"claim_expires_at": nil,
			"last_error":       truncateForColumn(reason, 255),
			"updated_at":       now,
		}).Error
}

// ClearForSession removes the command once the session is over. A session that
// ended on its own has nothing left to revoke, and leaving the row behind would
// have the delivery loop re-examining it forever.
func (s *SessionRevokeCommandStore) ClearForSession(ctx context.Context, sessionID string) error {
	if s == nil || s.db == nil {
		return errors.New("session revoke command store is not configured")
	}
	sessionID = trimSpace(sessionID)
	if sessionID == "" {
		return errors.New("session_id is required")
	}
	return s.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Delete(&models.SessionRevokeCommand{}).Error
}

// All lists every command, delivered or not. It is the read a test or an
// operator uses to answer "was this revoke ever attempted?", which is exactly the
// question the undelivered case used to make unanswerable.
func (s *SessionRevokeCommandStore) All(ctx context.Context) ([]models.SessionRevokeCommand, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("session revoke command store is not configured")
	}
	var rows []models.SessionRevokeCommand
	err := s.db.WithContext(ctx).Order("id asc").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func trimSpace(v string) string {
	start, end := 0, len(v)
	for start < end && (v[start] == ' ' || v[start] == '\t' || v[start] == '\n' || v[start] == '\r') {
		start++
	}
	for end > start && (v[end-1] == ' ' || v[end-1] == '\t' || v[end-1] == '\n' || v[end-1] == '\r') {
		end--
	}
	return v[start:end]
}

// truncateForColumn cuts a string to a column width without splitting a rune: a
// byte cut in the middle of a multi-byte character writes invalid UTF-8 into a
// text column, and Postgres refuses the whole row.
func truncateForColumn(v string, max int) string {
	if len(v) <= max {
		return v
	}
	cut := max
	for cut > 0 && !isASCIIByte(v[cut]) {
		cut--
	}
	return v[:cut]
}

func isASCIIByte(b byte) bool { return b < 0x80 }

// -----------------------------------------------------------------------
// Delivery
// -----------------------------------------------------------------------

// revokeCommandClaimLease bounds one delivery attempt. It is short because an
// attempt that dies is retried quickly: the whole point of the store is that a
// lost command does not wait for the presence-lease sweep.
const revokeCommandClaimLease = 10 * time.Second

// SessionRevokeDeliverer publishes one command to the instance that holds the
// socket. It is the channel half, kept behind an interface so the loop can be
// tested without a Redis connection.
type SessionRevokeDeliverer interface {
	Publish(ctx context.Context, instanceID, sessionID string) error
}

// SessionRevokeCommandDeliverer turns recorded commands into published ones.
//
// It exists because recording a command is only half of closing the gap: a row
// nobody acts on is a more convincing lie than no row at all, since it looks
// like the revoke is being handled. This is the loop that acts on it.
type SessionRevokeCommandDeliverer struct {
	store     *SessionRevokeCommandStore
	channel   SessionRevokeDeliverer
	onDeliver func(sessionID string)
}

func NewSessionRevokeCommandDeliverer(
	store *SessionRevokeCommandStore,
	channel SessionRevokeDeliverer,
	onDeliver func(sessionID string),
) *SessionRevokeCommandDeliverer {
	return &SessionRevokeCommandDeliverer{store: store, channel: channel, onDeliver: onDeliver}
}

// DeliverOnce delivers everything currently owed. It returns how many commands
// it acted on so a test can assert the loop ran without waiting on a ticker.
//
// A command is acted on once per call: claim, publish, mark. A publish that
// fails releases the claim with the reason recorded, so the next call tries
// again rather than treating the attempt as done.
func (d *SessionRevokeCommandDeliverer) DeliverOnce(ctx context.Context) (int, error) {
	if d == nil || d.store == nil || d.channel == nil {
		return 0, errors.New("session revoke deliverer is not configured")
	}
	pending, err := d.store.Pending(ctx)
	if err != nil {
		return 0, err
	}
	delivered := 0
	for i := range pending {
		cmd := &pending[i]
		if err := d.store.Claim(ctx, cmd.ID, revokeCommandClaimLease); err != nil {
			// Another loop holds it. Leave it alone rather than fighting over a
			// command whose delivery is already in flight.
			continue
		}
		if err := d.channel.Publish(ctx, cmd.InstanceID, cmd.SessionID); err != nil {
			_ = d.store.NoteFailure(ctx, cmd.ID, err.Error())
			continue
		}
		if _, err := d.store.MarkDeliveredBySession(ctx, cmd.SessionID); err != nil {
			// The publish landed, so the command is not lost - only the bookkeeping
			// failed. Retrying on the next call is safe because delivery is
			// idempotent: closing a socket that is already closed is a no-op.
			_ = d.store.NoteFailure(ctx, cmd.ID, err.Error())
			continue
		}
		delivered++
		if d.onDeliver != nil {
			d.onDeliver(cmd.SessionID)
		}
	}
	return delivered, nil
}

// Run polls until ctx is cancelled, the same shape as the notification delivery
// worker.
func (d *SessionRevokeCommandDeliverer) Run(ctx context.Context) {
	ticker := time.NewTicker(revokeCommandInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := d.DeliverOnce(ctx); err != nil {
				log.Printf("[session-revoke] delivery pass failed: %v", err)
			}
		}
	}
}

// revokeCommandInterval is how often owed commands are delivered. Short enough
// that a command recorded by a process that then died is delivered within
// seconds rather than the minute the presence sweep takes, long enough that an
// idle fleet does no work at all.
const revokeCommandInterval = 2 * time.Second
