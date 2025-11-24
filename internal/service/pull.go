package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"

	"github.com/beanbocchi/templar/internal/db"
	"github.com/beanbocchi/templar/internal/model"
)

type PullParams struct {
	TemplateID uuid.UUID `validate:"required,uuid"`
	Version    int64     `validate:"required,min=1"`
}

func (s *Service) Pull(ctx context.Context, params PullParams) (io.ReadCloser, error) {
	key := getKey(params.TemplateID, params.Version)

	// Check if the template version exists, if not return an error
	if _, err := s.storage.GetTemplateVersion(ctx, db.GetTemplateVersionParams{
		TemplateID:    params.TemplateID.String(),
		VersionNumber: params.Version,
	}); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.NewError("template_version.not_found", "Template %s version %d not found").Fmt(params.TemplateID.String(), params.Version)
		}
		return nil, fmt.Errorf("get template version: %w", err)
	}

	reader, err := s.objectStore.Download(ctx, key)
	if err != nil {
		return nil, model.NewError("object_store.get", "Failed to get object from object store: %w").Fmt(err)
	}

	return reader, nil
}
