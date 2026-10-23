package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"quickdesk/signaling/internal/config"
	"quickdesk/signaling/internal/database"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// QD-004: the Host needs a credential that is not the user's Bearer token and
// that cannot be replayed. Without Redis the route must refuse rather than
// hand out something that only looks one-shot.
func TestUploadTicketRouteRefusesWithoutRedis(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, _, _ := newUploadTicketStack(t, "")

	handler := NewSessionRecordingHandler(
		service.NewSessionRecordingService(db),
			nil,
		SessionRecordingStore{Dir: t.TempDir(), URLSecret: "test-secret"},
		service.NewAuditService(db),
	).WithUploadTickets(service.NewRecordingUploadTicketStore(nil, "test-secret"))

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("auth.user_id", uint(7))
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/me/recordings/1/upload-ticket", nil)

	handler.CreateUploadTicket(c)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", recorder.Code, recorder.Body.String())
	}
}

// With Redis the route issues a ticket bound to the recording's own session,
// so a Host cannot be handed a ticket for one recording and push another
// session's media under it.
func TestUploadTicketRouteBindsTheRecordingsSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, _, store := newUploadTicketStack(t, "sess-alpha")

	const userID = uint(9102)
	if err := db.Create(&models.User{
		ID:       userID,
		Username: "upload-ticket-session",
		Password: "test-hash",
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&models.SessionRecording{
		ID:       2,
		UserID:   userID,
		DeviceID: "900000002",
		SessionID: "sess-alpha",
		Status:   "recording",
	}).Error; err != nil {
		t.Fatalf("seed recording: %v", err)
	}

	handler := NewSessionRecordingHandler(
		service.NewSessionRecordingService(db),
		nil,
		SessionRecordingStore{Dir: t.TempDir(), URLSecret: "test-secret"},
		service.NewAuditService(db),
	).WithUploadTickets(store)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Params = gin.Params{{Key: "id", Value: "2"}}
	c.Set("auth.user_id", userID)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/me/recordings/2/upload-ticket", nil)

	handler.CreateUploadTicket(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		UploadURL string `json:"upload_url"`
		SingleUse bool   `json:"single_use"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.UploadURL == "" || !payload.SingleUse {
		t.Fatalf("payload = %+v, want a single-use upload url", payload)
	}

	// The issued ticket must carry the session, and redeeming it against a
	// different session must be refused.
	query := payload.UploadURL[indexOf(payload.UploadURL, "?t=")+3:]
	redeemed, err := store.Consume(context.Background(), query)
	if err != nil {
		t.Fatalf("Consume(): %v", err)
	}
	if redeemed.SessionID != "sess-alpha" {
		t.Fatalf("ticket session = %q, want sess-alpha", redeemed.SessionID)
	}
	if redeemed.MatchesSession("sess-beta") {
		t.Fatal("the issued ticket must not authorise another session")
	}
}

// The ticket must actually open the upload route: a Host holding one has no
// Bearer token, so a credential that cannot be presented there is useless.
func TestUploadTicketOpensTheArtifactUploadRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, _, store := newUploadTicketStack(t, "sess-alpha")

	const userID = uint(9103)
	if err := db.Create(&models.User{
		ID:       userID,
		Username: "upload-ticket-route",
		Password: "test-hash",
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&models.SessionRecording{
		ID:        3,
		UserID:    userID,
		DeviceID:  "900000003",
		SessionID: "sess-alpha",
		Status:    "recording",
	}).Error; err != nil {
		t.Fatalf("seed recording: %v", err)
	}

	handler := NewSessionRecordingHandler(
		service.NewSessionRecordingService(db),
		nil,
		SessionRecordingStore{Dir: t.TempDir(), URLSecret: "test-secret"},
		service.NewAuditService(db),
	).WithUploadTickets(store)

	ticket, _, err := store.IssueForSession(3, userID, "sess-alpha", 5*60*1_000_000_000)
	if err != nil {
		t.Fatalf("IssueForSession(): %v", err)
	}

	router := gin.New()
	router.PUT("/v1/recordings/:id/artifact", handler.UploadArtifactByTicket)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPut,
		"/v1/recordings/3/artifact?t=" + ticket + "&ext=.h264",
		bytes.NewReader([]byte("host-recorded-bytes")),
	)
	request.Header.Set("Content-Type", "application/octet-stream")
	request.Header.Set("Upload-Offset", "0")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK && recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want the upload to be accepted; body = %s",
			recorder.Code, recorder.Body.String())
	}
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

func newUploadTicketStack(t *testing.T, sessionID string) (*gorm.DB, *redis.Client, *service.RecordingUploadTicketStore) {
	t.Helper()
	db, err := database.InitSQLite(&config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   filepath.Join(t.TempDir(), "upload-ticket.db"),
		},
	})
	if err != nil {
		t.Fatalf("InitSQLite(): %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(): %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&models.User{}, &models.SessionRecording{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return db, client, service.NewRecordingUploadTicketStore(client, "test-secret")
}
// The cross-session guard is the point of binding the session into the ticket:
// a ticket minted for one session must be refused for a recording that belongs
// to another, instead of writing the bytes into the wrong row.
func TestUploadTicketRefusesADifferentSessionsRecording(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, _, store := newUploadTicketStack(t, "sess-alpha")

	const userID = uint(9104)
	if err := db.Create(&models.User{
		ID:       userID,
		Username: "upload-ticket-cross-session",
		Password: "test-hash",
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	// The ticket is minted for the alpha recording.
	if err := db.Create(&models.SessionRecording{
		ID:        4,
		UserID:    userID,
		DeviceID:  "900000004",
		SessionID: "sess-alpha",
		Status:    "recording",
	}).Error; err != nil {
		t.Fatalf("seed alpha recording: %v", err)
	}
	// The beta recording is the attacker's target: same owner, different session.
	if err := db.Create(&models.SessionRecording{
		ID:        5,
		UserID:    userID,
		DeviceID:  "900000005",
		SessionID: "sess-beta",
		Status:    "recording",
	}).Error; err != nil {
		t.Fatalf("seed beta recording: %v", err)
	}

	handler := NewSessionRecordingHandler(
		service.NewSessionRecordingService(db),
		nil,
		SessionRecordingStore{Dir: t.TempDir(), URLSecret: "test-secret"},
		service.NewAuditService(db),
	).WithUploadTickets(store)

	ticket, _, err := store.IssueForSession(4, userID, "sess-alpha", 5*60*1_000_000_000)
	if err != nil {
		t.Fatalf("IssueForSession(): %v", err)
	}

	router := gin.New()
	router.PUT("/v1/recordings/:id/artifact", handler.UploadArtifactByTicket)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPut,
		"/v1/recordings/5/artifact?t="+ticket+"&ext=.h264",
		bytes.NewReader([]byte("wrong-session-bytes")),
	)
	request.Header.Set("Content-Type", "application/octet-stream")
	request.Header.Set("Upload-Offset", "0")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body = %s", recorder.Code, recorder.Body.String())
	}

	// The beta recording must still have no artifact: the refusal happens
	// before any byte is stored.
	var beta models.SessionRecording
	if err := db.First(&beta, 5).Error; err != nil {
		t.Fatalf("load beta recording: %v", err)
	}
	if beta.ArtifactPath != "" {
		t.Fatalf("beta recording gained an artifact path %q", beta.ArtifactPath)
	}
}
// A Host that declares a digest and sends different bytes must be refused, and
// the recording must stay unattached. The digest check is the server's only
// chance to notice a corrupt transfer, because the Host is the only party that
// ever saw the media.
func TestTicketUploadRefusesADigestMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, _, store := newUploadTicketStack(t, "sess-alpha")

	const userID = uint(9203)
	if err := db.Create(&models.User{
		ID:       userID,
		Username: "upload-ticket-digest",
		Password: "test-hash",
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&models.SessionRecording{
		ID:        6,
		UserID:    userID,
		DeviceID:  "900000006",
		SessionID: "sess-alpha",
		Status:    "recording",
	}).Error; err != nil {
		t.Fatalf("seed recording: %v", err)
	}

	handler := NewSessionRecordingHandler(
		service.NewSessionRecordingService(db),
		nil,
		SessionRecordingStore{Dir: t.TempDir(), URLSecret: "test-secret"},
		service.NewAuditService(db),
	).WithUploadTickets(store)

	ticket, _, err := store.IssueForSession(6, userID, "sess-alpha", 5*60*1_000_000_000)
	if err != nil {
		t.Fatalf("IssueForSession(): %v", err)
	}

	router := gin.New()
	router.PUT("/v1/recordings/:id/artifact", handler.UploadArtifactByTicket)

	declared := sha256.Sum256([]byte("declared-bytes"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPut,
		"/v1/recordings/6/artifact?t="+ticket+"&ext=.h264",
		bytes.NewReader([]byte("actual-bytes")),
	)
	request.Header.Set("Content-Type", "application/octet-stream")
	request.Header.Set("Upload-Offset", "0")
	request.Header.Set("X-Artifact-Sha256", hex.EncodeToString(declared[:]))
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", recorder.Code, recorder.Body.String())
	}

	var row models.SessionRecording
	if err := db.First(&row, 6).Error; err != nil {
		t.Fatalf("load recording: %v", err)
	}
	if row.ArtifactPath != "" {
		t.Fatalf("recording gained artifact_path %q despite the digest mismatch", row.ArtifactPath)
	}
}
