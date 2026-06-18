package common

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-contrib/static"
)

// Credit: https://github.com/gin-contrib/static/issues/19

type embedFileSystem struct {
	http.FileSystem
}

func (e *embedFileSystem) Exists(prefix string, path string) bool {
	_, err := e.Open(path)
	if err != nil {
		return false
	}
	return true
}

func (e *embedFileSystem) Open(name string) (http.File, error) {
	if name == "/" {
		// This will make sure the index page goes to NoRouter handler,
		// which will use the replaced index bytes with analytic codes.
		return nil, os.ErrNotExist
	}
	return e.FileSystem.Open(name)
}

func EmbedFolder(fsEmbed embed.FS, targetPath string) static.ServeFileSystem {
	efs, err := fs.Sub(fsEmbed, targetPath)
	if err != nil {
		panic(err)
	}
	return &embedFileSystem{
		FileSystem: http.FS(efs),
	}
}

type safeLocalFileSystem struct {
	root string
}

func (s safeLocalFileSystem) Open(name string) (http.File, error) {
	cleanName := strings.TrimPrefix(path.Clean("/"+name), "/")
	fullPath := filepath.Join(s.root, filepath.FromSlash(cleanName))
	resolvedPath, err := filepath.EvalSymlinks(fullPath)
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(s.root, resolvedPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return nil, os.ErrPermission
	}
	return os.Open(resolvedPath)
}

func LocalFolder(root string) (static.ServeFileSystem, error) {
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	return &embedFileSystem{
		FileSystem: safeLocalFileSystem{root: resolvedRoot},
	}, nil
}
