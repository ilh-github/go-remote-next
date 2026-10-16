package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// Delivery states for a notification outbox row.
const (
	DeliveryStatusPending    = "pending"
	DeliveryStatusDelivering = "delivering"
	DeliveryStatusDelivered  = "delivered"
	DeliveryStatusFailed     = "failed"
)

// ErrDeliveryNotRetryable means the caller asked to replay a delivery that is
// already delivered (or not parked as failed), so there is nothing to re-arm.
var ErrDeliveryNotRetryable = errors.New("delivery is not retryable")

const (
	// notificationMaxAttempts is how many failed posts a row gets before it
	// is parked as `failed` (still visible through the API).
	notificationMaxAttempts = 8
	// notificationBackoffCap bounds the exponential backoff.
	notificationBackoffCap = 5 * time.Minute
	// notificationClaimBatch is how many due rows one worker pass takes.
	notificationClaimBatch = 20
	// notificationPerChannelBatch caps how many of those rows may come from a
	// single channel. The worker delivers a batch sequentially, so without this
	// cap one endpoint's backlog fills every batch and every other channel's
	// notifications wait behind it (head-of-line blocking).
	notificationPerChannelBatch = 5
	// notificationWorkerInterval is the queue poll interval.
	notificationWorkerInterval = 2 * time.Second
	// notificationPurgeBatch bounds how many rows one archive-then-delete pass
	// reads into memory.
	notificationPurgeBatch = 1000
)

// NotificationDeliveryService is the durable half of user notifications: it
// stores one outbox row per (event, channel) and retries until the endpoint
// answers 2xx. Delivery is at-least-once — a receiver that errors after
// processing may see the event twice, so consumers should treat
// `event` + `timestamp` + `data.log_id` as an idempotency key.
type NotificationDeliveryService struct {
	db *gorm.DB
	// archiveDir, when set, makes retention append expired rows to a JSONL file
	// before deleting them. Empty means the old behaviour: hard delete.
	archiveDir string
}

func NewNotificationDeliveryService(db *gorm.DB) *NotificationDeliveryService {
	return &NotificationDeliveryService{db: db}
}

// WithArchiveDir enables archive-before-delete for retention: expired terminal
// rows are appended to a daily JSONL file under dir, and only deleted once that
// write has been flushed to disk. Returns the receiver so it can be chained.
func (s *NotificationDeliveryService) WithArchiveDir(dir string) *NotificationDeliveryService {
	s.archiveDir = strings.TrimSpace(dir)
	return s
}

// ArchiveDir reports the configured archive directory ("" = plain delete).
func (s *NotificationDeliveryService) ArchiveDir() string {
	return s.archiveDir
}

// CountByStatus returns one count per delivery status, including zeroes, so an
// operator can see the whole outbox at a glance.
func (s *NotificationDeliveryService) CountByStatus(ctx context.Context) (map[string]int64, error) {
	type row struct {
		Status string
		Total  int64
	}
	var rows []row
	if err := s.db.WithContext(ctx).
		Model(&models.NotificationDelivery{}).
		Select("status, count(*) AS total").
		Group("status").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	counts := map[string]int64{
		DeliveryStatusPending:    0,
		DeliveryStatusDelivering: 0,
		DeliveryStatusDelivered:  0,
		DeliveryStatusFailed:     0,
	}
	for _, r := range rows {
		counts[r.Status] = r.Total
	}
	return counts, nil
}

// Enqueue stores one delivery attempt for a channel. Called from the event
// subscriber, so it must stay cheap: no HTTP here.
func (s *NotificationDeliveryService) Enqueue(ctx context.Context, ch models.UserNotificationChannel, eventType, deviceID string, payload interface{}) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	row := &models.NotificationDelivery{
		ChannelID:     ch.ID,
		UserID:        ch.UserID,
		EventType:     eventType,
		DeviceID:      deviceID,
		Payload:       string(encoded),
		Status:        DeliveryStatusPending,
		NextAttemptAt: time.Now().UTC(),
	}
	return s.db.WithContext(ctx).Create(row).Error
}

// Run polls the outbox until ctx is cancelled.
func (s *NotificationDeliveryService) Run(ctx context.Context) {
	ticker := time.NewTicker(notificationWorkerInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.DrainOnce(ctx)
		}
	}
}

