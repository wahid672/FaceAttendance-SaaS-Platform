package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/faceattendance/go-api/internal/middleware"
	"github.com/faceattendance/go-api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserHandler struct {
	userService service.UserService
}

type EmployeeHandler = UserHandler

func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func NewEmployeeHandler(employeeService service.EmployeeService) *EmployeeHandler {
	return NewUserHandler(employeeService)
}

type CreateUserRequest struct {
	OfficeID     *string `json:"office_id,omitempty"`
	Name         string  `json:"name" binding:"required"`
	Email        string  `json:"email" binding:"required,email"`
	Password     string  `json:"password" binding:"required,min=6"`
	UserCode     string  `json:"user_code"`
	EmployeeCode string  `json:"employee_code"`
}

type CreateEmployeeRequest = CreateUserRequest

type BulkCreateUsersRequest struct {
	Users []CreateUserRequest `json:"users"`
}

type BulkDeleteUsersRequest struct {
	UserIDs []string `json:"user_ids"`
}

func getCallerUserID(c *gin.Context) (uuid.UUID, bool) {
	if val, exists := c.Get(middleware.CtxKeyUserID); exists {
		if id, ok := val.(uuid.UUID); ok {
			return id, true
		}
	}
	if val, exists := c.Get(middleware.CtxKeyEmployeeID); exists {
		if id, ok := val.(uuid.UUID); ok {
			return id, true
		}
	}
	return uuid.Nil, false
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	tenantIDVal, exists := c.Get(middleware.CtxKeyTenantID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Unauthorized: missing tenant context"})
		return
	}
	tenantID := tenantIDVal.(uuid.UUID)

	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request payload: " + err.Error(),
		})
		return
	}

	code := strings.TrimSpace(req.UserCode)
	if code == "" {
		code = strings.TrimSpace(req.EmployeeCode)
	}
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Field 'user_code' (atau 'employee_code') is required",
		})
		return
	}

	var officeUUID *uuid.UUID
	if req.OfficeID != nil && *req.OfficeID != "" {
		parsed, err := uuid.Parse(*req.OfficeID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Invalid office_id format: must be valid UUID",
			})
			return
		}
		officeUUID = &parsed
	}

	serviceReq := service.CreateUserRequest{
		TenantID:     tenantID,
		OfficeID:     officeUUID,
		Name:         req.Name,
		Email:        req.Email,
		Password:     req.Password,
		UserCode:     code,
		EmployeeCode: code,
	}

	user, err := h.userService.CreateUser(c.Request.Context(), serviceReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "User created successfully",
		"user": gin.H{
			"id":            user.ID.String(),
			"tenant_id":     user.TenantID.String(),
			"office_id":     user.OfficeID,
			"name":          user.Name,
			"email":         user.Email,
			"user_code":     user.UserCode,
			"employee_code": user.UserCode,
			"is_active":     user.IsActive,
			"is_enrolled":   false,
			"created_at":    user.CreatedAt,
		},
	})
}

func (h *UserHandler) CreateEmployee(c *gin.Context) {
	h.CreateUser(c)
}

func (h *UserHandler) BulkCreateUsers(c *gin.Context) {
	tenantIDVal, exists := c.Get(middleware.CtxKeyTenantID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Unauthorized: missing tenant context"})
		return
	}
	tenantID := tenantIDVal.(uuid.UUID)

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Failed to read request body"})
		return
	}

	var rawUsers []CreateUserRequest
	var wrapper BulkCreateUsersRequest

	if err := json.Unmarshal(bodyBytes, &wrapper); err == nil && len(wrapper.Users) > 0 {
		rawUsers = wrapper.Users
	} else if err := json.Unmarshal(bodyBytes, &rawUsers); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid JSON format. Expected { 'users': [...] } or array of users [ {...} ]",
		})
		return
	}

	if len(rawUsers) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "List of users cannot be empty"})
		return
	}

	var serviceReqs []service.CreateUserRequest
	for _, u := range rawUsers {
		code := strings.TrimSpace(u.UserCode)
		if code == "" {
			code = strings.TrimSpace(u.EmployeeCode)
		}

		var officeUUID *uuid.UUID
		if u.OfficeID != nil && *u.OfficeID != "" {
			if parsed, err := uuid.Parse(*u.OfficeID); err == nil {
				officeUUID = &parsed
			}
		}

		serviceReqs = append(serviceReqs, service.CreateUserRequest{
			TenantID:     tenantID,
			OfficeID:     officeUUID,
			Name:         u.Name,
			Email:        u.Email,
			Password:     u.Password,
			UserCode:     code,
			EmployeeCode: code,
		})
	}

	result, err := h.userService.BulkCreateUsers(c.Request.Context(), tenantID, serviceReqs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	statusCode := http.StatusCreated
	if result.FailedCount > 0 && result.SuccessCount == 0 {
		statusCode = http.StatusBadRequest
	}

	c.JSON(statusCode, gin.H{
		"success":         result.SuccessCount > 0,
		"message":         "Bulk create users completed",
		"total_requested": result.TotalRequested,
		"success_count":   result.SuccessCount,
		"failed_count":    result.FailedCount,
		"errors":          result.Errors,
		"data":            result.Users,
	})
}

