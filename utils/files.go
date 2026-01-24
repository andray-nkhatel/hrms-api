package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"hrms-api/config"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Allowed file extensions for documents
var AllowedDocumentExtensions = map[string]bool{
	".pdf":  true,
	".doc":  true,
	".docx": true,
	".xls":  true,
	".xlsx": true,
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".txt":  true,
	".csv":  true,
}

// Allowed MIME types for documents
var AllowedMimeTypes = map[string]bool{
	"application/pdf":    true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/vnd.ms-excel": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
	"image/jpeg": true,
	"image/png":  true,
	"text/plain": true,
	"text/csv":   true,
}

// ValidateFileExtension checks if the file extension is allowed
func ValidateFileExtension(filename string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	if !AllowedDocumentExtensions[ext] {
		return fmt.Errorf("file extension %s is not allowed. Allowed extensions: pdf, doc, docx, xls, xlsx, jpg, jpeg, png, txt, csv", ext)
	}
	return nil
}

// ValidateMimeType checks if the MIME type is allowed
func ValidateMimeType(mimeType string) error {
	// Remove charset if present (e.g., "text/plain; charset=utf-8" -> "text/plain")
	mimeType = strings.Split(mimeType, ";")[0]
	mimeType = strings.TrimSpace(mimeType)

	if !AllowedMimeTypes[mimeType] {
		return fmt.Errorf("MIME type %s is not allowed", mimeType)
	}
	return nil
}

// ValidateFileSize checks if the file size is within limits
func ValidateFileSize(size int64) error {
	if size > config.AppConfig.MaxFileSize {
		maxMB := config.AppConfig.MaxFileSize / (1024 * 1024)
		return fmt.Errorf("file size exceeds maximum allowed size of %d MB", maxMB)
	}
	if size == 0 {
		return fmt.Errorf("file is empty")
	}
	return nil
}

// GenerateSecureFileName generates a secure, unique filename
func GenerateSecureFileName(originalFilename string, employeeID uint) (string, error) {
	// Generate random bytes
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random filename: %w", err)
	}

	// Create hex string from random bytes
	randomHex := hex.EncodeToString(randomBytes)

	// Get original extension
	ext := filepath.Ext(originalFilename)

	// Create filename: employeeID_timestamp_randomhex.ext
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("%d_%d_%s%s", employeeID, timestamp, randomHex, ext)

	return filename, nil
}

// SaveFile saves an uploaded file to the documents directory
func SaveFile(file io.Reader, filename string, employeeID uint) (string, int64, error) {
	// Ensure documents directory exists
	documentsDir := config.AppConfig.DocumentsPath
	if err := os.MkdirAll(documentsDir, 0755); err != nil {
		return "", 0, fmt.Errorf("failed to create documents directory: %w", err)
	}

	// Create employee-specific subdirectory
	employeeDir := filepath.Join(documentsDir, fmt.Sprintf("employee_%d", employeeID))
	if err := os.MkdirAll(employeeDir, 0755); err != nil {
		return "", 0, fmt.Errorf("failed to create employee directory: %w", err)
	}

	// Full file path
	filePath := filepath.Join(employeeDir, filename)

	// Create the file
	dst, err := os.Create(filePath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	// Copy file content
	size, err := io.Copy(dst, file)
	if err != nil {
		os.Remove(filePath) // Clean up on error
		return "", 0, fmt.Errorf("failed to save file: %w", err)
	}

	// Return relative path from documents directory
	relativePath := filepath.Join(fmt.Sprintf("employee_%d", employeeID), filename)
	return relativePath, size, nil
}

// GetFileMimeType detects MIME type from file extension
func GetFileMimeType(filename string) string {
	ext := filepath.Ext(filename)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		return "application/octet-stream"
	}
	return mimeType
}

// FileExists checks if a file exists
func FileExists(filePath string) bool {
	fullPath := filepath.Join(config.AppConfig.DocumentsPath, filePath)
	_, err := os.Stat(fullPath)
	return !os.IsNotExist(err)
}

// GetFullFilePath returns the full file system path for a document
func GetFullFilePath(relativePath string) string {
	return filepath.Join(config.AppConfig.DocumentsPath, relativePath)
}

// DeleteFile deletes a file from the documents directory
func DeleteFile(relativePath string) error {
	fullPath := GetFullFilePath(relativePath)
	return os.Remove(fullPath)
}

// Allowed file extensions for leave forms (PNG and PDF only)
var AllowedLeaveFormExtensions = map[string]bool{
	".pdf": true,
	".png": true,
}

// Allowed MIME types for leave forms
var AllowedLeaveFormMimeTypes = map[string]bool{
	"application/pdf": true,
	"image/png":       true,
}

// ValidateLeaveFormFileExtension checks if the file extension is allowed for leave forms (PNG/PDF only)
func ValidateLeaveFormFileExtension(filename string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	if !AllowedLeaveFormExtensions[ext] {
		return fmt.Errorf("file extension %s is not allowed. Only PNG and PDF files are allowed for leave forms", ext)
	}
	return nil
}

// ValidateLeaveFormMimeType checks if the MIME type is allowed for leave forms
func ValidateLeaveFormMimeType(mimeType string) error {
	// Remove charset if present
	mimeType = strings.Split(mimeType, ";")[0]
	mimeType = strings.TrimSpace(mimeType)

	if !AllowedLeaveFormMimeTypes[mimeType] {
		return fmt.Errorf("MIME type %s is not allowed. Only PNG and PDF files are allowed for leave forms", mimeType)
	}
	return nil
}

// SaveLeaveFormFile saves an uploaded leave form file to the leave forms directory
func SaveLeaveFormFile(file io.Reader, filename string, employeeID uint, leaveID uint) (string, int64, error) {
	// Ensure leave forms directory exists
	leaveFormsDir := filepath.Join(config.AppConfig.DocumentsPath, "leave_forms")
	if err := os.MkdirAll(leaveFormsDir, 0755); err != nil {
		return "", 0, fmt.Errorf("failed to create leave forms directory: %w", err)
	}

	// Create employee-specific subdirectory
	employeeDir := filepath.Join(leaveFormsDir, fmt.Sprintf("employee_%d", employeeID))
	if err := os.MkdirAll(employeeDir, 0755); err != nil {
		return "", 0, fmt.Errorf("failed to create employee directory: %w", err)
	}

	// Full file path
	filePath := filepath.Join(employeeDir, filename)

	// Create the file
	dst, err := os.Create(filePath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	// Copy file content
	size, err := io.Copy(dst, file)
	if err != nil {
		os.Remove(filePath) // Clean up on error
		return "", 0, fmt.Errorf("failed to save file: %w", err)
	}

	// Return relative path from leave_forms directory
	relativePath := filepath.Join("leave_forms", fmt.Sprintf("employee_%d", employeeID), filename)
	return relativePath, size, nil
}

// GetLeaveFormFilePath returns the full file system path for a leave form
func GetLeaveFormFilePath(relativePath string) string {
	return filepath.Join(config.AppConfig.DocumentsPath, relativePath)
}

// DeleteLeaveFormFile deletes a leave form file
func DeleteLeaveFormFile(relativePath string) error {
	fullPath := GetLeaveFormFilePath(relativePath)
	return os.Remove(fullPath)
}
