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
	TenantID     string  `json:"tenant_id"`
	OfficeID     *string `json:"office_id,omitempty"`
	Name         string  `json:"name"`
	Email        string  `json:"email"`
	UserCode     string  `json:"user_code"`
	EmployeeCode string  `json:"employee_code"`
	IsActive     bool    `json:"is_active"`
	IsEnrolled   bool    `json:"is_enrolled"`
}

type LoginResponse struct {
	Success  bool     `json:"success"`
	Token    string   `json:"token"`
	User     UserData `json:"user"`
	Employee UserData `json:"employee"`
	Tenant   struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Subdomain string `json:"subdomain"`
	} `json:"tenant"`
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

	userData := UserData{
		ID:           employee.ID.String(),
		TenantID:     employee.TenantID.String(),
		OfficeID:     officeIDStr,
		Name:         employee.Name,
		Email:        employee.Email,
		UserCode:     code,
		EmployeeCode: code,
		IsActive:     employee.IsActive,
		IsEnrolled:   employee.FaceEmbedding != nil,
	}

	resp.User = userData
	resp.Employee = userData

	resp.Tenant.ID = tenant.ID.String()
	resp.Tenant.Name = tenant.Name
	resp.Tenant.Subdomain = tenant.Subdomain

	c.JSON(http.StatusOK, resp)
}
