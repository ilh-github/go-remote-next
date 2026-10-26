package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"quickdesk/signaling/internal/models"
)

// QD-909, read half: the owner can already *write* file and clipboard audit
// rows, but until now could not read them back anywhere except the admin
// console. That leaves the investigative question - "what did my controller
// copy at 14:03?" - unanswerable by the one person who can act on the answer.
//
// These tests pin that the read is scoped to the caller's own rows. The write
// endpoints already refuse a device the caller does not own, so the actor on a
// row is its device's owner; the read must not become a way to see anybody
// else's.
func TestOwnerAuditReadsBackTheirOwnTransfers(t *testing.T) {
	db := newOwnerAuditTestDB(t)
	const (
		deviceID = "711111111"
		userID   = uint(9701)
	)
	h, _ := newOwnerAuditHandler(t, db, deviceID, userID)

	recordClipboardAudit(t, h.Clipboard, h.ctx, clipboardAuditBody{
		DeviceID:   deviceID,
		TransferID: "clip-read-1",
		Direction:  "upload",
		Status:     "completed",
		MimeType:   "text/plain",
		Bytes:      42,
	})

	rec := listOwnerAudit(t, h, userID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	items := decodeOwnerAuditItems(t, rec)
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if items[0]["action"] != "clipboard.sync.upload" {
		t.Fatalf("action = %v, want clipboard.sync.upload", items[0]["action"])
	}
	// The byte count is the point of the row: it answers "how much went where"
	// without the text that would turn the trail into a credential store. It
	// lives inside `details`, which the envelope carries as a JSON string, so the
	// assertion reads the decoded details rather than the raw body - matching a
	// literal in the body would be asserting on escaping.
	var details struct {
		Bytes    int64  `json:"bytes"`
		MimeType string `json:"mime_type"`
	}
	if err := json.Unmarshal([]byte(items[0]["details"].(string)), &details); err != nil {
		t.Fatalf("details was not JSON: %v", err)
	}
	if details.Bytes != 42 {
		t.Fatalf("bytes = %d, want 42", details.Bytes)
	}
	if details.MimeType != "text/plain" {
		t.Fatalf("mime_type = %q, want text/plain", details.MimeType)
	}
}

// Somebody else's transfers must not appear. The actor id is the scope, and a
// read that leaked across accounts would make the audit trail a surveillance
// tool instead of an attribution record.
func TestOwnerAuditDoesNotShowAnotherAccountsRows(t *testing.T) {
	db := newOwnerAuditTestDB(t)
	const (
		deviceID = "711111111"
		userID   = uint(9702)
		otherID  = uint(9703)
	)
	h, _ := newOwnerAuditHandler(t, db, deviceID, userID)

	recordClipboardAudit(t, h.Clipboard, h.ctx, clipboardAuditBody{
		DeviceID:   deviceID,
		TransferID: "clip-mine",
		Direction:  "upload",
		Status:     "completed",
		MimeType:   "text/plain",
		Bytes:      10,
	})
	// Another user's row, written directly: the write endpoint would refuse it,
	// so the only way it exists is the admin path or a direct service call.
	if err := db.Create(&models.AuditLog{
		ActorUserID:  otherID,
		Action:       "clipboard.sync.upload",
		ResourceType: "clipboard_sync",
		ResourceID:   "clip-theirs",
		Details:      `{"device_id":"711111111","bytes":999}`,
	}).Error; err != nil {
		t.Fatalf("seed foreign row: %v", err)
	}

	rec := listOwnerAudit(t, h, userID, "")
	items := decodeOwnerAuditItems(t, rec)
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1 - another account's row was visible", len(items))
	}
	if jsonContains(rec.Body.String(), "clip-theirs") {
		t.Fatalf("a foreign transfer id was returned: %s", rec.Body.String())
	}
	if jsonContains(rec.Body.String(), "999") {
		t.Fatalf("a foreign row's details were returned: %s", rec.Body.String())
	}
}

