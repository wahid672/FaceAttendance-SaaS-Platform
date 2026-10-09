package handler

import (
	"net/http"
	"strconv"

	"github.com/faceattendance/go-api/internal/model"
	"github.com/faceattendance/go-api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PlatformHandler struct {
	platformService service.PlatformService
}

func NewPlatformHandler(platformService service.PlatformService) *PlatformHandler {
	return &PlatformHandler{platformService: platformService}
}

type UpdateSettingsRequest struct {
	AppName      string  `json:"app_name" binding:"required"`
	LogoURL      *string `json:"logo_url"`
	FaviconURL   *string `json:"favicon_url"`
	CompanyName  *string `json:"company_name"`
	SupportEmail *string `json:"support_email"`
	FooterText   *string `json:"footer_text"`
}

type UpdateTenantRequest struct {
	Name      string `json:"name"`
	Subdomain string `json:"subdomain"`
	IsActive  *bool  `json:"is_active"`
}

func (h *PlatformHandler) GetSettings(c *gin.Context) {
	settings, err := h.platformService.GetSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": settings})
}

func (h *PlatformHandler) UpdateSettings(c *gin.Context) {
	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	settings := &model.PlatformSettings{
		AppName:      req.AppName,
		LogoURL:      req.LogoURL,
		FaviconURL:   req.FaviconURL,
		CompanyName:  req.CompanyName,
		SupportEmail: req.SupportEmail,
		FooterText:   req.FooterText,
	}

	updated, err := h.platformService.UpdateSettings(c.Request.Context(), settings)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Platform settings updated successfully",
		"data":    updated,
	})
}

func (h *PlatformHandler) ListTenants(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")

	result, err := h.platformService.ListTenants(c.Request.Context(), page, limit, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"total":   result.Total,
		"page":    result.Page,
		"limit":   result.Limit,
		"data":    result.Tenants,
	})
}

func (h *PlatformHandler) GetTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid tenant ID format"})
		return
	}

	tenant, err := h.platformService.GetTenant(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": tenant})
}

func (h *PlatformHandler) CreateTenant(c *gin.Context) {
	var req service.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	tenant, adminUser, err := h.platformService.CreateTenant(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Tenant and administrator created successfully",
		"tenant":  tenant,
		"admin": gin.H{
			"id":        adminUser.ID.String(),
			"name":      adminUser.Name,
			"email":     adminUser.Email,
			"user_code": adminUser.UserCode,
			"role":      adminUser.Role,
		},
	})
}

func (h *PlatformHandler) UpdateTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid tenant ID format"})
		return
	}

	var req UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	updated, err := h.platformService.UpdateTenant(c.Request.Context(), id, req.Name, req.Subdomain, isActive)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Tenant updated successfully",
		"data":    updated,
	})
}
