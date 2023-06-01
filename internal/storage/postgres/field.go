package postgres

import (
	"github.com/dipdup-io/starknet-id/internal/storage"
	"github.com/dipdup-net/go-lib/database"
	"github.com/dipdup-net/indexer-sdk/pkg/storage/postgres"
)

// Field -
type Field struct {
	*postgres.Table[*storage.Field]
}

// NewField -
func NewField(db *database.PgGo) *Field {
	return &Field{
		Table: postgres.NewTable[*storage.Field](db),
	}
}
