package handler

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/repository"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
)

// QD-110 promises export alongside audit, archive and retention. The admin side
// has it -- `/admin/audit-logs:export` serves CSV and JSON -- but the owner's own
// read of the same trail has no way out. An owner who wants to hand their
// transfer history to somebody else, or keep a copy outside the server, has to
// ask an admin for it. The rows are theirs and the read is already scoped to
// them, so the export belongs on the same surface.
//
// The scope is the whole point of these tests: an export that leaked a
// neighbour's row would be a worse bug than having no export, so the boundaries
// are asserted rather than assumed.

func TestOwnerAuditExportRefusesAnUnknownFormat(t *testing.T) {
	db := newOwnerAuditTestDB(t)
	f, _ := newOwnerAuditHandler(t, db, "700000301", 990701)

	rec := httptest.NewRecorder()
	ownerAuditRouter(t, f.Handler, 990701).ServeHTTP(
		rec,
		httptest.NewRequest(http.MethodGet, "/v1/me/audit-logs:export?format=xml", nil),
	)

	// Same reason the admin export refuses one: answering with a file the caller
	// did not ask for is worse than refusing.
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}
}

func TestOwnerAuditExportCsvCarriesTheHeaderAndTheOwnersRows(t *testing.T) {
	db := newOwnerAuditTestDB(t)
	const (
		deviceID = "700000302"
		userID   = uint(990702)
	)
	f, c := newOwnerAuditHandler(t, db, deviceID, userID)
	recordFileTransferAudit(t, f.Files, c, fileTransferAuditBody{
		DeviceID:         deviceID,
		TransferID:       "tr-export-1",
		Direction:        "upload",
		Status:           "completed",
		Filename:         "report.pdf",
		BytesTransferred: 2048,
		TotalBytes:       2048,
	})

	rec := httptest.NewRecorder()
	ownerAuditRouter(t, f.Handler, userID).ServeHTTP(
		rec,
		httptest.NewRequest(http.MethodGet, "/v1/me/audit-logs:export?format=csv", nil),
	)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/csv") {
		t.Fatalf("Content-Type = %q, want text/csv", got)
	}
	if got := rec.Header().Get("Content-Disposition"); !strings.Contains(got, "attachment") {
		t.Fatalf("Content-Disposition = %q, want an attachment", got)
	}

	rows, err := csv.NewReader(strings.NewReader(rec.Body.String())).ReadAll()
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}
	if len(rows) < 2 {
		t.Fatalf("csv has %d rows, want a header plus at least one entry", len(rows))
	}
	header := strings.Join(rows[0], ",")
	for _, want := range []string{"action", "resource_id", "details", "created_at"} {
		if !strings.Contains(header, want) {
			t.Fatalf("csv header %q is missing %q", header, want)
		}
	}
	joined := strings.Join(rows[1], ",")
	if !strings.Contains(joined, "tr-export-1") {
		t.Fatalf("the exported row does not name the transfer: %s", joined)
	}
}

func TestOwnerAuditExportJsonIsTheRows(t *testing.T) {
	db := newOwnerAuditTestDB(t)
	const (
		deviceID = "700000303"
		userID   = uint(990703)
	)
	f, c := newOwnerAuditHandler(t, db, deviceID, userID)
	recordFileTransferAudit(t, f.Files, c, fileTransferAuditBody{
		DeviceID:   deviceID,
		TransferID: "tr-export-2",
		Direction:  "download",
		Status:     "completed",
	})

	rec := httptest.NewRecorder()
	ownerAuditRouter(t, f.Handler, userID).ServeHTTP(
		rec,
		httptest.NewRequest(http.MethodGet, "/v1/me/audit-logs:export?format=json", nil),
	)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var rows []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("json export has %d rows, want the 1 the owner reported", len(rows))
	}
	if rows[0]["resource_id"] != "tr-export-2" {
		t.Fatalf("resource_id = %v, want tr-export-2", rows[0]["resource_id"])
	}
}

// The export must be scoped exactly like the read it mirrors. A row another
// account reported for their own device must not appear in this owner's file,
// even though the export has no page limit to hide behind.
func TestOwnerAuditExportExcludesAnotherOwnersRows(t *testing.T) {
	db := newOwnerAuditTestDB(t)
	const (
		mine   = "700000304"
		theirs = "700000305"
		userID = uint(990704)
		other  = uint(990705)
	)
	f, c := newOwnerAuditHandler(t, db, mine, userID)
	// The second account needs its own user and device rows before it can report
	// anything; `newOwnerAuditHandler` seeds both, exactly as it did for the first.

	recordFileTransferAudit(t, f.Files, c, fileTransferAuditBody{
		DeviceID: mine, TransferID: "tr-mine", Direction: "upload", Status: "completed",
	})
	// The second account needs its own user and device rows before a row can name
	// its device. Seeded here rather than through `newOwnerAuditHandler`, which
	// reuses one fixed username and so cannot build a second account on the same
	// database.
	if err := db.Create(&models.User{
		ID: other, Username: "owner-audit-other", Password: "test-hash",
	}).Error; err != nil {
		t.Fatalf("seed other user: %v", err)
	}
	otherOwner := other
	if err := db.Create(&models.Device{
		DeviceID: theirs, DeviceUUID: "uuid-owner-audit-other", UserID: &otherOwner,
	}).Error; err != nil {
		t.Fatalf("seed other device: %v", err)
	}
	devices := service.NewDeviceService(repository.NewDeviceRepository(db), service.NewDeviceSecretService())
	otherFiles := NewFileTransferAuditHandler(devices, service.NewAuditService(db))
	otherRec := httptest.NewRecorder()
	otherCtx, _ := gin.CreateTestContext(otherRec)
	otherCtx.Set("auth.user_id", other)
	recordFileTransferAudit(t, otherFiles, otherCtx, fileTransferAuditBody{
		DeviceID: theirs, TransferID: "tr-theirs", Direction: "upload", Status: "completed",
	})

	rec := httptest.NewRecorder()
	ownerAuditRouter(t, f.Handler, userID).ServeHTTP(
		rec,
		httptest.NewRequest(http.MethodGet, "/v1/me/audit-logs:export?format=json", nil),
	)

	var rows []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, row := range rows {
		if row["resource_id"] == "tr-theirs" {
			t.Fatal("another account's transfer appeared in this owner's export")
		}
	}
	if len(rows) != 1 {
		t.Fatalf("export has %d rows, want only the owner's own 1", len(rows))
	}
}

// An unknown action is refused for the same reason the list refuses one: a typo
// must not come back as an empty file that looks like "you did nothing".
func TestOwnerAuditExportRefusesAnUnknownAction(t *testing.T) {
	db := newOwnerAuditTestDB(t)
	f, _ := newOwnerAuditHandler(t, db, "700000306", 990706)

	rec := httptest.NewRecorder()
	ownerAuditRouter(t, f.Handler, 990706).ServeHTTP(
		rec,
		httptest.NewRequest(http.MethodGet, "/v1/me/audit-logs:export?format=csv&action=bogus.action", nil),
	)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}
}
