package router

import (
	"net/http"
	"strings"

	"github.com/55gY/new-api-lite/controller"
	"github.com/55gY/new-api-lite/middleware"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

// ThemeAssets holds the classic frontend assets.
type ThemeAssets struct {
	ClassicFileSystem static.ServeFileSystem
	ClassicIndexPage  []byte
}

func SetWebRouter(router *gin.Engine, assets ThemeAssets) {
	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(middleware.GlobalWebRateLimit())
	router.Use(middleware.Cache())
	router.Use(static.Serve("/", assets.ClassicFileSystem))
	router.NoRoute(func(c *gin.Context) {
		c.Set(middleware.RouteTagKey, "web")
		if strings.HasPrefix(c.Request.RequestURI, "/v1") || strings.HasPrefix(c.Request.RequestURI, "/api") || strings.HasPrefix(c.Request.RequestURI, "/assets") || strings.HasPrefix(c.Request.RequestURI, "/suno") {
			controller.RelayNotFound(c)
			return
		}
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", assets.ClassicIndexPage)
	})
}