// PurgeTerminal deletes finished rows (delivered or failed) that have not been
// touched since cutoff. In-flight rows (pending/delivering) are never touched,
// so a long backoff is not mistaken for history.
//
// When an archive directory is configured the rows are appended to a daily
// JSONL file first and only deleted after that write lands on disk. A failed
// archive write therefore aborts the purge: retention must not be the step that
// silently destroys the only copy of a delivery history.
func (s *NotificationDeliveryService) PurgeTerminal(ctx context.Context, cutoff time.Time) (int64, error) {
	if s.archiveDir == "" {
		result := s.db.WithContext(ctx).
			Where("status IN ?", terminalDeliveryStatuses()).
			Where("updated_at < ?", cutoff.UTC()).
			Delete(&models.NotificationDelivery{})
		return result.RowsAffected, result.Error
	}

	var rows []models.NotificationDelivery
	if err := s.db.WithContext(ctx).
		Where("status IN ?", terminalDeliveryStatuses()).
		Where("updated_at < ?", cutoff.UTC()).
		Order("id").
		Limit(notificationPurgeBatch).
		Find(&rows).Error; err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	if err := s.archiveRows(rows); err != nil {
		return 0, fmt.Errorf("archive notification deliveries before purge: %w", err)
	}
	ids := make([]uint, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	// Re-check the status in the delete so a row that went back to pending
	// between the read and the delete is not removed by the archive path.
	result := s.db.WithContext(ctx).
		Where("id IN ?", ids).
		Where("status IN ?", terminalDeliveryStatuses()).
		Delete(&models.NotificationDelivery{})
	return result.RowsAffected, result.Error
}

// archiveRows appends one JSON object per row to the day's archive file and
// fsyncs it, so the bytes are durable before the caller deletes them.
func (s *NotificationDeliveryService) archiveRows(rows []models.NotificationDelivery) error {
	if err := os.MkdirAll(s.archiveDir, 0o755); err != nil {
		return err
	}
	name := fmt.Sprintf("notification-deliveries-%s.jsonl", time.Now().UTC().Format("2006-01-02"))
	path := filepath.Join(s.archiveDir, name)

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	for _, row := range rows {
		if err := encoder.Encode(row); err != nil {
			return err
		}
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(buf.Bytes()); err != nil {
		return err
	}
	return file.Sync()
}

func terminalDeliveryStatuses() []string {
	return []string{DeliveryStatusDelivered, DeliveryStatusFailed}
}

// RunRetention sweeps expired terminal rows on a ticker until ctx is
// cancelled. retentionDays <= 0 keeps history forever, like the recording
// retention setting, so the outbox can be audited indefinitely if needed.
func (s *NotificationDeliveryService) RunRetention(ctx context.Context, interval time.Duration, retentionDays int) {
	if retentionDays <= 0 || interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
			removed, err := s.PurgeTerminal(ctx, cutoff)
			if err != nil {
				log.Printf("[notification] retention sweep failed: %v", err)
				continue
			}
			if removed > 0 {
				log.Printf("[notification] retention purged %d terminal deliveries older than %s",
					removed, cutoff.Format(time.RFC3339))
			}
		}
	}
}

// DrainOnce claims and delivers every row that is due right now. Exported so
// tests and maintenance commands can drive the queue directly.
func (s *NotificationDeliveryService) DrainOnce(ctx context.Context) {
	claimed, err := s.claimDue(ctx, notificationClaimBatch)
	if err != nil {
		log.Printf("[notification] claim failed: %v", err)
		return
	}
	for _, row := range claimed {
		s.deliver(ctx, row)
	}
}

// claimChannelRow is the single-column projection phase 1 needs: which
// channels have due work. A named type keeps GORM's column mapping explicit.
type claimChannelRow struct {
	ChannelID uint
}

