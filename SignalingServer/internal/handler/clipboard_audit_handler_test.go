package handler

import (
	"encoding/json"
	"strings"
	"testing"

	"quickdesk/signaling/internal/models"
)

// QD-909: clipboard sync is the one data path whose payload is by definition
// whatever the user last copied - a password, a token, a private message. The
// audit row therefore records that a transfer happened and how big it was, and
// never the text. These tests pin that boundary, because "just log the content
// for debugging" is the change that would turn an audit trail into a credential
// store.
func TestClipboardAuditNeverStoresTheClipboardText(t *testing.T) {
	db := newClipboardAuditTestDB(t)
	const (
		deviceID = "911111111"
		userID   = uint(9601)
	)
	h, ctx := newClipboardAuditHandler(t, db, deviceID, userID)

	const secret = "hunter2-SUPER-SECRET-TOKEN"
	recorder := recordClipboardAudit(t, h, ctx, clipboardAuditBody{
		DeviceID:   deviceID,
		TransferID: "clip-1",
		Direction:  "upload",
		Status:     "completed",
		MimeType:   "text/plain",
		Bytes:      int64(len(secret)),
	})
	if recorder.Code != 204 {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}

	var rows []models.AuditLog
	if err := db.Find(&rows).Error; err != nil {
		t.Fatalf("load audit rows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("audit rows = %d, want 1", len(rows))
	}
	if strings.Contains(rows[0].Details, secret) {
		t.Fatalf("the clipboard text reached the audit row: %s", rows[0].Details)
	}
	if strings.Contains(rows[0].Details, "SUPER") {
		t.Fatalf("a fragment of the clipboard text reached the audit row: %s", rows[0].Details)
	}
	if rows[0].Action != "clipboard.sync.upload" {
		t.Fatalf("action = %q, want clipboard.sync.upload", rows[0].Action)
	}
}

// The row has to be useful without the text: which device, which direction, how
// many bytes, and what MIME type. Those are what an investigator actually asks.
func TestClipboardAuditRecordsTheMetadataThatIdentifiesTheTransfer(t *testing.T) {
	db := newClipboardAuditTestDB(t)
	const (
		deviceID = "911111112"
		userID   = uint(9602)
	)
	h, ctx := newClipboardAuditHandler(t, db, deviceID, userID)

	recorder := recordClipboardAudit(t, h, ctx, clipboardAuditBody{
		DeviceID:   deviceID,
		TransferID: "clip-meta",
		Direction:  "download",
		Status:     "completed",
		MimeType:   "text/plain",
		Bytes:      4096,
	})
	if recorder.Code != 204 {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}

	var row models.AuditLog
	if err := db.First(&row).Error; err != nil {
		t.Fatalf("load audit row: %v", err)
	}
	var details map[string]any
	if err := json.Unmarshal([]byte(row.Details), &details); err != nil {
		t.Fatalf("details were not JSON: %v", err)
	}
	if details["device_id"] != deviceID {
		t.Fatalf("device_id = %v, want %s", details["device_id"], deviceID)
	}
	if details["mime_type"] != "text/plain" {
		t.Fatalf("mime_type = %v, want text/plain", details["mime_type"])
	}
	if details["bytes"] != float64(4096) {
		t.Fatalf("bytes = %v, want 4096", details["bytes"])
	}
	if row.Action != "clipboard.sync.download" {
		t.Fatalf("action = %q, want clipboard.sync.download", row.Action)
	}
}

// A retried report must not double-count, the same way a retried file transfer
// report does not: the transfer id is the identity.
func TestClipboardAuditIsIdempotentByTransferID(t *testing.T) {
	db := newClipboardAuditTestDB(t)
	const (
		deviceID = "911111113"
		userID   = uint(9603)
	)
	h, ctx := newClipboardAuditHandler(t, db, deviceID, userID)

	body := clipboardAuditBody{
		DeviceID:   deviceID,
		TransferID: "clip-dup",
		Direction:  "upload",
		Status:     "completed",
		MimeType:   "text/plain",
		Bytes:      12,
	}
	first := recordClipboardAudit(t, h, ctx, body)
	second := recordClipboardAudit(t, h, ctx, body)
	if first.Code != 204 || second.Code != 204 {
		t.Fatalf("statuses = %d and %d, want 204 twice", first.Code, second.Code)
	}

	var count int64
	if err := db.Model(&models.AuditLog{}).Count(&count).Error; err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("audit rows = %d, want 1", count)
	}
}

// An unknown direction is refused rather than recorded as something else: the
// audit trail has to say which way the bytes went.
func TestClipboardAuditRejectsUnknownDirectionAndMimeType(t *testing.T) {
	db := newClipboardAuditTestDB(t)
	const (
		deviceID = "911111114"
		userID   = uint(9604)
	)
	h, ctx := newClipboardAuditHandler(t, db, deviceID, userID)

	for _, bad := range []clipboardAuditBody{
		{DeviceID: deviceID, TransferID: "c1", Direction: "sideways", Status: "completed", MimeType: "text/plain", Bytes: 1},
		{DeviceID: deviceID, TransferID: "c2", Direction: "upload", Status: "completed", MimeType: "", Bytes: 1},
		{DeviceID: deviceID, TransferID: "c3", Direction: "upload", Status: "halfway", MimeType: "text/plain", Bytes: 1},
		{DeviceID: deviceID, TransferID: "", Direction: "upload", Status: "completed", MimeType: "text/plain", Bytes: 1},
		{DeviceID: deviceID, TransferID: "c5", Direction: "upload", Status: "completed", MimeType: "text/plain", Bytes: -1},
	} {
		recorder := recordClipboardAudit(t, h, ctx, bad)
		if recorder.Code != 400 {
			t.Fatalf("body %+v: status = %d, want 400", bad, recorder.Code)
		}
	}

	var count int64
	if err := db.Model(&models.AuditLog{}).Count(&count).Error; err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("a refused report still wrote %d rows", count)
	}
}

// A device that belongs to somebody else must not accept a report: the audit
// trail is owner-scoped, and accepting one would let any caller write history
// onto another account's device.
func TestClipboardAuditRefusesADeviceTheCallerDoesNotOwn(t *testing.T) {
	db := newClipboardAuditTestDB(t)
	h, ctx := newClipboardAuditHandler(t, db, "911111115", 9605)
	// A device that exists but belongs to somebody else, which is the case the
	// 403 is for. A device that does not exist at all is a 404, and that is a
	// different answer to a different question.
	const strangerID = "911999999"
	stranger := uint(9799)
	if err := db.Create(&models.User{
		ID: stranger, Username: "clipboard-audit-stranger", Password: "test-hash",
	}).Error; err != nil {
		t.Fatalf("seed stranger: %v", err)
	}
	if err := db.Create(&models.Device{
		DeviceID: strangerID, DeviceUUID: "uuid-clipboard-audit-stranger", UserID: &stranger,
	}).Error; err != nil {
		t.Fatalf("seed stranger device: %v", err)
	}

	recorder := recordClipboardAudit(t, h, ctx, clipboardAuditBody{
		DeviceID:   strangerID,
		TransferID: "clip-other",
		Direction:  "upload",
		Status:     "completed",
		MimeType:   "text/plain",
		Bytes:      10,
	})
	if recorder.Code != 403 {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}
}
