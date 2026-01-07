package middleware

import (
	pkg "CMS/internal/pkg/jwt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Cors 跨域中间件（前后端分离必备）
func Cors() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // 生产环境替换为前端域名（如http://localhost:8081）
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}

func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 从请求头中获取 Authorization 字段
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "请求头中缺少 Authorization"})
			c.Abort() // 终止请求，不再执行后续处理
			return
		}

		// 2. 检查 Authorization 格式是否为 "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization 格式错误（应为 Bearer <token>）"})
			c.Abort()
			return
		}

		// 3. 解析并验证 Token
		tokenString := parts[1]
		// 解析 Token 并指定验证后的载荷类型
		token, err := jwt.ParseWithClaims(
			tokenString,
			&pkg.UserClaims{}, // 与生成时的载荷结构一致
			func(token *jwt.Token) (interface{}, error) {
				// 验证签名算法是否为预期的 HS256
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte("aiuegfiuewgfiuwfeiuwheqowhfoiqfiifenfeqnfeq"), nil // 使用相同的密钥验证签名
			},
		)

		// 4. 处理验证错误
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的 Token：" + err.Error()})
			c.Abort()
			return
		}

		// 5. 验证通过，提取用户信息并存入上下文
		if claims, ok := token.Claims.(*pkg.UserClaims); ok && token.Valid {
			// 将 sid 存入上下文，后续接口可通过 c.Get("sid") 获取
			c.Set("uuid", claims.UserID)
			println(claims.UserID)
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token 验证失败"})
			c.Abort()
			return
		}
		// 继续执行后续的接口处理函数
		c.Next()
	}
}