// claimDue atomically flips up to limit due rows to `delivering`. The
// FOR UPDATE SKIP LOCKED claim keeps two workers (or two server instances)
// from sending the same row twice.
//
// The claim is per-channel fair rather than strictly oldest-first. Ordering the
// whole table by id looks right until one endpoint accumulates a backlog: from
// then on every batch is filled by that channel's oldest rows and every other
// channel's notifications sit behind them, no matter how small those channels
// are. So phase 1 picks the channels that hold the oldest due work and phase 2
// takes at most notificationPerChannelBatch rows from each. The batch can
// therefore be shorter than limit when few channels have work; that is the
// deliberate cost of never letting one channel monopolise the worker.
func (s *NotificationDeliveryService) claimDue(ctx context.Context, limit int) ([]models.NotificationDelivery, error) {
	if limit <= 0 {
		return nil, nil
	}
	var claimed []models.NotificationDelivery
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		// Phase 1: which channels have due work, oldest work first. This cannot
		// carry FOR UPDATE - Postgres rejects FOR UPDATE together with the
		// aggregates/window functions a per-channel pick needs - so the locking
		// happens in phase 2, where SKIP LOCKED still guarantees each row goes
		// to exactly one claimer.
		var channelRows []claimChannelRow
		if err := tx.Raw(`SELECT channel_id FROM notification_deliveries
			WHERE status = ? AND next_attempt_at <= ?
			GROUP BY channel_id
			ORDER BY MIN(id)
			LIMIT ?`,
			DeliveryStatusPending, now, limit).Scan(&channelRows).Error; err != nil {
			return err
		}
		for _, ch := range channelRows {
			if len(claimed) >= limit {
				break
			}
			want := min(notificationPerChannelBatch, limit-len(claimed))
			var rows []models.NotificationDelivery
			if err := tx.Raw(`SELECT * FROM notification_deliveries
				WHERE status = ? AND next_attempt_at <= ? AND channel_id = ?
				ORDER BY id
				LIMIT ?
				FOR UPDATE SKIP LOCKED`,
				DeliveryStatusPending, now, ch.ChannelID, want).Scan(&rows).Error; err != nil {
				return err
			}
			claimed = append(claimed, rows...)
		}
		if len(claimed) == 0 {
			return nil
		}
		ids := make([]uint, 0, len(claimed))
		for _, row := range claimed {
			ids = append(ids, row.ID)
		}
		return tx.Model(&models.NotificationDelivery{}).
			Where("id IN ?", ids).
			Updates(map[string]interface{}{
				"status":     DeliveryStatusDelivering,
				"updated_at": time.Now().UTC(),
			}).Error
	})
	return claimed, err
}

// deliver performs one HTTP attempt for a claimed row and records the
// outcome: delivered, rescheduled with backoff, or parked as failed.
func (s *NotificationDeliveryService) deliver(ctx context.Context, row models.NotificationDelivery) {
	var channel models.UserNotificationChannel
	if err := s.db.WithContext(ctx).First(&channel, row.ChannelID).Error; err != nil {
		// The channel was deleted while the event was queued: nothing to send.
		s.finishDelivery(ctx, row.ID, 0, DeliveryStatusFailed, 0, "channel no longer exists")
		return
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(row.Payload), &payload); err != nil {
		s.finishDelivery(ctx, row.ID, channel.ID, DeliveryStatusFailed, 0, "payload is not valid JSON")
		return
	}

	status, err := postWebhookOnce(channel.URL, channel.Secret, payload)
	attempts := row.Attempts + 1
	now := time.Now().UTC()

	if err == nil && status >= 200 && status < 300 {
		s.finishDelivery(ctx, row.ID, channel.ID, DeliveryStatusDelivered, status, "")
		return
	}

	message := messageFor(status, err)
	if attempts >= notificationMaxAttempts {
		s.finishDelivery(ctx, row.ID, channel.ID, DeliveryStatusFailed, status, message)
		return
	}
	if uerr := s.db.WithContext(ctx).Model(&models.NotificationDelivery{}).
		Where("id = ?", row.ID).
		Updates(map[string]interface{}{
			"status":          DeliveryStatusPending,
			"attempts":        attempts,
			"last_status":     status,
			"last_error":      message,
			"next_attempt_at": now.Add(notificationBackoff(attempts)),
			"updated_at":      now,
		}).Error; uerr != nil {
		log.Printf("[notification] reschedule delivery %d failed: %v", row.ID, uerr)
	}
	s.touchChannel(ctx, channel.ID, status, now)
}

