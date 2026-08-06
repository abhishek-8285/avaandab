package service

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"

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
var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
}

// Allowed document types for uploads.
var allowedDocTypes = map[string]bool{
	"application/pdf": true,
	"image/jpeg":      true,
	"image/png":       true,
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
	defer file.Close()

	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	filename := uuid.NewString() + ext

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

	// Save file to disk
	uploadPath := filepath.Join(s.cfg.UploadDir, subdir)
	_ = uploadPath

	f := domain.File{
		ID:             domain.FileID(generateID()),
		Filename:       filename,
		OriginalName:   header.Filename,
		Path:           filepath.Join(subdir, filename),
		Size:           header.Size,
		MimeType:       detectMimeType(header),
		UploadableType: uploadableType,
		UploadableID:   &uploadableID,
	}

	// In V1, we store metadata in DB and save files to disk
	// The actual file save would go here
	created, err := s.store.CreateFile(ctx, f)
	if err != nil {
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

	// Delete from disk (placeholder for V1)
	_ = f

	// Delete from database
	if err := s.store.DeleteFile(ctx, id); err != nil {
		return err
	}

	s.log.Info("file deleted", "file_id", id)
	return nil
}

// detectMimeType guesses the MIME type from the filename extension.
func detectMimeType(header *multipart.FileHeader) string {
	ext := filepath.Ext(header.Filename)
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}