func (h *UserHandler) ImportUsersCSV(c *gin.Context) {
	tenantIDVal, exists := c.Get(middleware.CtxKeyTenantID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Unauthorized: missing tenant context"})
		return
	}
	tenantID := tenantIDVal.(uuid.UUID)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		fileHeader, err = c.FormFile("csv")
	}
	if fileHeader == nil || err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "File CSV is required in form-data field 'file' (or 'csv')",
		})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Failed to read CSV file: " + err.Error()})
		return
	}
	defer file.Close()

	result, err := h.userService.ImportUsersCSV(c.Request.Context(), tenantID, file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	statusCode := http.StatusOK
	if result.FailedCount > 0 && result.SuccessCount == 0 {
		statusCode = http.StatusBadRequest
	}

	c.JSON(statusCode, gin.H{
		"success":         result.SuccessCount > 0,
		"message":         "Import users from CSV completed",
		"total_requested": result.TotalRequested,
		"success_count":   result.SuccessCount,
		"failed_count":    result.FailedCount,
		"errors":          result.Errors,
		"data":            result.Users,
	})
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	tenantIDVal, exists := c.Get(middleware.CtxKeyTenantID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Unauthorized: missing tenant context"})
		return
	}
	tenantID := tenantIDVal.(uuid.UUID)

	callerID, exists := getCallerUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Unauthorized: missing caller user context"})
		return
	}

	targetIDParam := c.Param("id")
	targetID, err := uuid.Parse(targetIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid user ID parameter: must be a valid UUID",
		})
		return
	}

	if targetID == callerID {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Gagal: Tidak dapat menghapus akun diri sendiri",
		})
		return
	}

	if err := h.userService.DeleteUser(c.Request.Context(), tenantID, callerID, targetID); err != nil {
		if strings.Contains(err.Error(), "cannot delete your own") {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Gagal: Tidak dapat menghapus akun diri sendiri"})
			return
		}
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "User deleted successfully",
		"deleted_id": targetID.String(),
	})
}

func (h *UserHandler) DeleteEmployee(c *gin.Context) {
	h.DeleteUser(c)
}

func (h *UserHandler) BulkDeleteUsers(c *gin.Context) {
	tenantIDVal, exists := c.Get(middleware.CtxKeyTenantID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Unauthorized: missing tenant context"})
		return
	}
	tenantID := tenantIDVal.(uuid.UUID)

	callerID, exists := getCallerUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Unauthorized: missing caller user context"})
		return
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Failed to read request body"})
		return
	}

	var idStrings []string
	var wrapper BulkDeleteUsersRequest

	if err := json.Unmarshal(bodyBytes, &wrapper); err == nil && len(wrapper.UserIDs) > 0 {
		idStrings = wrapper.UserIDs
	} else if err := json.Unmarshal(bodyBytes, &idStrings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid JSON format. Expected { 'user_ids': ['...'] } or array ['...']",
		})
		return
	}

	var parsedUUIDs []uuid.UUID
	for _, s := range idStrings {
		if parsed, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
			parsedUUIDs = append(parsedUUIDs, parsed)
		}
	}

	if len(parsedUUIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "No valid user UUIDs provided"})
		return
	}

	res, err := h.userService.BulkDeleteUsers(c.Request.Context(), tenantID, callerID, parsedUUIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":         true,
		"message":         "Bulk delete completed",
		"total_requested": res.TotalRequested,
		"deleted_count":   res.DeletedCount,
		"skipped_self":    res.SkippedSelf,
		"deleted_ids":     res.DeletedIDs,
	})
}

