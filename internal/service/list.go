package service

import (
	"context"
	"fmt"

	"github.com/guregu/null/v6"

	"github.com/beanbocchi/templar/internal/db"
	"github.com/beanbocchi/templar/internal/model"
)

type ListTemplateParams struct {
	Search null.String `validate:"omitempty,min=1"`
}

func (s *Service) ListTemplate(ctx context.Context, params ListTemplateParams) ([]db.Template, error) {
	templates, err := s.storage.ListTemplates(ctx, db.ListTemplatesParams{
		Limit:  100,
		Offset: 0,
	})
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}

	return templates, nil
}

type ListVersionsParams struct {
	TemplateID string `validate:"required,uuid"`
}

func (s *Service) ListVersions(ctx context.Context, params ListVersionsParams) ([]db.TemplateVersion, error) {
	versions, err := s.storage.ListTemplateVersions(ctx, params.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("list template versions: %w", err)
	}

	return versions, nil
}

type GetTemplateVersionParams struct {
	TemplateID string `validate:"required,uuid"`
	Version    int64  `validate:"required,min=1"`
}

func (s *Service) GetTemplateVersion(ctx context.Context, params GetTemplateVersionParams) (db.TemplateVersion, error) {
	version, err := s.storage.GetTemplateVersion(ctx, db.GetTemplateVersionParams{
		TemplateID:    params.TemplateID,
		VersionNumber: params.Version,
	})
	if err != nil {
		return db.TemplateVersion{}, fmt.Errorf("get template version: %w", err)
	}

	return version, nil
}

type ListJobsParams struct {
	model.PaginationParams
}

func (s *Service) ListJobs(ctx context.Context, params ListJobsParams) ([]db.Job, error) {
	jobs, err := s.storage.ListJobs(ctx, db.ListJobsParams{
		Limit:  int64(params.Limit),
		Offset: int64(params.Offset()),
	})
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}

	return jobs, nil
}
