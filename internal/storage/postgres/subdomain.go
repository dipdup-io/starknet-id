package postgres

import (
	"context"

	"github.com/dipdup-io/starknet-id/internal/storage"
	"github.com/dipdup-net/go-lib/database"
	"github.com/dipdup-net/indexer-sdk/pkg/storage/postgres"
)

// Subdomain -
type Subdomain struct {
	*postgres.Table[*storage.Subdomain]
}

// NewSubdomain -
func NewSubdomain(db *database.Bun) *Subdomain {
	return &Subdomain{
		Table: postgres.NewTable[*storage.Subdomain](db),
	}
}

// GetByResolverId -
func (s *Subdomain) GetByResolverId(ctx context.Context, resolverId uint64) (result storage.Subdomain, err error) {
	err = s.DB().NewSelect().Model(&result).
		Where("resolver_id = ?", resolverId).
		Limit(1).Scan(ctx)
	return
}
