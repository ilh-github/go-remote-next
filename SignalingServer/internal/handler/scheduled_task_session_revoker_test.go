package handler

import (
	"context"
	"testing"

	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"
)

func TestScheduledTaskRemoteSessionRevokerUsesTheDeviceBatchPath(t *testing.T) {
	db, deviceID := newDeviceRevokeTestDB(t, 42)
	sessions := service.NewRemoteSessionService(db)
	ctx := context.Background()
	for _, input := range []service.OpenRemoteSessionInput{
		{SessionID: "scheduled_batch_local", DeviceID: deviceID, InstanceID: "inst-here"},
		{SessionID: "scheduled_batch_far", DeviceID: deviceID, InstanceID: "inst-far"},
	} {
		if _, err := sessions.Open(ctx, input); err != nil {
			t.Fatalf("open %s: %v", input.SessionID, err)
		}
	}
	revoker := &fakeRevoker{
		instanceID: "inst-here",
		local:      map[string]bool{"scheduled_batch_local": true},
		onClose: func(sessionID string) {
			if _, _, err := sessions.Close(
				ctx, sessionID, models.RemoteSessionEndRevoked,
			); err != nil {
				t.Errorf("close %s: %v", sessionID, err)
			}
		},
	}
	adapter := NewScheduledTaskRemoteSessionRevoker(sessions, revoker)
	revoked, failed, truncated, err := adapter.RevokeDeviceSessions(
		ctx, 990282, deviceID)
	if err != nil {
		t.Fatalf("adapter revoke: %v", err)
	}
	if revoked != 1 || failed != 1 || truncated {
		t.Fatalf("revoked=%d failed=%d truncated=%t, want 1,1,false",
			revoked, failed, truncated)
	}
}