// finishDelivery writes a terminal state and records the outcome on the
// channel so the API can show whether the endpoint is healthy.
func (s *NotificationDeliveryService) finishDelivery(ctx context.Context, id, channelID uint, status string, httpStatus int, message string) {
	now := time.Now().UTC()
	updates := map[string]interface{}{
		"status":      status,
		"last_status": httpStatus,
		"last_error":  message,
		"updated_at":  now,
		"attempts":    gorm.Expr("attempts + 1"),
	}
	if status == DeliveryStatusDelivered {
		updates["delivered_at"] = now
	}
	if err := s.db.WithContext(ctx).Model(&models.NotificationDelivery{}).
		Where("id = ?", id).
		Updates(updates).Error; err != nil {
		log.Printf("[notification] finish delivery %d failed: %v", id, err)
	}
	if channelID != 0 {
		s.touchChannel(ctx, channelID, httpStatus, now)
	}
}

// touchChannel records the last outcome on the channel row.
func (s *NotificationDeliveryService) touchChannel(ctx context.Context, channelID uint, status int, at time.Time) {
	if err := s.db.WithContext(ctx).Model(&models.UserNotificationChannel{}).
		Where("id = ?", channelID).
		Updates(map[string]interface{}{
			"last_triggered": at,
			"last_status":    status,
		}).Error; err != nil {
		log.Printf("[notification] update channel %d status: %v", channelID, err)
	}
}

