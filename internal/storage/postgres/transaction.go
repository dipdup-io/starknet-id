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

// SaveState -
func (t Transaction) SaveState(ctx context.Context, state *models.State) error {
	_, err := t.Tx().NewInsert().Model(state).
		On("CONFLICT (name) DO UPDATE").
		Set("last_height = excluded.last_height").
		Set("last_time = excluded.last_time").
		Exec(ctx)
	return err
}

func (t Transaction) SaveAddress(ctx context.Context, addresses ...*models.Address) error {
	if len(addresses) == 0 {
		return nil
	}
	_, err := t.Tx().NewInsert().Model(&addresses).
		On("CONFLICT (id) DO UPDATE").
		Set("class_id = excluded.class_id").
		Exec(ctx)
	return err
}
