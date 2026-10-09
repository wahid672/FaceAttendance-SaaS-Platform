package handler

import (
	"net/http"

	"github.com/faceattendance/go-api/internal/service"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type UserData struct {
	ID           string  `json:"id"`
	TenantID     *string `json:"tenant_id,omitempty"`
	OfficeID     *string `json:"office_id,omitempty"`
	Role         string  `json:"role"`
	Name         string  `json:"name"`
	Email        *string `json:"email,omitempty"`
	UserCode     string  `json:"user_code"`
	EmployeeCode string  `json:"employee_code,omitempty"`
	IsActive     bool    `json:"is_active"`
	IsEnrolled   bool    `json:"is_enrolled"`
}

type TenantData struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Subdomain string `json:"subdomain"`
}

type LoginResponse struct {
	Success  bool        `json:"success"`
	Token    string      `json:"token"`
	User     UserData    `json:"user"`
	Employee UserData    `json:"employee"`
	Tenant   *TenantData `json:"tenant,omitempty"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request payload: " + err.Error(),
		})
		return
	}

	token, employee, tenant, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	var resp LoginResponse
	resp.Success = true
	resp.Token = token

	code := employee.UserCode
	if code == "" {
		code = employee.EmployeeCode
	}

	var officeIDStr *string
	if employee.OfficeID != nil {
		s := employee.OfficeID.String()
		officeIDStr = &s
	}

	var tenantIDStr *string
	if employee.TenantID != nil {
		s := employee.TenantID.String()
		tenantIDStr = &s
	}

	userData := UserData{
		ID:           employee.ID.String(),
		TenantID:     tenantIDStr,
		OfficeID:     officeIDStr,
		Role:         employee.Role,
		Name:         employee.Name,
		Email:        employee.Email,
		UserCode:     code,
		EmployeeCode: code,
		IsActive:     employee.IsActive,
		IsEnrolled:   employee.FaceEmbedding != nil,
	}

	resp.User = userData
	resp.Employee = userData

	if tenant != nil {
		resp.Tenant = &TenantData{
			ID:        tenant.ID.String(),
			Name:      tenant.Name,
			Subdomain: tenant.Subdomain,
		}
	}

	c.JSON(http.StatusOK, resp)
}
