package postgres

import (
	"context"

	models "github.com/dipdup-io/starknet-id/internal/storage"
	"github.com/dipdup-net/go-lib/config"
	"github.com/dipdup-net/go-lib/database"
	"github.com/dipdup-net/indexer-sdk/pkg/storage/postgres"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/uptrace/bun"
)

// Storage -
type Storage struct {
	*postgres.Storage

	Addresses   models.IAddress
	Domains     models.IDomain
	Subdomains  models.ISubdomain
	StarknetIds models.IStarknetId
	Fields      models.IField
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
		Addresses:   NewAddress(strg.Connection()),
		StarknetIds: NewStarknetId(strg.Connection()),
		Domains:     NewDomain(strg.Connection()),
		Subdomains:  NewSubdomain(strg.Connection()),
		Fields:      NewField(strg.Connection()),
	}

	return s, nil
}

func initDatabase(ctx context.Context, conn *database.Bun) error {
	data := make([]any, len(models.Models))
	for i := range models.Models {
		data[i] = models.Models[i]
	}

	if err := database.CreateTables(ctx, conn, data...); err != nil {
		if err := conn.Close(); err != nil {
			return err
		}
		return err
	}

	if err := database.MakeComments(ctx, conn, data...); err != nil {
		return errors.Wrap(err, "make comments")
	}

	return createIndices(ctx, conn)
}

func createIndices(ctx context.Context, conn *database.Bun) error {
	log.Info().Msg("creating indexes...")
	return conn.DB().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Address
		if _, err := tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS address_hash_idx ON address (hash)`); err != nil {
			return err
		}

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
		if _, err := tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS domain_address_idx ON domain (address_hash)`); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS domain_address_id_idx ON domain (address_id)`); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS domain_owner_idx ON domain (owner)`); err != nil {
			return err
		}

		// Subdomain
		if _, err := tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS subdomain_name_idx ON subdomain USING hash(subdomain)`); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS subdomain_resolver_id_idx ON subdomain (resolver_id)`); err != nil {
			return err
		}

		// Field
		if _, err := tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS field_name_idx ON field USING hash(name)`); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS field_starknet_id_idx ON field (owner_id)`); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS field_key_idx ON field (namespace,owner_id,name)`); err != nil {
			return err
		}

		return nil
	})
}
