package repositories_json

import (
	"os"
	"path/filepath"

	errors "github.com/drujensen/aiagent/internal/domain/errs"
)

// atomicWriteFile writes data to path by first writing to a temp file in the
// same directory, then renaming it into place. os.Rename is atomic on both
// POSIX and Windows, so a crash mid-write can never leave path truncated or
// partially written - a reader either sees the old complete file or the new
// complete file, never a mix.
func atomicWriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return errors.InternalErrorf("failed to create temp file for %s: %v", path, err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return errors.InternalErrorf("failed to write temp file for %s: %v", path, err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return errors.InternalErrorf("failed to close temp file for %s: %v", path, err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return errors.InternalErrorf("failed to rename temp file into place for %s: %v", path, err)
	}

	return nil
}
