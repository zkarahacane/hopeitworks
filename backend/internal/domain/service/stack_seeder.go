package service

import (
	"context"
	"log/slog"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// SeedStacks idempotently applies the stack catalogue from versioned config to the
// database. Each entry is UPSERTed (insert-or-update on key), so the catalogue config
// is the source of truth for image_ref/toolchain and image rebuilds no longer require
// an UPDATE migration. Safe to re-run on every boot.
//
// An empty catalogue is a no-op: it logs a warning and returns nil, leaving whatever
// the migration-inlined seed (000034) put in place. It NEVER deletes stacks not present
// in the catalogue, and it never crashes the boot path on an empty/absent catalogue.
func SeedStacks(ctx context.Context, repo port.StackRepository, catalogue []model.Stack, logger *slog.Logger) error {
	if len(catalogue) == 0 {
		logger.Warn("stack catalogue empty in config, skipping seed (falling back to migration-inlined seed)")
		return nil
	}

	for i := range catalogue {
		s := catalogue[i]
		upserted, err := repo.Upsert(ctx, &s)
		if err != nil {
			return err
		}
		logger.Info("stack catalogue upserted", "key", upserted.Key, "image_ref", upserted.ImageRef)
	}

	logger.Info("stack catalogue seeded", "count", len(catalogue))
	return nil
}
