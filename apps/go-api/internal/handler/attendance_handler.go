package handler

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/faceattendance/go-api/internal/middleware"
	"github.com/faceattendance/go-api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AttendanceHandler struct {
	attendanceService service.AttendanceService
}

func NewAttendanceHandler(attendanceService service.AttendanceService) *AttendanceHandler {
	return &AttendanceHandler{attendanceService: attendanceService}
}

func (h *AttendanceHandler) CheckIn(c *gin.Context) {
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
	employeeID := employeeIDVal.(uuid.UUID)

	// Read selfie photo
	fileHeader, err := c.FormFile("image")
	if err != nil {
		// try fallback key "photo" or "file"
		fileHeader, err = c.FormFile("photo")
		if err != nil {
			fileHeader, err = c.FormFile("file")
		}
	}
	if fileHeader == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Selfie image file is required (multipart field 'image')",
		})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Failed to read image file: " + err.Error()})
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Failed to read image content: " + err.Error()})
		return
	}

	// Read latitude
	latStr := c.PostForm("latitude")
	if latStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Field 'latitude' is required"})
		return
	}
	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid 'latitude' value"})
		return
	}

	// Read longitude
	lngStr := c.PostForm("longitude")
	if lngStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Field 'longitude' is required"})
		return
	}
	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid 'longitude' value"})
		return
	}

	// Read device_id
	deviceID := c.PostForm("device_id")
	if deviceID == "" {
		deviceID = "unknown-device"
	}

	// Read attendance_type
	attendanceType := c.PostForm("attendance_type")
	if attendanceType == "" {
		attendanceType = "IN"
	}

	checkInReq := service.CheckInRequest{
		TenantID:       tenantID,
		EmployeeID:     employeeID,
		ImageFilename:  fileHeader.Filename,
		ImageData:      fileBytes,
		Latitude:       lat,
		Longitude:      lng,
		DeviceID:       deviceID,
		AttendanceType: attendanceType,
	}

	result, err := h.attendanceService.CheckIn(c.Request.Context(), checkInReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	statusCode := http.StatusOK
	if !result.IsValid {
		statusCode = http.StatusBadRequest
	}

	c.JSON(statusCode, gin.H{
		"success": result.IsValid,
		"message": map[bool]string{
			true:  "Attendance recorded and verified successfully",
			false: "Attendance rejected: verification criteria not met",
		}[result.IsValid],
		"data": gin.H{
			"attendance_id":    result.AttendanceLog.ID.String(),
			"clock_time":       result.AttendanceLog.ClockTime,
			"attendance_type":  result.AttendanceLog.AttendanceType,
			"similarity_score": result.SimilarityScore,
			"distance_meters":  result.DistanceMeters,
			"allowed_radius":   result.AllowedRadius,
			"is_valid":         result.IsValid,
			"validation_notes": result.ValidationNotes,
		},
	})
}

func (h *AttendanceHandler) GetHistory(c *gin.Context) {
	employeeIDVal, exists := c.Get(middleware.CtxKeyEmployeeID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Unauthorized"})
		return
	}
	employeeID := employeeIDVal.(uuid.UUID)

	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}

	logs, err := h.attendanceService.GetAttendanceHistory(c.Request.Context(), employeeID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count":   len(logs),
		"data":    logs,
	})
}

func (h *AttendanceHandler) GetLogs(c *gin.Context) {
	tenantIDVal, exists := c.Get(middleware.CtxKeyTenantID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Unauthorized: missing tenant context"})
		return
	}
	tenantID := tenantIDVal.(uuid.UUID)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	var startDate *time.Time
	if startStr := c.Query("start_date"); startStr != "" {
		if t, err := time.Parse("2006-01-02", startStr); err == nil {
			startDate = &t
		}
	}

	var endDate *time.Time
	if endStr := c.Query("end_date"); endStr != "" {
		if t, err := time.Parse("2006-01-02", endStr); err == nil {
			endDay := t.Add(24*time.Hour - time.Nanosecond)
			endDate = &endDay
		}
	}

	var userUUID *uuid.UUID
	if userStr := c.Query("user_id"); userStr != "" {
		if parsed, err := uuid.Parse(userStr); err == nil {
			userUUID = &parsed
		}
	}

	var isValid *bool
	if validStr := c.Query("is_valid"); validStr != "" {
		if val, err := strconv.ParseBool(validStr); err == nil {
			isValid = &val
		}
	}

	result, err := h.attendanceService.ListLogs(c.Request.Context(), tenantID, page, limit, startDate, endDate, userUUID, isValid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"total":   result.Total,
		"page":    result.Page,
		"limit":   result.Limit,
		"data":    result.Logs,
	})
}

func (h *AttendanceHandler) GetSummary(c *gin.Context) {
	tenantIDVal, exists := c.Get(middleware.CtxKeyTenantID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Unauthorized: missing tenant context"})
		return
	}
	tenantID := tenantIDVal.(uuid.UUID)

	targetDate := time.Now()
	if dateStr := c.Query("date"); dateStr != "" {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			targetDate = t
		}
	}

	summary, err := h.attendanceService.GetSummary(c.Request.Context(), tenantID, targetDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    summary,
	})
}

