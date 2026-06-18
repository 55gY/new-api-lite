//go:build !noembed

package main

import (
	"embed"

	"github.com/55gY/new-api-lite/common"
	"github.com/55gY/new-api-lite/router"
)

//go:embed web/classic/dist
var classicBuildFS embed.FS

//go:embed web/classic/dist/index.html
var classicIndexPage []byte

func LoadThemeAssets() (router.ThemeAssets, error) {
	return router.ThemeAssets{
		ClassicFileSystem: common.EmbedFolder(classicBuildFS, "web/classic/dist"),
		ClassicIndexPage:  classicIndexPage,
	}, nil
}
