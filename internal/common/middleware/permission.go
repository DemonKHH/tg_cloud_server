package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/models"
)

// RequireRole 要求指定角色的中间件
func RequireRole(roles ...models.UserRole) gin.HandlerFunc {
	log := logger.Get().Named("permission_middleware")

	return func(c *gin.Context) {
		// 获取用户角色
		roleInterface, exists := c.Get("user_role")
		if !exists {
			log.Warn("User role not found in context")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "用户角色信息缺失",
			})
			c.Abort()
			return
		}

		userRole, ok := roleInterface.(models.UserRole)
		if !ok {
			log.Warn("Invalid user role type in context")
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "无效的用户角色",
			})
			c.Abort()
			return
		}

		// 检查角色是否在允许列表中
		hasRole := false
		for _, allowedRole := range roles {
			if userRole == allowedRole {
				hasRole = true
				break
			}
		}

		if !hasRole {
			log.Warn("Insufficient role permissions",
				zap.String("user_role", string(userRole)),
				zap.String("path", c.Request.URL.Path))
			c.JSON(http.StatusForbidden, gin.H{
				"error":          "forbidden",
				"message":        "权限不足，需要更高的角色权限",
				"required_roles": roles,
			})
			c.Abort()
			return
		}

		// 继续处理请求
		c.Next()
	}
}

// RequireAdmin 要求管理员角色的中间件（快捷方法）
func RequireAdmin() gin.HandlerFunc {
	return RequireRole(models.RoleAdmin)
}

// RequirePremium 要求高级用户或管理员的中间件（快捷方法）
func RequirePremium() gin.HandlerFunc {
	return RequireRole(models.RolePremium, models.RoleAdmin)
}

// RequirePermission 要求指定权限的中间件
func RequirePermission(permission string) gin.HandlerFunc {
	log := logger.Get().Named("permission_middleware")

	return func(c *gin.Context) {
		// 获取用户信息
		userProfileInterface, exists := c.Get("user_profile")
		if !exists {
			log.Warn("User profile not found in context")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "用户信息缺失",
			})
			c.Abort()
			return
		}

		userProfile, ok := userProfileInterface.(*models.UserProfile)
		if !ok {
			log.Warn("Invalid user profile type in context")
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "无效的用户信息",
			})
			c.Abort()
			return
		}

		// 构建临时User对象用于权限检查
		user := &models.User{
			ID:       userProfile.ID,
			Role:     userProfile.Role,
			IsActive: userProfile.IsActive,
		}

		// 检查权限
		if !user.HasPermission(permission) {
			log.Warn("Insufficient permissions",
				zap.Uint64("user_id", userProfile.ID),
				zap.String("required_permission", permission),
				zap.String("path", c.Request.URL.Path))
			c.JSON(http.StatusForbidden, gin.H{
				"error":               "forbidden",
				"message":             "权限不足",
				"required_permission": permission,
			})
			c.Abort()
			return
		}

		// 继续处理请求
		c.Next()
	}
}

// RequireAnyPermission 要求任意一个权限的中间件
func RequireAnyPermission(permissions ...string) gin.HandlerFunc {
	log := logger.Get().Named("permission_middleware")

	return func(c *gin.Context) {
		// 获取用户信息
		userProfileInterface, exists := c.Get("user_profile")
		if !exists {
			log.Warn("User profile not found in context")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "用户信息缺失",
			})
			c.Abort()
			return
		}

		userProfile, ok := userProfileInterface.(*models.UserProfile)
		if !ok {
			log.Warn("Invalid user profile type in context")
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "无效的用户信息",
			})
			c.Abort()
			return
		}

		// 构建临时User对象用于权限检查
		user := &models.User{
			ID:       userProfile.ID,
			Role:     userProfile.Role,
			IsActive: userProfile.IsActive,
		}

		// 检查是否拥有任意一个权限
		hasPermission := false
		for _, perm := range permissions {
			if user.HasPermission(perm) {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			log.Warn("Insufficient permissions",
				zap.Uint64("user_id", userProfile.ID),
				zap.Strings("required_permissions", permissions),
				zap.String("path", c.Request.URL.Path))
			c.JSON(http.StatusForbidden, gin.H{
				"error":                "forbidden",
				"message":              "权限不足",
				"required_permissions": permissions,
			})
			c.Abort()
			return
		}

		// 继续处理请求
		c.Next()
	}
}
