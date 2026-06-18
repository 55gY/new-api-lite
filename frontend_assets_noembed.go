//go:build noembed

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/55gY/new-api-lite/common"
	"github.com/55gY/new-api-lite/router"
)

const webDistDirEnv = "WEB_DIST_DIR"

func LoadThemeAssets() (router.ThemeAssets, error) {
	distDir, err := resolveWebDistDir()
	if err != nil {
		return router.ThemeAssets{}, err
	}
	indexPath := filepath.Join(distDir, "index.html")
	indexPage, err := os.ReadFile(indexPath)
	if err != nil {
		return router.ThemeAssets{}, fmt.Errorf("read frontend index.html: %w", err)
	}
	if len(indexPage) == 0 {
		return router.ThemeAssets{}, fmt.Errorf("frontend index.html is empty: %s", indexPath)
	}
	fileSystem, err := common.LocalFolder(distDir)
	if err != nil {
		return router.ThemeAssets{}, fmt.Errorf("open frontend dist directory: %w", err)
	}
	return router.ThemeAssets{
		ClassicFileSystem: fileSystem,
		ClassicIndexPage:  indexPage,
	}, nil
}

func resolveWebDistDir() (string, error) {
	distDir := os.Getenv(webDistDirEnv)
	if distDir == "" {
		exePath, err := os.Executable()
		if err != nil {
			return "", fmt.Errorf("resolve executable path: %w", err)
		}
		distDir = filepath.Join(filepath.Dir(exePath), "web", "classic", "dist")
	}
	absDir, err := filepath.Abs(filepath.Clean(distDir))
	if err != nil {
		return "", fmt.Errorf("resolve frontend dist path: %w", err)
	}
	info, err := os.Stat(absDir)
	if err != nil {
		return "", fmt.Errorf("frontend dist directory is not accessible: %s: %w", absDir, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("frontend dist path is not a directory: %s", absDir)
	}
	indexPath := filepath.Join(absDir, "index.html")
	if info, err := os.Stat(indexPath); err != nil {
		return "", fmt.Errorf("frontend index.html is not accessible: %s: %w", indexPath, err)
	} else if info.IsDir() {
		return "", fmt.Errorf("frontend index.html is a directory: %s", indexPath)
	}
	return absDir, nil
}
