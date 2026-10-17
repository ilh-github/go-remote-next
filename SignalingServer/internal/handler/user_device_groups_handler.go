package handler

import (
	"errors"
	"net/http"
	"strconv"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
)

type UserDeviceGroupsHandler struct {
	groups *service.UserDeviceGroupService
}

func NewUserDeviceGroupsHandler(groups *service.UserDeviceGroupService) *UserDeviceGroupsHandler {
	return &UserDeviceGroupsHandler{groups: groups}
}

func (h *UserDeviceGroupsHandler) List(c *gin.Context) {
	groups, err := h.groups.List(c.Request.Context(), middleware.MustUserID(c))
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": groups})
}

func (h *UserDeviceGroupsHandler) Create(c *gin.Context) {
	var req userDeviceGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	group, err := h.groups.Create(c.Request.Context(), middleware.MustUserID(c), service.UserDeviceGroupInput{
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
	})
	if err != nil {
		writeUserDeviceGroupProblem(c, err)
		return
	}
	c.JSON(http.StatusCreated, group)
}

func (h *UserDeviceGroupsHandler) Update(c *gin.Context) {
	groupID, err := parseUserDeviceGroupID(c)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid group_id")
		return
	}
	var req userDeviceGroupPatchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	group, err := h.groups.Update(c.Request.Context(), middleware.MustUserID(c), groupID, service.UserDeviceGroupPatch{
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
	})
	if err != nil {
		writeUserDeviceGroupProblem(c, err)
		return
	}
	c.JSON(http.StatusOK, group)
}

func (h *UserDeviceGroupsHandler) Delete(c *gin.Context) {
	groupID, err := parseUserDeviceGroupID(c)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid group_id")
		return
	}
	if err := h.groups.Delete(c.Request.Context(), middleware.MustUserID(c), groupID); err != nil {
		writeUserDeviceGroupProblem(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *UserDeviceGroupsHandler) ReplaceDevices(c *gin.Context) {
	groupID, err := parseUserDeviceGroupID(c)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid group_id")
		return
	}
	var req userDeviceGroupDevicesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	group, err := h.groups.ReplaceDevices(c.Request.Context(), middleware.MustUserID(c), groupID, req.DeviceIDs)
	if err != nil {
		writeUserDeviceGroupProblem(c, err)
		return
	}
	c.JSON(http.StatusOK, group)
}

type userDeviceGroupReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

type userDeviceGroupPatchReq struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Color       *string `json:"color"`
}

type userDeviceGroupDevicesReq struct {
	DeviceIDs []string `json:"device_ids"`
}

func parseUserDeviceGroupID(c *gin.Context) (uint, error) {
	id, err := strconv.ParseUint(c.Param("group_id"), 10, 32)
	return uint(id), err
}

func writeUserDeviceGroupProblem(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrUserDeviceGroupNotFound):
		ProblemNotFound(c, ProblemCodeDeviceGroupMissing, "Device group not found")
	case errors.Is(err, service.ErrUserDeviceGroupNameExists):
		ProblemConflict(c, ProblemCodeDeviceGroupExists, "A device group with that name already exists")
	case errors.Is(err, service.ErrUserDeviceGroupInvalid):
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
	case errors.Is(err, service.ErrUserDeviceNotOwned):
		ProblemForbidden(c, ProblemCodeDeviceNotOwned, "One or more devices are not owned by this user")
	default:
		ProblemInternal(c, err.Error())
	}
}
