package middleware

import (
	"net/http"
	"strings"

	"github.com/faceattendance/go-api/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	CtxKeyClaims     = "jwt_claims"
	CtxKeyTenantID   = "tenant_id"
	CtxKeyUserID     = "user_id"
	CtxKeyEmployeeID = "employee_id"
	CtxKeyRole       = "user_role"
)

func AuthMiddleware(authService service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Authorization header is required",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid authorization format. Format must be 'Bearer <token>'",
			})
			return
		}

		claims, err := authService.ValidateToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid or expired authentication token: " + err.Error(),
			})
			return
		}

		c.Set(CtxKeyClaims, claims)
		c.Set(CtxKeyTenantID, claims.TenantID)
		c.Set(CtxKeyUserID, claims.UserID)
		c.Set(CtxKeyEmployeeID, claims.EmployeeID)
		c.Set(CtxKeyRole, claims.Role)
		c.Next()
	}
}

// RequireSuperAdmin restricts access exclusively to platform super administrators
func RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get(CtxKeyRole)
		if !exists || roleVal != "superadmin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "Forbidden: Requires superadmin role",
			})
			return
		}
		c.Next()
	}
}

// RequireTenantAdmin restricts access to institution / tenant administrators (or superadmin)
func RequireTenantAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get(CtxKeyRole)
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "Forbidden: Missing user role",
			})
			return
		}
		role := roleVal.(string)
		if role != "superadmin" && role != "tenant_admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "Forbidden: Requires tenant_admin or superadmin role",
			})
			return
		}
		c.Next()
	}
}
