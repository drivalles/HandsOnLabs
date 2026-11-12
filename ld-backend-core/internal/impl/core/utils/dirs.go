package ldutils

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

func ResetDirectory(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		return fmt.Errorf("could not remove directory: %v", err)
	}

	err = os.MkdirAll(path, 0755)
	if err != nil {
		return fmt.Errorf("could not create directory: %v", err)
	}
	return nil
}

func CreateParent(path string) error {
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return fmt.Errorf("could not create directory: %v", err)
	}
	return nil
}

func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	if err == nil {
		return true // File exists
	}
	if errors.Is(err, os.ErrNotExist) {
		return false // File explicitly does not exist
	}
	// File may or may not exist (e.g., permission denied)
	return false
}

func RemoveFilesBefore(targetDir string, cutoff time.Time) error {
	files, err := os.ReadDir(targetDir)
	if err != nil {
		return fmt.Errorf("could not read directory: %v", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		info, err := file.Info()
		if err != nil {
			slog.Error(fmt.Sprintf("Could not get info for %s: %v", file.Name(), err))
			continue
		}

		if info.ModTime().Before(cutoff) {
			filePath := filepath.Join(targetDir, file.Name())

			err = os.Remove(filePath)
			if err != nil {
				slog.Error(fmt.Sprintf("failed to delete %s: %v", file.Name(), err))
			} else {
				slog.Info(fmt.Sprintf("successfully deleted: %s (Modified: %v)", file.Name(), info.ModTime()))
			}
		}
	}
	return nil
}

func CreatePaths(basePath string, paths ...string) []string {
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		result = append(result, filepath.Join(basePath, path))
	}
	return result
}
