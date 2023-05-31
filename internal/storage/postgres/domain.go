package postgres

import (
	"github.com/dipdup-io/starknet-id/internal/storage"
	"github.com/dipdup-net/go-lib/database"
	"github.com/dipdup-net/indexer-sdk/pkg/storage/postgres"
)

// Domain -
type Domain struct {
	*postgres.Table[*storage.Domain]
}

// NewDomain -
func NewDomain(db *database.PgGo) *Domain {
	return &Domain{
		Table: postgres.NewTable[*storage.Domain](db),
	}
}