// File transfers and clipboard syncs are the two data-channel payloads, so one
// read has to cover both - otherwise the owner has to know which table a
// transfer landed in to ask about it.
func TestOwnerAuditCoversFileTransfersAndClipboardSyncs(t *testing.T) {
	db := newOwnerAuditTestDB(t)
	const (
		deviceID = "711111111"
		userID   = uint(9704)
	)
	h, _ := newOwnerAuditHandler(t, db, deviceID, userID)

	recordClipboardAudit(t, h.Clipboard, h.ctx, clipboardAuditBody{
		DeviceID:   deviceID,
		TransferID: "clip-both",
		Direction:  "upload",
		Status:     "completed",
		MimeType:   "text/plain",
		Bytes:      7,
	})
	recordFileTransferAudit(t, h.Files, h.ctx, fileTransferAuditBody{
		DeviceID:         deviceID,
		TransferID:       "file-both",
		Direction:        "download",
		Status:           "completed",
		Filename:         "report.pdf",
		BytesTransferred: 2048,
		TotalBytes:       2048,
	})

	rec := listOwnerAudit(t, h, userID, "")
	items := decodeOwnerAuditItems(t, rec)
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2 (one file, one clipboard)", len(items))
	}
	body := rec.Body.String()
	if !jsonContains(body, "file.transfer.download") {
		t.Fatalf("the file transfer was missing: %s", body)
	}
	if !jsonContains(body, "clipboard.sync.upload") {
		t.Fatalf("the clipboard sync was missing: %s", body)
	}
}

// An action filter narrows to one kind, so "just the clipboard" is answerable
// without the client filtering a page it should not have been sent.
func TestOwnerAuditFiltersByAction(t *testing.T) {
	db := newOwnerAuditTestDB(t)
	const (
		deviceID = "711111111"
		userID   = uint(9705)
	)
	h, _ := newOwnerAuditHandler(t, db, deviceID, userID)

	recordClipboardAudit(t, h.Clipboard, h.ctx, clipboardAuditBody{
		DeviceID:   deviceID,
		TransferID: "clip-filter",
		Direction:  "upload",
		Status:     "completed",
		MimeType:   "text/plain",
		Bytes:      3,
	})
	recordFileTransferAudit(t, h.Files, h.ctx, fileTransferAuditBody{
		DeviceID:         deviceID,
		TransferID:       "file-filter",
		Direction:        "upload",
		Status:           "completed",
		Filename:         "a.txt",
		BytesTransferred: 1,
		TotalBytes:       1,
	})

	rec := listOwnerAudit(t, h, userID, "clipboard.sync.upload")
	items := decodeOwnerAuditItems(t, rec)
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if items[0]["action"] != "clipboard.sync.upload" {
		t.Fatalf("action = %v, want clipboard.sync.upload", items[0]["action"])
	}
}

// An unknown action is refused rather than treated as "no filter". Silently
// answering "you did nothing" to a typo is how an investigation gets closed on
// the wrong evidence.
func TestOwnerAuditRejectsAnUnknownAction(t *testing.T) {
	db := newOwnerAuditTestDB(t)
	const (
		deviceID = "711111111"
		userID   = uint(9706)
	)
	h, _ := newOwnerAuditHandler(t, db, deviceID, userID)

	rec := listOwnerAudit(t, h, userID, "recording.delete")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 - an unreadable action must not look like an empty result", rec.Code)
	}
}

// The list is cursor-paginated like every other v1 list, so a busy account does
// not get one unbounded response.
func TestOwnerAuditPaginates(t *testing.T) {
	db := newOwnerAuditTestDB(t)
	const (
		deviceID = "711111111"
		userID   = uint(9707)
	)
	h, _ := newOwnerAuditHandler(t, db, deviceID, userID)

	for i := 0; i < 3; i++ {
		recordClipboardAudit(t, h.Clipboard, h.ctx, clipboardAuditBody{
			DeviceID:   deviceID,
			TransferID: "clip-page-" + string(rune('a'+i)),
			Direction:  "upload",
			Status:     "completed",
			MimeType:   "text/plain",
			Bytes:      int64(i),
		})
	}

	rec := listOwnerAuditWithLimit(t, h, userID, "", 2)
	items := decodeOwnerAuditItems(t, rec)
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2", len(items))
	}
	var page struct {
		NextCursor string `json:"next_cursor"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode page: %v", err)
	}
	if page.NextCursor == "" {
		t.Fatal("a third row exists but no next_cursor was returned")
	}

	rec = listOwnerAuditFromCursor(t, h, userID, "", page.NextCursor)
	items = decodeOwnerAuditItems(t, rec)
	if len(items) != 1 {
		t.Fatalf("second page items = %d, want 1", len(items))
	}
}

func jsonContains(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) &&
		func() bool {
			for i := 0; i+len(needle) <= len(haystack); i++ {
				if haystack[i:i+len(needle)] == needle {
					return true
				}
			}
			return false
		}()
}
