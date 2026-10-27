package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/repository"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// QD-111 promises idempotency "by transfer ID". The key `LogActorOnce` actually
// uses is (actor, action, resource_type, resource_id), and the action carries the
// direction - `file.transfer.upload` and `file.transfer.download` are different
// actions. So a transfer id that appears with both directions produces two rows,
// and the promise holds for a retry but not for the id itself.
//
// That is not a hypothetical shape: the controller reports upload and download
// separately, and a client that reuses one id across both (a sync that pushes
// then pulls the same file, or a retry after a direction change) gets two audit
// rows for what the operator will read as one transfer.

func newDirectionAuditFixture(t *testing.T, userID uint, deviceID string) (*gin.Engine, *gorm.DB) {
	t.Helper()
	db := newFileTransferAuditTestDB(t)
	if err := db.Create(&models.User{
		ID:       userID,
		Username: "file-transfer-audit-direction",
		Password: "test-hash",
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	owner := userID
	if err := db.Create(&models.Device{
		DeviceID:   deviceID,
		DeviceUUID: "uuid-file-transfer-audit-direction",
		UserID:     &owner,
	}).Error; err != nil {
		t.Fatalf("seed device: %v", err)
	}

	h := NewFileTransferAuditHandler(
		service.NewDeviceService(repository.NewDeviceRepository(db), service.NewDeviceSecretService()),
		service.NewAuditService(db),
	)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("auth.user_id", userID)
		c.Next()
	})
	router.POST("/v1/me/file-transfer-audits", h.Create)
	return router, db
}

func postDirectionAudit(t *testing.T, router *gin.Engine, deviceID, transferID, direction string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"device_id":         deviceID,
		"transfer_id":       transferID,
		"direction":         direction,
		"status":            "completed",
		"filename":          "archive.zip",
		"bytes_transferred": 100,
		"total_bytes":       100,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/me/file-transfer-audits", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// The same transfer id with both directions must be one audit row, not two. The
// direction is still recorded - it belongs in the details, where it describes the
// transfer, rather than in the identity key, where it splits one transfer into
// two.
func TestFileTransferAuditTreatsOneTransferIDAsOneTransferAcrossDirections(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const (
		userID     = uint(4103)
		deviceID   = "884104103"
		transferID = "transfer-both-directions"
	)
	router, db := newDirectionAuditFixture(t, userID, deviceID)

	if rec := postDirectionAudit(t, router, deviceID, transferID, "upload"); rec.Code != http.StatusNoContent {
		t.Fatalf("upload status = %d, want 204: %s", rec.Code, rec.Body.String())
	}
	rec := postDirectionAudit(t, router, deviceID, transferID, "download")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("download status = %d, want 204: %s", rec.Code, rec.Body.String())
	}

	var rows []models.AuditLog
	if err := db.Model(&models.AuditLog{}).Find(&rows).Error; err != nil {
		t.Fatalf("read audit rows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("audit rows = %d, want one row per transfer id regardless of direction", len(rows))
	}
}
