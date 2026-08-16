package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"transport-app/internal/domain"
)

// FileService handles file uploads and storage.
type FileService struct {
	baseService
}

// UploadResult contains information about an uploaded file.
type UploadResult struct {
	File domain.File
}

// Allowed image types for uploads.
var AllowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// Allowed document types for uploads.
var AllowedDocTypes = map[string]bool{
	"application/pdf": true,
	"image/jpeg":      true,
	"image/png":       true,
	"image/webp":      true,
}

// allowedUploadContentTypes are the content types accepted by UploadFile,
// validated against the file's magic bytes.
var allowedUploadContentTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/gif":       true,
	"image/webp":      true,
	"application/pdf": true,
}

// UploadFile saves an uploaded file to disk and creates a database record.
func (s *FileService) UploadFile(ctx context.Context, header *multipart.FileHeader, uploadableType string, uploadableID string) (domain.File, error) {
	if header == nil {
		return domain.File{}, fmt.Errorf("no file provided")
	}

	file, err := header.Open()
	if err != nil {
		return domain.File{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Validate content by magic bytes from the first 512 bytes
	buf := make([]byte, 512)
	n, err := io.ReadFull(file, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return domain.File{}, fmt.Errorf("failed to read file: %w", err)
	}
	if n == 0 {
		return domain.File{}, fmt.Errorf("empty file")
	}
	detected := http.DetectContentType(buf[:n])
	if !allowedUploadContentTypes[detected] {
		return domain.File{}, fmt.Errorf("unsupported file type %q", detected)
	}
	if declared := header.Header.Get("Content-Type"); declared != "" && declared != "application/octet-stream" && declared != detected {
		return domain.File{}, fmt.Errorf("declared content type %q does not match file content %q", declared, detected)
	}

	// Generate a safe unique filename: random ID + sanitized extension
	filename := uuid.NewString() + safeExtension(header.Filename, detected)

	// Determine upload subdirectory
	var subdir string
	switch uploadableType {
	case "driver_license":
		subdir = "drivers"
	case "vehicle_insurance", "vehicle_permit":
		subdir = "vehicles"
	case "company_logo":
		subdir = "company"
	default:
		subdir = "misc"
	}

	dir := filepath.Join(uploadDir(s), subdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return domain.File{}, fmt.Errorf("failed to create upload directory: %w", err)
	}

	path := filepath.Join(dir, filename)
	dest, err := os.Create(path)
	if err != nil {
		return domain.File{}, fmt.Errorf("failed to create file: %w", err)
	}
	if _, err := io.Copy(dest, io.MultiReader(bytes.NewReader(buf[:n]), file)); err != nil {
		_ = dest.Close()
		_ = os.Remove(path)
		return domain.File{}, fmt.Errorf("failed to write file: %w", err)
	}
	if err := dest.Close(); err != nil {
		_ = os.Remove(path)
		return domain.File{}, fmt.Errorf("failed to close file: %w", err)
	}

	f := domain.File{
		ID:             domain.FileID(generateID()),
		Filename:       filename,
		OriginalName:   header.Filename,
		Path:           filepath.Join(subdir, filename),
		Size:           header.Size,
		MimeType:       detected,
		UploadableType: uploadableType,
		UploadableID:   &uploadableID,
	}

	created, err := s.store.CreateFile(ctx, f)
	if err != nil {
		_ = os.Remove(path)
		return domain.File{}, err
	}

	s.log.Info("file uploaded", "file_id", created.ID, "type", uploadableType)
	return created, nil
}

// GetFile retrieves a file by ID.
func (s *FileService) GetFile(ctx context.Context, id domain.FileID) (domain.File, error) {
	return s.store.GetFileByID(ctx, id)
}

// GetFilesByEntity retrieves files for a specific entity.
func (s *FileService) GetFilesByEntity(ctx context.Context, uploadableType string, uploadableID string) ([]domain.File, error) {
	return s.store.GetFilesByUploadable(ctx, uploadableType, uploadableID)
}

// DeleteFile removes a file from storage and database.
func (s *FileService) DeleteFile(ctx context.Context, id domain.FileID) error {
	f, err := s.store.GetFileByID(ctx, id)
	if err != nil {
		return err
	}

	// Delete from disk
	if f.Path != "" {
		path := filepath.Join(uploadDir(s), f.Path)
		_ = os.Remove(path)
	}

	// Delete from database
	if err := s.store.DeleteFile(ctx, id); err != nil {
		return err
	}

	s.log.Info("file deleted", "file_id", id)
	return nil
}

// uploadDir returns the configured uploads directory, defaulting to ./uploads.
func uploadDir(s *FileService) string {
	if s.cfg != nil && s.cfg.UploadDir != "" {
		return s.cfg.UploadDir
	}
	return "./uploads"
}

// safeExtension returns a whitelisted lowercase extension for the stored file,
// derived from the original filename; falls back to the extension matching the
// detected content type. Never includes path separators or "..".
func safeExtension(filename, contentType string) string {
	ext := strings.ToLower(filepath.Ext(filepath.Base(filepath.Clean(filename))))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".pdf":
		if ext == ".jpeg" {
			return ".jpg"
		}
		return ext
	}
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "application/pdf":
		return ".pdf"
	default:
		return ".bin"
	}
}
