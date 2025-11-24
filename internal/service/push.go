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
	"github.com/beanbocchi/templar/internal/utils/progress"
)

type PushParams struct {
	TemplateID uuid.UUID `validate:"required,uuid"`
	Version    int64     `validate:"required,min=1"`
	File       *multipart.FileHeader
}

func (s *Service) Push(ctx context.Context, params PushParams) error {
	tx, err := s.storage.BeginTx()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check if the template exists, if not create it
	if _, err := tx.GetTemplate(ctx, params.TemplateID.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if _, err := tx.CreateTemplate(ctx, db.CreateTemplateParams{
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
	if _, err := tx.GetTemplateVersion(ctx, db.GetTemplateVersionParams{
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

	metadata, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	// Create a job to push the template
	job, err := tx.CreateJob(ctx, db.CreateJobParams{
		Type:          "template.push",
		TemplateID:    params.TemplateID.String(),
		VersionNumber: ptr.Int64(params.Version),
		Status:        "pending",
		Progress:      0,
		StartedAt:     time.Now(),
		Metadata:      string(metadata),
	})
	if err != nil {
		return fmt.Errorf("create job: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	// Background upload the file to the object store
	jobFn := func() {
		fmt.Printf("Pushing template: %s\n", params.TemplateID.String())
		ctx := context.Background()
		key := getKey(params.TemplateID, params.Version)

		src, err := params.File.Open()
		if err != nil {
			s.storage.UpdateJob(ctx, db.UpdateJobParams{
				ID:           job.ID,
				Status:       ptr.String("error"),
				ErrorMessage: ptr.String(err.Error()),
				CompletedAt:  ptr.Time(time.Now()),
			})
			return
		}
		defer src.Close()

		// Compute hash while uploading using TeeReader
		hasher := blake3.New()
		hashReader := io.TeeReader(src, hasher)
		progressReader := progress.NewReader(hashReader, params.File.Size)

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
					fmt.Printf("Uploading template: %f\n", progress)
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
			return
		}

		// Create template version with computed hash
		hashStr := hex.EncodeToString(hasher.Sum(nil))
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
			return
		}

		// Mark job as completed
		s.storage.UpdateJob(ctx, db.UpdateJobParams{
			ID:          job.ID,
			Status:      ptr.String("completed"),
			CompletedAt: ptr.Time(time.Now()),
		})
	}

	s.jobs <- jobFn

	return nil
}
