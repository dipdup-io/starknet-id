package postgres

import (
	"context"

	models "github.com/dipdup-io/starknet-id/internal/storage"
	"github.com/dipdup-net/indexer-sdk/pkg/storage"
)

// Transaction -
type Transaction struct {
	storage.Transaction
}

// BeginTransaction -
func BeginTransaction(ctx context.Context, tx storage.Transactable) (Transaction, error) {
	t, err := tx.BeginTransaction(ctx)
	return Transaction{t}, err
}

// SaveAddress -
func (t Transaction) SaveState(ctx context.Context, state *models.State) error {
	_, err := t.Tx().NewInsert().Model(state).
		On("CONFLICT (name) DO UPDATE").
		Set("last_height = excluded.last_height").
		Set("last_time = excluded.last_time").
		Exec(ctx)
	return err
}