func (h *UserHandler) EnrollFace(c *gin.Context) {
	tenantIDVal, exists := c.Get(middleware.CtxKeyTenantID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Unauthorized: missing tenant context"})
		return
	}
	tenantID := tenantIDVal.(uuid.UUID)

	callerID, exists := getCallerUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Unauthorized: missing user context"})
		return
	}
	targetUserID := callerID

	// Allow specifying user_id or employee_id override
	if queryUserID := c.PostForm("user_id"); queryUserID != "" {
		if parsed, err := uuid.Parse(queryUserID); err == nil {
			targetUserID = parsed
		}
	} else if queryUserID := c.PostForm("employee_id"); queryUserID != "" {
		if parsed, err := uuid.Parse(queryUserID); err == nil {
			targetUserID = parsed
		}
	} else if queryUserID := c.Query("user_id"); queryUserID != "" {
		if parsed, err := uuid.Parse(queryUserID); err == nil {
			targetUserID = parsed
		}
	} else if queryUserID := c.Query("employee_id"); queryUserID != "" {
		if parsed, err := uuid.Parse(queryUserID); err == nil {
			targetUserID = parsed
		}
	}

	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Failed to parse multipart form: " + err.Error()})
		return
	}

	var uploadedFiles []service.UploadedFile

	// 1. Check for multiple files with key "images"
	if form := c.Request.MultipartForm; form != nil && form.File != nil {
		if files, ok := form.File["images"]; ok {
			for _, fileHeader := range files {
				file, err := fileHeader.Open()
				if err != nil {
					continue
				}
				data, err := io.ReadAll(file)
				file.Close()
				if err == nil && len(data) > 0 {
					uploadedFiles = append(uploadedFiles, service.UploadedFile{
						Filename: fileHeader.Filename,
						Data:     data,
					})
				}
			}
		}
	}

	// 2. Check for single file with key "image" or "file"
	if len(uploadedFiles) == 0 {
		for _, key := range []string{"image", "file"} {
			fileHeader, err := c.FormFile(key)
			if err == nil && fileHeader != nil {
				file, err := fileHeader.Open()
				if err == nil {
					data, err := io.ReadAll(file)
					file.Close()
					if err == nil && len(data) > 0 {
						uploadedFiles = append(uploadedFiles, service.UploadedFile{
							Filename: fileHeader.Filename,
							Data:     data,
						})
						break
					}
				}
			}
		}
	}

	if len(uploadedFiles) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "No image files provided. Please upload under form field 'images' (3-5 photos) or 'image' (1 photo).",
		})
		return
	}

	embedding, err := h.userService.EnrollFace(c.Request.Context(), tenantID, targetUserID, uploadedFiles)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":              true,
		"message":              "Face enrolled and registered successfully",
		"user_id":              targetUserID.String(),
		"employee_id":          targetUserID.String(),
		"samples_processed":    len(uploadedFiles),
		"embedding_dimensions": len(embedding),
	})
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	tenantIDVal, _ := c.Get(middleware.CtxKeyTenantID)
	tenantID := tenantIDVal.(uuid.UUID)

	callerID, _ := getCallerUserID(c)

	user, err := h.userService.GetUser(c.Request.Context(), tenantID, callerID)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"user": gin.H{
			"id":                 user.ID.String(),
			"tenant_id":          user.TenantID.String(),
			"office_id":          user.OfficeID,
			"name":               user.Name,
			"email":              user.Email,
			"user_code":          user.UserCode,
			"employee_code":      user.UserCode,
			"is_active":          user.IsActive,
			"is_enrolled":        user.FaceEmbedding != nil,
			"face_registered_at": user.FaceRegisteredAt,
		},
		"employee": gin.H{
			"id":                 user.ID.String(),
			"tenant_id":          user.TenantID.String(),
			"office_id":          user.OfficeID,
			"name":               user.Name,
			"email":              user.Email,
			"user_code":          user.UserCode,
			"employee_code":      user.UserCode,
			"is_active":          user.IsActive,
			"is_enrolled":        user.FaceEmbedding != nil,
			"face_registered_at": user.FaceRegisteredAt,
		},
	})
}
