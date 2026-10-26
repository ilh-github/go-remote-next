package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func newUserDeviceGroupsHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := openTestPostgres(t)
	migrateTestSchema(t, db,
		&models.User{},
		&models.Device{},
		&models.UserDeviceGroup{},
		&models.UserDeviceGroupMember{},
	)
	return db
}

func seedUserDeviceGroupsHandlerUser(t *testing.T, db *gorm.DB, id uint, username string) {
	t.Helper()
	clearUserDeviceGroupsHandlerFixture(t, db, id)
	_ = db.Exec("DELETE FROM users WHERE id = ?", id).Error
	if err := db.Exec(
		"INSERT INTO users (id, username, password) VALUES (?, ?, 'test-hash')",
		id, username,
	).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() {
		clearUserDeviceGroupsHandlerFixture(t, db, id)
		_ = db.Exec("DELETE FROM users WHERE id = ?", id).Error
	})
}

func clearUserDeviceGroupsHandlerFixture(t *testing.T, db *gorm.DB, userID uint) {
	t.Helper()
	for _, stmt := range []string{
		"DELETE FROM user_device_group_members WHERE group_id IN (SELECT id FROM user_device_groups WHERE user_id = ?)",
		"DELETE FROM user_device_groups WHERE user_id = ?",
		"DELETE FROM devices WHERE user_id = ?",
	} {
		if err := db.Exec(stmt, userID).Error; err != nil {
			t.Fatalf("clear fixture for user %d: %v", userID, err)
		}
	}
}

func seedUserDeviceGroupsHandlerDevice(t *testing.T, db *gorm.DB, deviceID, uuid string, ownerID uint) {
	t.Helper()
	_ = db.Exec("DELETE FROM devices WHERE device_id = ?", deviceID).Error
	owner := ownerID
	if err := db.Create(&models.Device{
		DeviceID:   deviceID,
		DeviceUUID: uuid,
		DeviceName: "group-handler-device",
		UserID:     &owner,
	}).Error; err != nil {
		t.Fatalf("seed device: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Exec("DELETE FROM devices WHERE device_id = ?", deviceID).Error
	})
}

func userDeviceGroupsRequest(h http.Handler, method, path string, userID uint, body any, params gin.Params) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		encoded, _ := json.Marshal(body)
		reader = bytes.NewReader(encoded)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	c.Params = params
	c.Set("auth.user_id", userID)
	h.ServeHTTP(c.Writer, c.Request)
	return rec
}

