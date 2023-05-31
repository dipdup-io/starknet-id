package postgres

import (
	"context"

	models "github.com/dipdup-io/starknet-id/internal/storage"
	"github.com/dipdup-net/go-lib/config"
	"github.com/dipdup-net/go-lib/database"
	"github.com/dipdup-net/indexer-sdk/pkg/storage/postgres"
	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/rs/zerolog/log"
)

// Storage -
type Storage struct {
	*postgres.Storage

	Domains     models.IDomain
	StarknetIds models.IStarknetId
	State       models.IState
}

// Create -
func Create(ctx context.Context, cfg config.Database) (Storage, error) {
	strg, err := postgres.Create(ctx, cfg, initDatabase)
	if err != nil {
		return Storage{}, err
	}

	s := Storage{
		Storage:     strg,
		State:       NewState(strg.Connection()),
		StarknetIds: NewStarknetId(strg.Connection()),
		Domains:     NewDomain(strg.Connection()),
	}

	return s, nil
}

func initDatabase(ctx context.Context, conn *database.PgGo) error {
	for _, data := range models.Models {
		if err := conn.DB().WithContext(ctx).Model(data).CreateTable(&orm.CreateTableOptions{
			IfNotExists: true,
		}); err != nil {
			if err := conn.Close(); err != nil {
				return err
			}
			return err
		}
	}

	if err := makeComments(ctx, conn); err != nil {
		return err
	}

	return createIndices(ctx, conn)
}

func createIndices(ctx context.Context, conn *database.PgGo) error {
	log.Info().Msg("creating indexes...")
	return conn.DB().RunInTransaction(ctx, func(tx *pg.Tx) error {
		// Starknet id
		if _, err := tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS starknet_identity_idx ON starknet_id (starknet_id)`); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS starkner_id_owner_idx ON starknet_id (owner_address)`); err != nil {
			return err
		}

		// Domain
		if _, err := tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS domain_name_idx ON domain USING hash(domain)`); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS domain_address_idx ON domain (address)`); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS domain_address_id_idx ON domain (address_id)`); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS domain_owner_id_idx ON domain (owner_id)`); err != nil {
			return err
		}

		return nil
	})
}
