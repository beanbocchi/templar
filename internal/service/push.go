package service

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/google/uuid"
	"github.com/zeebo/blake3"

	"github.com/beanbocchi/templar/internal/db"
	"github.com/beanbocchi/templar/internal/model"
	"github.com/beanbocchi/templar/internal/utils/progressr"
)

type PushParams struct {
	TemplateID uuid.UUID `validate:"required,uuid"`
	Version    int64     `validate:"required,min=1"`
	File       *multipart.FileHeader
}

func (s *Service) Push(ctx context.Context, params PushParams) error {
	// Create a job to push the template - OUTSIDE transaction to be visible immediately
	job, err := s.storage.CreateJob(ctx, db.CreateJobParams{
		Type:          "template.push",
		TemplateID:    params.TemplateID.String(),
		VersionNumber: ptr.Int64(params.Version),
		Status:        "pending",
		Progress:      0,
		StartedAt:     time.Now(),
	})
	if err != nil {
		return fmt.Errorf("create job: %w", err)
	}

	// Check if the template exists, if not create it
	if _, err := s.storage.GetTemplate(ctx, params.TemplateID.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if _, err := s.storage.CreateTemplate(ctx, db.CreateTemplateParams{
				ID:   params.TemplateID.String(),
				Name: params.TemplateID.String(),
				// Description: sql.NullString{String: params.TemplateID.String(), Valid: true},
			}); err != nil {
				return fmt.Errorf("create template: %w", err)
			}
		} else {
			return fmt.Errorf("get template: %w", err)
		}
	}

	// Check if the template version already exists
	if _, err := s.storage.GetTemplateVersion(ctx, db.GetTemplateVersionParams{
		TemplateID:    params.TemplateID.String(),
		VersionNumber: params.Version,
	}); err != nil {
		// Only return an error if the error is not a no rows error
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("get template version: %w", err)
		}
	} else {
		return model.NewError("template_version.already_exists", "Template %s version %d already exists").Fmt(params.TemplateID.String(), params.Version)
	}

	fmt.Printf("Pushing template: %s\n", params.TemplateID.String())
	key := getKey(params.TemplateID, params.Version)

	src, err := params.File.Open()
	if err != nil {
		s.storage.UpdateJob(ctx, db.UpdateJobParams{
			ID:           job.ID,
			Status:       ptr.String("error"),
			ErrorMessage: ptr.String(err.Error()),
			CompletedAt:  ptr.Time(time.Now()),
		})
		return fmt.Errorf("open file: %w", err)
	}
	defer src.Close()

	// Compute hash while uploading using TeeReader
	hasher := blake3.New()
	hashReader := io.TeeReader(src, hasher)
	progressReader := progressr.NewReader(hashReader, params.File.Size)

	// Monitor progress
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				progress := progressReader.Progress()
				s.storage.UpdateJob(ctx, db.UpdateJobParams{
					ID:       job.ID,
					Status:   ptr.String("uploading"),
					Progress: ptr.Int64(int64(progress * 100)),
				})
				if progress >= 1.0 {
					return
				}
			}
		}
	}()

	// Upload file
	if err := s.objectStore.Upload(ctx, key, progressReader); err != nil {
		s.storage.UpdateJob(ctx, db.UpdateJobParams{
			ID:           job.ID,
			Status:       ptr.String("error"),
			ErrorMessage: ptr.String(err.Error()),
			CompletedAt:  ptr.Time(time.Now()),
		})
		return fmt.Errorf("upload file: %w", err)
	}

	// Create template version with computed hash
	hashStr := hex.EncodeToString(hasher.Sum(nil))
	fmt.Printf("Hash: %s\n", hashStr)
	if _, err := s.storage.CreateTemplateVersion(ctx, db.CreateTemplateVersionParams{
		ID:            uuid.New().String(),
		TemplateID:    params.TemplateID.String(),
		VersionNumber: params.Version,
		ObjectKey:     key,
		FileSize:      ptr.Int64(params.File.Size),
		FileHash:      ptr.String(hashStr),
	}); err != nil {
		s.storage.UpdateJob(ctx, db.UpdateJobParams{
			ID:           job.ID,
			Status:       ptr.String("error"),
			ErrorMessage: ptr.String(err.Error()),
			CompletedAt:  ptr.Time(time.Now()),
		})
		return fmt.Errorf("create template version: %w", err)
	}

	// Mark job as completed
	s.storage.UpdateJob(ctx, db.UpdateJobParams{
		ID:          job.ID,
		Status:      ptr.String("completed"),
		CompletedAt: ptr.Time(time.Now()),
	})

	return nil
}
