package main

import (
	"context"

	"github.com/dipdup-io/starknet-id/internal/storage"
	"github.com/dipdup-io/starknet-id/internal/storage/postgres"
	sdk "github.com/dipdup-net/indexer-sdk/pkg/storage"
	"github.com/pkg/errors"
)

// Store -
type Store struct {
	pg postgres.Storage
}

// NewStore -
func NewStore(pg postgres.Storage) Store {
	return Store{pg}
}

// Save -
func (s Store) Save(ctx context.Context, blockCtx *BlockContext) error {
	tx, err := s.pg.Transactable.BeginTransaction(ctx)
	if err != nil {
		return err
	}
	defer tx.Close(ctx)

	if err := s.saveStarknetId(ctx, tx, blockCtx); err != nil {
		return tx.HandleError(ctx, err)
	}
	if err := s.addDomains(ctx, tx, blockCtx); err != nil {
		return tx.HandleError(ctx, err)
	}

	if err := tx.Flush(ctx); err != nil {
		return tx.HandleError(ctx, err)
	}
	return nil
}

func (s Store) saveStarknetId(ctx context.Context, tx sdk.Transaction, blockCtx *BlockContext) error {
	if err := blockCtx.mintedStarknetIds.Range(func(s string, si *storage.StarknetId) (bool, error) {
		err := tx.Add(ctx, si)
		return false, err
	}); err != nil {
		return errors.Wrap(err, "saving minted starknet id")
	}

	if err := blockCtx.burnedStarknetIds.Range(func(s string, si *storage.StarknetId) (bool, error) {
		_, err := tx.Exec(ctx, `DELETE FROM starknet_id WHERE starknet_id = ?`, si.StarknetId.String())
		return false, err
	}); err != nil {
		return errors.Wrap(err, "saving burned starknet id")
	}

	if err := blockCtx.transferredStarknetIds.Range(func(s string, si *storage.StarknetId) (bool, error) {
		_, err := tx.Exec(ctx, `UPDATE starknet_id SET owner = ? WHERE starknet_id = ?`, si.OwnerAddress, si.StarknetId.String())
		return false, err
	}); err != nil {
		return errors.Wrap(err, "saving transferred starknet id")
	}
	return nil
}

func (s Store) addDomains(ctx context.Context, tx sdk.Transaction, blockCtx *BlockContext) error {
	if err := blockCtx.domains.Range(func(k string, v *storage.Domain) (bool, error) {
		_, err := tx.Exec(ctx, `INSERT INTO domain (id, address_id, address, domain, owner, expiry)
			VALUES (?,?,?,?,?,?)
			ON CONFLICT (domain)
			DO 
			UPDATE SET address_id = excluded.address_id, address = excluded.address, owner = excluded.owner, expiry = excluded.expiry`,
			v.Id, v.AddressId, v.Address, v.Domain, v.Owner.String(), v.Expiry,
		)
		return false, err
	}); err != nil {
		return errors.Wrap(err, "saving domain")
	}
	if err := blockCtx.transferredDomains.Range(func(s string, si *storage.Domain) (bool, error) {
		_, err := tx.Exec(ctx, `UPDATE domain SET owner = ? WHERE domain = ?`, si.Owner, si.Domain)
		return false, err
	}); err != nil {
		return errors.Wrap(err, "saving transferred domain")
	}
	return nil
}
