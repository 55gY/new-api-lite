package middleware

import (
	"github.com/55gY/new-api-lite/common"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	config := cors.DefaultConfig()
	// 允许任意来源。注意：CORS 规范禁止 Access-Control-Allow-Origin: * 与
	// Access-Control-Allow-Credentials: true 同时出现（浏览器会拒绝携带凭证的跨域响应），
	// 且带凭证时 Access-Control-Allow-Headers: * 也不被当作通配符。
	// 本网关的鉴权走 Authorization 头/API Token，管理端在默认部署下与后端同源（前端内嵌），
	// 无需跨域携带 Cookie，因此这里不启用 AllowCredentials，从而使通配来源/头部保持合法有效。
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"*"}
	return cors.New(config)
}

func PoweredBy() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-New-Api-Version", common.Version)
		c.Next()
	}
}
