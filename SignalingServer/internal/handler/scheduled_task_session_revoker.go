package handler

import (
	"context"

	"quickdesk/signaling/internal/service"
)

// ScheduledTaskRemoteSessionRevoker adapts the existing cross-instance revoke
// path to the scheduler's control-plane action. The action has already selected
// devices whose live session rows belong to the task owner, so this adapter only
// performs the same per-device close the owner-facing endpoint performs.
type ScheduledTaskRemoteSessionRevoker struct {
	sessions *service.RemoteSessionService
	revoker  SessionRevoker
}

func NewScheduledTaskRemoteSessionRevoker(
	sessions *service.RemoteSessionService,
	revoker SessionRevoker,
) *ScheduledTaskRemoteSessionRevoker {
	return &ScheduledTaskRemoteSessionRevoker{
		sessions: sessions,
		revoker:  revoker,
	}
}

func (r *ScheduledTaskRemoteSessionRevoker) RevokeDeviceSessions(
	ctx context.Context,
	_ uint,
	deviceID string,
) (int, int, bool, error) {
	revoked, failed, truncated, _, err := revokeDeviceSessions(
		ctx, r.sessions, r.revoker, deviceID, maxBatchRevokeSessions)
	return revoked, failed, truncated, err
}