// ListByChannel returns the delivery log for one channel, newest first.
func (s *NotificationDeliveryService) ListByChannel(ctx context.Context, channelID uint, limit int) ([]models.NotificationDelivery, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var rows []models.NotificationDelivery
	err := s.db.WithContext(ctx).
		Where("channel_id = ?", channelID).
		Order("id DESC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

// Retry re-arms one parked (`failed`) delivery so the worker will try it again
// without waiting for a new event. The attempt budget is reset to zero on
// purpose: a transient outage should not permanently consume a row's tries,
// and the previous failure is still visible in `last_error` until the next
// attempt overwrites it. Rows that are delivered or already in flight are not
// retryable.
func (s *NotificationDeliveryService) Retry(ctx context.Context, channelID, deliveryID uint) (*models.NotificationDelivery, error) {
	var row models.NotificationDelivery
	if err := s.db.WithContext(ctx).
		Where("id = ? AND channel_id = ?", deliveryID, channelID).
		First(&row).Error; err != nil {
		return nil, err
	}
	if row.Status != DeliveryStatusFailed {
		return nil, ErrDeliveryNotRetryable
	}
	now := time.Now().UTC()
	if err := s.db.WithContext(ctx).Model(&models.NotificationDelivery{}).
		Where("id = ?", row.ID).
		Updates(map[string]interface{}{
			"status":          DeliveryStatusPending,
			"attempts":        0,
			"next_attempt_at": now,
			"updated_at":      now,
		}).Error; err != nil {
		return nil, err
	}
	row.Status = DeliveryStatusPending
	row.Attempts = 0
	row.NextAttemptAt = now
	row.UpdatedAt = now
	return &row, nil
}

// MaxRetryFailedLimit bounds RetryFailedFilters.Limit so one request cannot
// arm an unbounded batch.
const MaxRetryFailedLimit = 1000

// MaxRetryFailedChannelIDs bounds RetryFailedFilters.ChannelIDs. The list is a
// narrowing filter, not a bulk-ingest surface, so a handful of endpoints is
// the realistic size and anything much larger is a caller mistake.
const MaxRetryFailedChannelIDs = 50

// MaxRetryFailedSpreadMinutes bounds RetryFailedFilters.SpreadOverMinutes. A
// day is plenty: the point of a spread replay is to smooth a burst, not to
// schedule a backlog for later.
const MaxRetryFailedSpreadMinutes = 1440

// notificationReplaySpreadBuckets is how many due-times a spread replay uses.
// Re-arming every row at once hands the endpoint the whole backlog in one burst
// and can trip a rate limiter that would have coped with the same volume
// spread out; a handful of buckets is enough to avoid that without turning the
// replay into a drip.
const notificationReplaySpreadBuckets = 20

// RetryFailedFilters narrows a bulk replay. The zero value means "every parked
// delivery in scope", which is how the endpoints behaved before filters
// existed.
type RetryFailedFilters struct {
	// EventType limits the replay to one event type, e.g.
	// "device.access.denied".
	EventType string
	// DeviceID limits the replay to deliveries about one device. Rows written
	// before the device_id column existed have an empty value and are never
	// matched by this filter (they still replay without one).
	DeviceID string
	// Limit caps how many rows a single call re-arms, oldest first. 0 means no
	// cap; the handler clamps anything above MaxRetryFailedLimit.
	Limit int
	// SpreadOverMinutes paces the replay across that many minutes instead of
	// making every matched row due at once. 0 keeps the original behaviour
	// (everything due immediately); the handler clamps anything above
	// MaxRetryFailedSpreadMinutes.
	SpreadOverMinutes int
	// ChannelIDs limits the replay to specific channels - the cross-channel
	// endpoint's answer to "one of my five endpoints is broken, leave the
	// others alone". Empty means every channel in scope. On the per-channel
	// endpoint this can only narrow further, never widen.
	ChannelIDs []uint
	// CreatedAfter / CreatedBefore bound the replay by when the event was
	// queued. Both are inclusive. created_at is used rather than updated_at on
	// purpose: updated_at moves on every attempt, so a window over it would
	// mean something different each time it is evaluated.
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
}

// RetryFailedResult reports what a bulk replay matched and what it did.
type RetryFailedResult struct {
	// Matched counts parked rows that satisfy the filters, before Limit.
	Matched int64
	// Requeued counts rows actually re-armed. Always 0 on a dry run.
	Requeued int64
}

// retryFailedQuery builds the "which parked rows are in scope" predicate. It is
// called once per statement (count, then select) because reusing a GORM chain
// after Count can carry statement state.
func retryFailedQuery(ctx context.Context, db *gorm.DB, scopeColumn string, scopeValue uint, f RetryFailedFilters) *gorm.DB {
	q := db.WithContext(ctx).Model(&models.NotificationDelivery{}).
		Where(scopeColumn+" = ? AND status = ?", scopeValue, DeliveryStatusFailed)
	if et := strings.TrimSpace(f.EventType); et != "" {
		q = q.Where("event_type = ?", et)
	}
	if d := strings.TrimSpace(f.DeviceID); d != "" {
		q = q.Where("device_id = ?", d)
	}
	if len(f.ChannelIDs) > 0 {
		q = q.Where("channel_id IN ?", f.ChannelIDs)
	}
	if f.CreatedAfter != nil {
		q = q.Where("created_at >= ?", f.CreatedAfter.UTC())
	}
	if f.CreatedBefore != nil {
		q = q.Where("created_at <= ?", f.CreatedBefore.UTC())
	}
	return q
}

// RetryFailedForChannel re-arms parked (`failed`) deliveries for one channel.
// It is the bulk form of Retry, for an operator who fixed a flaky endpoint and
// wants the backlog retried without clicking each row.
//
// A dry run counts what would be re-armed and changes nothing, so an operator
// can look before they leap; that matters because re-arming immediately wakes
// the delivery worker, and the obvious fix ("replay everything") is often not
// what the situation calls for.
func (s *NotificationDeliveryService) RetryFailedForChannel(ctx context.Context, channelID uint, f RetryFailedFilters, dryRun bool) (RetryFailedResult, error) {
	return s.retryFailed(ctx, "channel_id", channelID, f, dryRun)
}

// RetryFailedForUser is the cross-channel form: every parked delivery for one
// user, whatever channel it came from. Handy after an outage that took several
// endpoints down at once, and still scoped strictly by user_id so one account
// can never replay another's backlog.
func (s *NotificationDeliveryService) RetryFailedForUser(ctx context.Context, userID uint, f RetryFailedFilters, dryRun bool) (RetryFailedResult, error) {
	if userID == 0 {
		// The whole point of the scope is that a zero value must never turn
		// into "every user's backlog".
		return RetryFailedResult{}, errors.New("user id is required")
	}
	return s.retryFailed(ctx, "user_id", userID, f, dryRun)
}

func (s *NotificationDeliveryService) retryFailed(ctx context.Context, scopeColumn string, scopeValue uint, f RetryFailedFilters, dryRun bool) (RetryFailedResult, error) {
	var out RetryFailedResult
	if err := retryFailedQuery(ctx, s.db, scopeColumn, scopeValue, f).Count(&out.Matched).Error; err != nil {
		return out, err
	}
	if dryRun || out.Matched == 0 {
		return out, nil
	}

	// Oldest first, so a capped replay drains the backlog deterministically
	// instead of re-arming an arbitrary slice each call.
	ids := retryFailedQuery(ctx, s.db, scopeColumn, scopeValue, f).Select("id").Order("id ASC")
	if f.Limit > 0 {
		ids = ids.Limit(f.Limit)
	}
	if f.SpreadOverMinutes > 0 {
		var idList []uint
		if err := ids.Pluck("id", &idList).Error; err != nil {
			return out, err
		}
		return s.rearmSpread(ctx, idList, f.SpreadOverMinutes, out)
	}
	now := time.Now().UTC()
	result := s.db.WithContext(ctx).Model(&models.NotificationDelivery{}).
		Where("id IN (?)", ids).
		Updates(map[string]interface{}{
			"status":          DeliveryStatusPending,
			"attempts":        0,
			"next_attempt_at": now,
			"updated_at":      now,
		})
	if result.Error != nil {
		return out, result.Error
	}
	out.Requeued = result.RowsAffected
	return out, nil
}

// rearmSpread re-arms rows in staggered buckets so a bulk replay reaches the
// endpoint gradually instead of all at once. The window is nailed to the
// caller's number: the first bucket becomes due after one bucket interval and
// the last at the end of the window, so `spread_over_minutes: N` really does
// take N minutes to work through even though every row is already `pending`.
//
// Each bucket is its own UPDATE. That is deliberate: the status is re-checked
// in every statement, so a row that something else re-armed (or a concurrent
// retry of the same range) is left alone rather than being counted as ours.
func (s *NotificationDeliveryService) rearmSpread(ctx context.Context, idList []uint, spreadMinutes int, out RetryFailedResult) (RetryFailedResult, error) {
	if len(idList) == 0 {
		return out, nil
	}
	buckets := notificationReplaySpreadBuckets
	if len(idList) < buckets {
		buckets = len(idList)
	}
	perBucket := (len(idList) + buckets - 1) / buckets
	step := time.Duration(spreadMinutes) * time.Minute / time.Duration(buckets)
	now := time.Now().UTC()
	var requeued int64
	for start := 0; start < len(idList); start += perBucket {
		end := min(start+perBucket, len(idList))
		bucket := start / perBucket
		due := now.Add(time.Duration(bucket+1) * step)
		result := s.db.WithContext(ctx).Model(&models.NotificationDelivery{}).
			Where("id IN ? AND status = ?", idList[start:end], DeliveryStatusFailed).
			Updates(map[string]interface{}{
				"status":          DeliveryStatusPending,
				"attempts":        0,
				"next_attempt_at": due,
				"updated_at":      now,
			})
		if result.Error != nil {
			return out, result.Error
		}
		requeued += result.RowsAffected
	}
	out.Requeued = requeued
	return out, nil
}

// RetryAllFailed is RetryFailedForChannel with no filters, kept for callers
// written before filters existed.
func (s *NotificationDeliveryService) RetryAllFailed(ctx context.Context, channelID uint) (int64, error) {
	res, err := s.RetryFailedForChannel(ctx, channelID, RetryFailedFilters{}, false)
	return res.Requeued, err
}

// RetryAllFailedForUser is RetryFailedForUser with no filters.
func (s *NotificationDeliveryService) RetryAllFailedForUser(ctx context.Context, userID uint) (int64, error) {
	res, err := s.RetryFailedForUser(ctx, userID, RetryFailedFilters{}, false)
	return res.Requeued, err
}

// notificationBackoff grows 2s, 4s, 8s … up to the cap.
func notificationBackoff(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	backoff := time.Duration(1<<uint(min(attempts, 10))) * time.Second
	if backoff > notificationBackoffCap {
		return notificationBackoffCap
	}
	return backoff
}

func messageFor(status int, err error) string {
	if err != nil {
		return err.Error()
	}
	if status == 0 {
		return "transport error"
	}
	return "HTTP " + strconv.Itoa(status)
}
