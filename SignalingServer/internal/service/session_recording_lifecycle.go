package service

import (
	"context"
	"errors"
	"os"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// RecordingArtifactUsage is the storage summary surfaced to admins.
type RecordingArtifactUsage struct {
	Count      int64
	TotalBytes int64
}

// SweepStaleRecordings flips recordings that have been stuck in "recording"
// since before the cutoff to "failed".
//
// Without this a client that crashes mid-recording leaves a row that claims a
// capture is still in progress forever, and the UI offers neither a result nor
// a retry.
func (s *SessionRecordingService) SweepStaleRecordings(ctx context.Context, cutoff time.Time) (int64, error) {
	result := s.db.WithContext(ctx).
		Model(&models.SessionRecording{}).
		Where("status = ? AND updated_at < ?", "recording", cutoff.UTC()).
		Updates(map[string]interface{}{
			"status":     "failed",
			"updated_at": time.Now().UTC(),
		})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// ArtifactExpired decides whether a row's media file is past its retention.
//
// A row is only eligible when it still points at a file; the index row itself
// is deliberately kept so history survives the cleanup.
func ArtifactExpired(row models.SessionRecording, cutoff time.Time) bool {
	if row.ArtifactPath == "" && row.ArtifactURL == "" {
		return false
	}
	stamp := row.UpdatedAt
	if row.EndedAt != nil {
		stamp = *row.EndedAt
	}
	return stamp.Before(cutoff.UTC())
}

// ExpiredArtifacts returns the rows whose media files are past retention.
func (s *SessionRecordingService) ExpiredArtifacts(
	ctx context.Context,
	cutoff time.Time,
	limit int,
) ([]models.SessionRecording, error) {
	q := s.db.WithContext(ctx).
		Where("artifact_path <> '' OR artifact_url <> ''").
		Where("COALESCE(ended_at, updated_at) < ?", cutoff.UTC()).
		Order("id ASC")
	if limit <= 0 {
		limit = 200
	}
	var rows []models.SessionRecording
	if err := q.Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ClearArtifact forgets the stored media for a row while keeping the row.
func (s *SessionRecordingService) ClearArtifact(ctx context.Context, id uint) error {
	result := s.db.WithContext(ctx).
		Model(&models.SessionRecording{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"artifact_path": "",
			"artifact_url":  "",
			"size_bytes":    0,
			"updated_at":    time.Now().UTC(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ArtifactUsage reports how many recordings still hold media and how much disk
// they occupy according to the index.
func (s *SessionRecordingService) ArtifactUsage(ctx context.Context) (RecordingArtifactUsage, error) {
	var usage RecordingArtifactUsage
	row := struct {
		Count      int64
		TotalBytes int64
	}{}
	err := s.db.WithContext(ctx).
		Model(&models.SessionRecording{}).
		Select("COUNT(*) AS count, COALESCE(SUM(size_bytes), 0) AS total_bytes").
		Where("artifact_path <> '' OR artifact_url <> ''").
		Scan(&row).Error
	if err != nil {
		return usage, err
	}
	usage.Count = row.Count
	usage.TotalBytes = row.TotalBytes
	return usage, nil
}

// RecordingLifecycle periodically reclaims storage and unsticks recordings.
type RecordingLifecycle struct {
	recordings *SessionRecordingService
	staleAfter time.Duration
	retention  time.Duration
	interval   time.Duration
	removeFile func(string) error
	purgeBatch int
}

func NewRecordingLifecycle(
	recordings *SessionRecordingService,
	staleAfter time.Duration,
	retention time.Duration,
	interval time.Duration,
) *RecordingLifecycle {
	return &RecordingLifecycle{
		recordings: recordings,
		staleAfter: staleAfter,
		retention:  retention,
		interval:   interval,
		removeFile: os.Remove,
		purgeBatch: 200,
	}
}

// Run blocks until ctx is cancelled, sweeping on every tick. Retention of zero
// means "keep media forever", so only the stale sweep runs in that case.
func (l *RecordingLifecycle) Run(ctx context.Context) {
	if l == nil || l.recordings == nil || l.interval <= 0 {
		return
	}
	ticker := time.NewTicker(l.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _, _, _ = l.SweepOnce(ctx)
		}
	}
}

// SweepOnce performs one stale sweep plus one retention pass and reports what
// changed. It is also the implementation behind the admin trigger.
func (l *RecordingLifecycle) SweepOnce(ctx context.Context) (stale int64, purged int, freedBytes int64, err error) {
	if l == nil || l.recordings == nil {
		return 0, 0, 0, errors.New("recording lifecycle is not configured")
	}
	now := time.Now().UTC()
	if l.staleAfter > 0 {
		stale, err = l.recordings.SweepStaleRecordings(ctx, now.Add(-l.staleAfter))
		if err != nil {
			return stale, 0, 0, err
		}
	}
	if l.retention <= 0 {
		return stale, 0, 0, nil
	}
	cutoff := now.Add(-l.retention)
	rows, err := l.recordings.ExpiredArtifacts(ctx, cutoff, l.purgeBatch)
	if err != nil {
		return stale, 0, 0, err
	}
	for i := range rows {
		row := rows[i]
		if row.ArtifactPath != "" {
			if removeErr := l.removeFile(row.ArtifactPath); removeErr != nil && !os.IsNotExist(removeErr) {
				// Keep the row pointing at the file so a later pass can retry.
				continue
			}
		}
		if err = l.recordings.ClearArtifact(ctx, row.ID); err != nil {
			return stale, purged, freedBytes, err
		}
		purged++
		freedBytes += row.SizeBytes
	}
	return stale, purged, freedBytes, nil
}
