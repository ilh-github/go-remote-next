package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"quickdesk/signaling/internal/config"
	"quickdesk/signaling/internal/database"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// QD-110's archive half, owner side.
//
// Retention appends expired rows to a per-day JSONL file before deleting them, so
// the live table stays bounded without losing the trail. The admin console can
// search those files; the owner cannot. That is the same asymmetry the export
// had: the rows are theirs, they were written by their own controller, and after
// retention runs they are only reachable by an admin.
//
// `QueryArchive` already filters on ActorUserID, so this is wiring plus a scope
// that cannot be widened by the caller. The tests below are mostly about that
// scope: an archive read that let a caller pick somebody else's actor id would be
// worse than no read at all.

// newOwnerArchiveTestDB builds the SQLite store the audit service needs, and an
// archive directory holding one day of rows.
func newOwnerArchiveTestDB(t *testing.T, rows []models.AuditLog) (*gorm.DB, string) {
	t.Helper()
	db, err := database.InitSQLite(&config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   filepath.Join(t.TempDir(), "owner-archive.db"),
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
	if err := db.AutoMigrate(&models.User{}, &models.Device{}, &models.AuditLog{}); err != nil {
		t.Fatalf("AutoMigrate(): %v", err)
	}

	dir := filepath.Join(t.TempDir(), "archive")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	name := "audit-logs-" + time.Now().UTC().Format("2006-01-02") + ".jsonl"
	file, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	for _, row := range rows {
		if err := enc.Encode(row); err != nil {
			t.Fatalf("write archive: %v", err)
		}
	}
	return db, dir
}

func TestOwnerAuditArchiveReturnsOnlyTheCallersRows(t *testing.T) {
	const (
		userID = uint(9801)
		other  = uint(9802)
	)
	rows := []models.AuditLog{
		{Action: "file.transfer", ResourceID: "tr-mine", ActorUserID: userID},
		{Action: "file.transfer", ResourceID: "tr-theirs", ActorUserID: other},
	}
	db, dir := newOwnerArchiveTestDB(t, rows)
	audit := service.NewAuditService(db).WithArchiveDir(dir)
	h := NewOwnerAuditHandler(audit)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("auth.user_id", userID)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/me/audit-logs/archive", nil)
	h.Archive(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var response struct {
		Items []struct {
			ResourceID string `json:"resource_id"`
		} `json:"items"`
		Truncated bool `json:"truncated"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("items = %d, want only the caller's own 1", len(response.Items))
	}
	if response.Items[0].ResourceID != "tr-mine" {
		t.Fatalf("resource_id = %q, want tr-mine", response.Items[0].ResourceID)
	}
}

// The caller must not be able to widen the scope. `actor_user_id` is how the
// admin query narrows to one actor; accepted here it would be a way to read
// anybody's archived rows by typing their id.
func TestOwnerAuditArchiveIgnoresACallerSuppliedActor(t *testing.T) {
	const (
		userID = uint(9803)
		other  = uint(9804)
	)
	rows := []models.AuditLog{
		{Action: "file.transfer", ResourceID: "tr-mine", ActorUserID: userID},
		{Action: "file.transfer", ResourceID: "tr-theirs", ActorUserID: other},
	}
	db, dir := newOwnerArchiveTestDB(t, rows)
	audit := service.NewAuditService(db).WithArchiveDir(dir)
	h := NewOwnerAuditHandler(audit)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("auth.user_id", userID)
	c.Request = httptest.NewRequest(http.MethodGet,
		"/v1/me/audit-logs/archive?actor_user_id="+itoa(int(other)), nil)
	h.Archive(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "tr-theirs") {
		t.Fatal("a caller-supplied actor_user_id widened the read to another account's rows")
	}
}

// An action outside the owner's trail is refused rather than treated as "no
// filter", for the same reason the list and the export refuse one.
func TestOwnerAuditArchiveRefusesAnUnknownAction(t *testing.T) {
	db, dir := newOwnerArchiveTestDB(t, nil)
	audit := service.NewAuditService(db).WithArchiveDir(dir)
	h := NewOwnerAuditHandler(audit)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("auth.user_id", uint(9805))
	c.Request = httptest.NewRequest(http.MethodGet,
		"/v1/me/audit-logs/archive?action=bogus.action", nil)
	h.Archive(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}
}

// A direction-scoped filter is refused rather than widened. The archive query has
// no details filter, so honouring it would mean answering "here are both
// directions" to "what did I upload" - rows the caller did not ask for, which is
// exactly what the live read's direction filter prevents.
func TestOwnerAuditArchiveRefusesADirectionScopedAction(t *testing.T) {
	const userID = uint(9808)
	db, dir := newOwnerArchiveTestDB(t, []models.AuditLog{
		{Action: "file.transfer", ResourceID: "tr-up", ActorUserID: userID,
			Details: `{"directions":["upload"]}`},
		{Action: "file.transfer", ResourceID: "tr-down", ActorUserID: userID,
			Details: `{"directions":["download"]}`},
	})
	h := NewOwnerAuditHandler(service.NewAuditService(db).WithArchiveDir(dir))

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("auth.user_id", userID)
	c.Request = httptest.NewRequest(http.MethodGet,
		"/v1/me/audit-logs/archive?action=file.transfer.upload", nil)
	h.Archive(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}
}

// A day with no archive file is an empty answer, not an error: nothing expired
// that day is the normal case for a young deployment.
func TestOwnerAuditArchiveWithNoFileForThatDayIsEmpty(t *testing.T) {
	db, dir := newOwnerArchiveTestDB(t, nil)
	audit := service.NewAuditService(db).WithArchiveDir(dir)
	h := NewOwnerAuditHandler(audit)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("auth.user_id", uint(9806))
	c.Request = httptest.NewRequest(http.MethodGet,
		"/v1/me/audit-logs/archive?date=2020-01-01", nil)
	h.Archive(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var response struct {
		Items []json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(response.Items) != 0 {
		t.Fatalf("items = %d, want none for a day with no archive", len(response.Items))
	}
}

// An unset archive directory is a configuration fact the caller can act on, not
// a server error.
func TestOwnerAuditArchiveWithoutADirectorySaysSo(t *testing.T) {
	db := newOwnerAuditTestDB(t)
	h := NewOwnerAuditHandler(service.NewAuditService(db))

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("auth.user_id", uint(9807))
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/me/audit-logs/archive", nil)
	h.Archive(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}
}
