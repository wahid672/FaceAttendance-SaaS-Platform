package handler

import (
	"io"
	"net/http"

	"github.com/faceattendance/go-api/internal/middleware"
	"github.com/faceattendance/go-api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type EmployeeHandler struct {
	employeeService service.EmployeeService
}

func NewEmployeeHandler(employeeService service.EmployeeService) *EmployeeHandler {
	return &EmployeeHandler{employeeService: employeeService}
}

func (h *EmployeeHandler) EnrollFace(c *gin.Context) {
	tenantIDVal, exists := c.Get(middleware.CtxKeyTenantID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Unauthorized: missing tenant context"})
		return
	}
	tenantID := tenantIDVal.(uuid.UUID)

	employeeIDVal, exists := c.Get(middleware.CtxKeyEmployeeID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Unauthorized: missing employee context"})
		return
	}
	targetEmployeeID := employeeIDVal.(uuid.UUID)

	// If an admin/authorized user specifies another employee_id in form or query, allow override
	if queryEmployeeID := c.PostForm("employee_id"); queryEmployeeID != "" {
		if parsed, err := uuid.Parse(queryEmployeeID); err == nil {
			targetEmployeeID = parsed
		}
	} else if queryEmployeeID := c.Query("employee_id"); queryEmployeeID != "" {
		if parsed, err := uuid.Parse(queryEmployeeID); err == nil {
			targetEmployeeID = parsed
		}
	}

	// Parse multipart form (max 32MB)
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

	// 2. Check for single file with key "image" or "file" if "images" was not provided
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

	embedding, err := h.employeeService.EnrollFace(c.Request.Context(), tenantID, targetEmployeeID, uploadedFiles)
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
		"employee_id":          targetEmployeeID.String(),
		"samples_processed":    len(uploadedFiles),
		"embedding_dimensions": len(embedding),
	})
}

func (h *EmployeeHandler) GetProfile(c *gin.Context) {
	tenantIDVal, _ := c.Get(middleware.CtxKeyTenantID)
	tenantID := tenantIDVal.(uuid.UUID)

	employeeIDVal, _ := c.Get(middleware.CtxKeyEmployeeID)
	employeeID := employeeIDVal.(uuid.UUID)

	emp, err := h.employeeService.GetEmployee(c.Request.Context(), tenantID, employeeID)
	if err != nil || emp == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Employee not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"employee": gin.H{
			"id":                 emp.ID.String(),
			"tenant_id":          emp.TenantID.String(),
			"office_id":          emp.OfficeID,
			"name":               emp.Name,
			"email":              emp.Email,
			"employee_code":      emp.EmployeeCode,
			"is_active":          emp.IsActive,
			"is_enrolled":        emp.FaceEmbedding != nil,
			"face_registered_at": emp.FaceRegisteredAt,
		},
	})
}
