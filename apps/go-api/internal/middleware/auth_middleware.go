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
	CtxKeyEmployeeID = "employee_id"
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
		c.Set(CtxKeyEmployeeID, claims.EmployeeID)
		c.Next()
	}
}
