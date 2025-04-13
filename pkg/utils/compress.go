package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/alexmullins/zip"
)

// CompressWithPassword compresses a file with password protection
func CompressWithPassword(sourcePath string, password string) (string, error) {
	// Check if source exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return "", fmt.Errorf("source file not found: %s", sourcePath)
	}

	// Create zip file path
	zipPath := sourcePath + ".zip"

	// Create the zip file
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return "", fmt.Errorf("failed to create zip file: %v", err)
	}
	defer zipFile.Close()

	// Create zip writer
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Open source file
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %v", err)
	}
	defer sourceFile.Close()

	// Get base filename
	baseFilename := filepath.Base(sourcePath)

	// Create encrypted entry
	writer, err := zipWriter.Encrypt(baseFilename, password)
	if err != nil {
		return "", fmt.Errorf("failed to create encrypted entry: %v", err)
	}

	// Copy file content
	if _, err := io.Copy(writer, sourceFile); err != nil {
		return "", fmt.Errorf("failed to write to zip: %v", err)
	}

	return zipPath, nil
}
