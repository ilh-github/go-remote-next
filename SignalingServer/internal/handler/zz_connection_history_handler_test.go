package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// The HTTP shape of these two endpoints matters as much as the service behind
// them, because the client is written against it: the id form has to reject a
// non-numeric id before it reaches the database, and a row belonging to somebody
// else has to answer 404 rather than 403, since telling the caller the id exists
// would confirm something they should not learn.

func newConnectionHistoryHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := openTestPostgres(t)
	applyTestMigration(t, db)
	if err := db.Exec("DELETE FROM connection_histories").Error; err != nil {
		t.Fatalf("clear: %v", err)
	}
	return db
}

func seedConnectionHistoryRow(t *testing.T, db *gorm.DB, userID uint, deviceID string) uint {
	t.Helper()
	// `connection_histories.user_id` is a foreign key, so the owner has to exist
	// before a row claiming them can.
	if err := db.Exec(
		"INSERT INTO users (id, username, password) VALUES (?, ?, 'test-hash') ON CONFLICT (id) DO NOTHING",
		userID, fmt.Sprintf("qa-conn-handler-%d", userID),
	).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	row := models.ConnectionHistory{}
	if err := db.Raw(
		"INSERT INTO connection_histories (user_id, device_id, device_name, status) VALUES (?, ?, ?, 'success') RETURNING id",
		userID, deviceID, "Seeded "+deviceID,
	).Scan(&row).Error; err != nil {
		t.Fatalf("seed history: %v", err)
	}
	return row.ID
}

func newConnectionHistoryHandler(t *testing.T, db *gorm.DB) *DeviceHandler {
	t.Helper()
	return NewDeviceHandler(nil, nil, service.NewConnectionService(db), nil, nil, db)
}

func TestDeleteConnectionHandlerRejectsANonNumericId(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newConnectionHistoryHandlerTestDB(t)
	h := newConnectionHistoryHandler(t, db)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Params = gin.Params{{Key: "id", Value: "not-a-number"}}
	c.Set("auth.user_id", uint(990601))
	c.Request = httptest.NewRequest(http.MethodDelete, "/v1/me/connections/not-a-number", nil)

	h.DeleteConnection(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteConnectionHandlerAnswers404ForAnotherOwnersRow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newConnectionHistoryHandlerTestDB(t)
	const (
		owner    = uint(990602)
		intruder = uint(990603)
	)
	theirs := seedConnectionHistoryRow(t, db, owner, "700000201")
	h := newConnectionHistoryHandler(t, db)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprint(theirs)}}
	c.Set("auth.user_id", intruder)
	c.Request = httptest.NewRequest(http.MethodDelete, "/v1/me/connections/"+fmt.Sprint(theirs), nil)

	h.DeleteConnection(c)

	// 404, not 403: the caller's own view of that id is that it does not exist,
	// and confirming it does would leak that the id is real.
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", rec.Code, rec.Body.String())
	}
	var left int64
	if err := db.Model(&models.ConnectionHistory{}).Where("id = ?", theirs).Count(&left).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if left != 1 {
		t.Fatal("another owner's row was removed anyway")
	}
}

func TestClearConnectionsHandlerReportsHowManyRowsWent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newConnectionHistoryHandlerTestDB(t)
	const owner = uint(990604)
	seedConnectionHistoryRow(t, db, owner, "700000202")
	seedConnectionHistoryRow(t, db, owner, "700000203")
	h := newConnectionHistoryHandler(t, db)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("auth.user_id", owner)
	c.Request = httptest.NewRequest(http.MethodDelete, "/v1/me/connections", bytes.NewReader(nil))

	h.ClearConnections(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body = %s", rec.Code, rec.Body.String())
	}
	var response struct {
		Removed int64 `json:"removed"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// The UI confirms with this number, so it has to be the real count.
	if response.Removed != 2 {
		t.Fatalf("removed = %d, want 2", response.Removed)
	}
	var left int64
	if err := db.Model(&models.ConnectionHistory{}).Where("user_id = ?", owner).Count(&left).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if left != 0 {
		t.Fatalf("%d rows survived the clear", left)
	}
}
