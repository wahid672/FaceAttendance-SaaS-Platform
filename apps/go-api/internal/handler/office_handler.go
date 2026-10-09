package handler

import (
	"net/http"

	"github.com/faceattendance/go-api/internal/middleware"
	"github.com/faceattendance/go-api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OfficeHandler struct {
	officeService service.OfficeService
}

func NewOfficeHandler(officeService service.OfficeService) *OfficeHandler {
	return &OfficeHandler{officeService: officeService}
}

type CreateOfficeInput struct {
	Name         string  `json:"name" binding:"required"`
	Latitude     float64 `json:"latitude" binding:"required"`
	Longitude    float64 `json:"longitude" binding:"required"`
	RadiusMeters int     `json:"radius_meters"`
}

type UpdateOfficeInput struct {
	Name         string   `json:"name"`
	Latitude     *float64 `json:"latitude"`
	Longitude    *float64 `json:"longitude"`
	RadiusMeters *int     `json:"radius_meters"`
}

func (h *OfficeHandler) ListOffices(c *gin.Context) {
	tenantIDVal, exists := c.Get(middleware.CtxKeyTenantID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Unauthorized: missing tenant context"})
		return
	}
	tenantID := tenantIDVal.(uuid.UUID)

	offices, err := h.officeService.ListOffices(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    offices,
	})
}

func (h *OfficeHandler) GetOffice(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid office ID format"})
		return
	}

	office, err := h.officeService.GetOffice(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": office})
}

func (h *OfficeHandler) CreateOffice(c *gin.Context) {
	tenantIDVal, exists := c.Get(middleware.CtxKeyTenantID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Unauthorized: missing tenant context"})
		return
	}
	tenantID := tenantIDVal.(uuid.UUID)

	var req CreateOfficeInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	serviceReq := service.CreateOfficeRequest{
		Name:         req.Name,
		Latitude:     req.Latitude,
		Longitude:    req.Longitude,
		RadiusMeters: req.RadiusMeters,
	}

	office, err := h.officeService.CreateOffice(c.Request.Context(), tenantID, serviceReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Office created successfully",
		"data":    office,
	})
}

func (h *OfficeHandler) UpdateOffice(c *gin.Context) {
	tenantIDVal, exists := c.Get(middleware.CtxKeyTenantID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Unauthorized: missing tenant context"})
		return
	}
	tenantID := tenantIDVal.(uuid.UUID)

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid office ID format"})
		return
	}

	var req UpdateOfficeInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	serviceReq := service.UpdateOfficeRequest{
		Name:         req.Name,
		Latitude:     req.Latitude,
		Longitude:    req.Longitude,
		RadiusMeters: req.RadiusMeters,
	}

	updated, err := h.officeService.UpdateOffice(c.Request.Context(), tenantID, id, serviceReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Office updated successfully",
		"data":    updated,
	})
}

func (h *OfficeHandler) DeleteOffice(c *gin.Context) {
	tenantIDVal, exists := c.Get(middleware.CtxKeyTenantID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Unauthorized: missing tenant context"})
		return
	}
	tenantID := tenantIDVal.(uuid.UUID)

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid office ID format"})
		return
	}

	if err := h.officeService.DeleteOffice(c.Request.Context(), tenantID, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Office deleted successfully",
	})
}