func TestUserDeviceGroupsHandlerCreatesAndListsOwnerScopedGroups(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newUserDeviceGroupsHandlerTestDB(t)
	const (
		userA = uint(991431)
		userB = uint(991432)
	)
	seedUserDeviceGroupsHandlerUser(t, db, userA, "group-handler-owner-a")
	seedUserDeviceGroupsHandlerUser(t, db, userB, "group-handler-owner-b")
	h := NewUserDeviceGroupsHandler(service.NewUserDeviceGroupService(db))

	router := gin.New()
	router.POST("/groups", func(c *gin.Context) {
		c.Set("auth.user_id", userA)
		h.Create(c)
	})
	router.GET("/groups", func(c *gin.Context) {
		uid := userB
		if c.Query("owner") == "a" {
			uid = userA
		}
		c.Set("auth.user_id", uid)
		h.List(c)
	})

	created := userDeviceGroupsRequest(router, http.MethodPost, "/groups", userA, map[string]any{
		"name":        " Office ",
		"description": "weekly maintenance",
		"color":       "#1f6feb",
	}, nil)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", created.Code, created.Body.String())
	}
	var group struct {
		ID        uint     `json:"id"`
		Name      string   `json:"name"`
		DeviceIDs []string `json:"device_ids"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &group); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if group.ID == 0 || group.Name != "Office" || group.DeviceIDs == nil || len(group.DeviceIDs) != 0 {
		t.Fatalf("created group = %+v", group)
	}

	ownerAList := userDeviceGroupsRequest(router, http.MethodGet, "/groups?owner=a", userA, nil, nil)
	if ownerAList.Code != http.StatusOK {
		t.Fatalf("owner-a list status = %d, body = %s", ownerAList.Code, ownerAList.Body.String())
	}
	if !bytes.Contains(ownerAList.Body.Bytes(), []byte("Office")) {
		t.Fatalf("owner-a list does not contain group: %s", ownerAList.Body.String())
	}

	ownerBList := userDeviceGroupsRequest(router, http.MethodGet, "/groups", userB, nil, nil)
	if ownerBList.Code != http.StatusOK {
		t.Fatalf("owner-b list status = %d, body = %s", ownerBList.Code, ownerBList.Body.String())
	}
	var listB struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(ownerBList.Body.Bytes(), &listB); err != nil {
		t.Fatalf("decode owner-b list: %v", err)
	}
	if len(listB.Items) != 0 {
		t.Fatalf("owner-b sees owner-a groups: %s", ownerBList.Body.String())
	}
}

func TestUserDeviceGroupsHandlerRejectsForeignDeviceMembership(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newUserDeviceGroupsHandlerTestDB(t)
	const (
		userA   = uint(991441)
		userB   = uint(991442)
		deviceB = "885000442"
	)
	seedUserDeviceGroupsHandlerUser(t, db, userA, "group-handler-member-a")
	seedUserDeviceGroupsHandlerUser(t, db, userB, "group-handler-member-b")
	seedUserDeviceGroupsHandlerDevice(t, db, deviceB, "uuid-group-handler-foreign", userB)
	svc := service.NewUserDeviceGroupService(db)
	group, err := svc.Create(t.Context(), userA, service.UserDeviceGroupInput{Name: "Restricted"})
	if err != nil {
		t.Fatalf("seed group: %v", err)
	}
	h := NewUserDeviceGroupsHandler(svc)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("auth.user_id", userA)
		c.Next()
	})
	router.PUT("/groups/:group_id/devices", h.ReplaceDevices)
	rec := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]any{"device_ids": []string{deviceB}})
	req := httptest.NewRequest(http.MethodPut, "/groups/"+strconv.FormatUint(uint64(group.ID), 10)+"/devices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s, want 403", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("DEVICE_NOT_OWNED")) {
		t.Fatalf("problem code missing: %s", rec.Body.String())
	}
}

func TestUserDeviceGroupsHandlerUpdatesAndDeletes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newUserDeviceGroupsHandlerTestDB(t)
	const userA = uint(991451)
	seedUserDeviceGroupsHandlerUser(t, db, userA, "group-handler-update-delete")
	svc := service.NewUserDeviceGroupService(db)
	group, err := svc.Create(t.Context(), userA, service.UserDeviceGroupInput{Name: "Old name"})
	if err != nil {
		t.Fatalf("seed group: %v", err)
	}
	h := NewUserDeviceGroupsHandler(svc)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("auth.user_id", userA)
		c.Next()
	})
	router.PATCH("/groups/:group_id", h.Update)
	router.DELETE("/groups/:group_id", h.Delete)

	patchBody, _ := json.Marshal(map[string]any{"name": "New name", "description": "updated"})
	patchReq := httptest.NewRequest(http.MethodPatch, "/groups/"+strconv.FormatUint(uint64(group.ID), 10), bytes.NewReader(patchBody))
	patchReq.Header.Set("Content-Type", "application/json")
	patchRec := httptest.NewRecorder()
	router.ServeHTTP(patchRec, patchReq)
	if patchRec.Code != http.StatusOK {
		t.Fatalf("patch status = %d, body = %s", patchRec.Code, patchRec.Body.String())
	}
	if !bytes.Contains(patchRec.Body.Bytes(), []byte("New name")) {
		t.Fatalf("patched group missing new name: %s", patchRec.Body.String())
	}

	deleteRec := httptest.NewRecorder()
	deleteReq := httptest.NewRequest(http.MethodDelete, "/groups/"+strconv.FormatUint(uint64(group.ID), 10), nil)
	router.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", deleteRec.Code, deleteRec.Body.String())
	}
	groups, err := svc.List(t.Context(), userA)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(groups) != 0 {
		t.Fatalf("groups after handler delete = %+v", groups)
	}
}
