package postgres

import (
	"github.com/dipdup-io/starknet-id/internal/storage"
	"github.com/dipdup-net/go-lib/database"
	"github.com/dipdup-net/indexer-sdk/pkg/storage/postgres"
)

// StarknetId -
type StarknetId struct {
	*postgres.Table[*storage.StarknetId]
}

// NewStarknetId -
func NewStarknetId(db *database.PgGo) *StarknetId {
	return &StarknetId{
		Table: postgres.NewTable[*storage.StarknetId](db),
	}
}
