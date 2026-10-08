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

type LoginResponse struct {
	Success  bool   `json:"success"`
	Token    string `json:"token"`
	Employee struct {
		ID           string  `json:"id"`
		TenantID     string  `json:"tenant_id"`
		OfficeID     *string `json:"office_id,omitempty"`
		Name         string  `json:"name"`
		Email        string  `json:"email"`
		EmployeeCode string  `json:"employee_code"`
		IsActive     bool    `json:"is_active"`
		IsEnrolled   bool    `json:"is_enrolled"`
	} `json:"employee"`
	Tenant struct {
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

	resp.Employee.ID = employee.ID.String()
	resp.Employee.TenantID = employee.TenantID.String()
	if employee.OfficeID != nil {
		officeIDStr := employee.OfficeID.String()
		resp.Employee.OfficeID = &officeIDStr
	}
	resp.Employee.Name = employee.Name
	resp.Employee.Email = employee.Email
	resp.Employee.EmployeeCode = employee.EmployeeCode
	resp.Employee.IsActive = employee.IsActive
	resp.Employee.IsEnrolled = employee.FaceEmbedding != nil

	resp.Tenant.ID = tenant.ID.String()
	resp.Tenant.Name = tenant.Name
	resp.Tenant.Subdomain = tenant.Subdomain

	c.JSON(http.StatusOK, resp)
}
