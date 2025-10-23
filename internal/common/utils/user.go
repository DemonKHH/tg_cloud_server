package utils

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
)

// getUserID 从gin.Context中获取用户ID
func GetUserID(c *gin.Context) (uint64, error) {
	// 从JWT token中获取用户ID，通常在middleware中设置
	userIDStr, exists := c.Get("user_id")
	if !exists {
		return 0, fmt.Errorf("user not authenticated")
	}

	// 尝试直接转换为uint64
	if userID, ok := userIDStr.(uint64); ok {
		return userID, nil
	}

	// 尝试从string转换
	if userIDString, ok := userIDStr.(string); ok {
		userID, err := strconv.ParseUint(userIDString, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid user ID format: %w", err)
		}
		return userID, nil
	}

	// 尝试从float64转换（JSON numbers通常是float64）
	if userIDFloat, ok := userIDStr.(float64); ok {
		return uint64(userIDFloat), nil
	}

	return 0, fmt.Errorf("user ID has unexpected type: %T", userIDStr)
}
