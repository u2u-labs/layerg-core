package server

import (
	"os"
	"path/filepath"
)

// Open a file relative to the runtime path.
func FileRead(rootPath, relPath string) (*os.File, error) {
	path := filepath.Join(rootPath, relPath)

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return f, nil
}
