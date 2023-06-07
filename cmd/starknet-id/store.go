package main

import (
	"context"
	"time"

	"github.com/dipdup-io/starknet-id/internal/storage"
	"github.com/dipdup-io/starknet-id/internal/storage/postgres"
	sdk "github.com/dipdup-net/indexer-sdk/pkg/storage"
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// Action
type Action int

// actions
const (
	ActionInsert Action = iota
	ActionUpdate
	ActionDelete
)

// TypeWithAction -
type TypeWithAction[T any] struct {
	Action Action
	Data   T
}

// NewTypeWithAction -
func NewTypeWithAction[T any](data T, action Action) *TypeWithAction[T] {
	return &TypeWithAction[T]{
		Action: action,
		Data:   data,
	}
}

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
	if blockCtx.isEmpty() {
		return nil
	}
	since := time.Now()

	tx, err := s.pg.Transactable.BeginTransaction(ctx)
	if err != nil {
		return err
	}
	defer tx.Close(ctx)

	if err := s.saveAddresses(ctx, tx, blockCtx); err != nil {
		return tx.HandleError(ctx, err)
	}

	if err := s.saveStarknetId(ctx, tx, blockCtx); err != nil {
		return tx.HandleError(ctx, err)
	}
	if err := s.addDomains(ctx, tx, blockCtx); err != nil {
		return tx.HandleError(ctx, err)
	}
	if err := s.saveFields(ctx, tx, blockCtx); err != nil {
		return tx.HandleError(ctx, err)
	}

	if _, err := tx.Exec(ctx, `INSERT INTO state (name, last_height, last_time)
		VALUES (?,?,?)
		ON CONFLICT (name)
		DO
		UPDATE SET last_height = excluded.last_height, last_time = excluded.last_time
	`, blockCtx.state.Name, blockCtx.state.LastHeight, blockCtx.state.LastTime); err != nil {
		return tx.HandleError(ctx, err)
	}

	if err := tx.Flush(ctx); err != nil {
		return tx.HandleError(ctx, err)
	}
	blockCtx.reset()

	log.Info().
		Str("channel", blockCtx.state.Name).
		Uint64("height", blockCtx.state.LastHeight).
		Uint64("save_time_ms", uint64(time.Since(since).Milliseconds())).
		Msg("indexed")
	return nil
}

func (s Store) saveAddresses(ctx context.Context, tx sdk.Transaction, blockCtx *BlockContext) error {
	if blockCtx.addresses.Len() == 0 {
		return nil
	}
	addresses := make([]any, 0)
	if err := blockCtx.addresses.Range(func(k string, v *storage.Address) (bool, error) {
		addresses = append(addresses, v)
		return false, nil
	}); err != nil {
		return err
	}

	if err := tx.BulkSave(ctx, addresses); err != nil {
		return errors.Wrap(err, "saving addresses")
	}
	return nil
}

func (s Store) saveStarknetId(ctx context.Context, tx sdk.Transaction, blockCtx *BlockContext) error {
	if blockCtx.starknetIds.Len() > 0 {
		minted := make([]any, 0)
		burned := make([]string, 0)
		if err := blockCtx.starknetIds.Range(func(s string, typ *TypeWithAction[*storage.StarknetId]) (bool, error) {
			switch typ.Action {
			case ActionDelete:
				burned = append(burned, typ.Data.StarknetId.String())
			case ActionInsert:
				minted = append(minted, typ.Data)
			case ActionUpdate:
				_, err := tx.Exec(ctx,
					`UPDATE starknet_id SET owner_address = ?, owner_id = ? WHERE starknet_id = ?`,
					typ.Data.OwnerAddress, typ.Data.OwnerId, typ.Data.StarknetId.String())
				return false, err
			}

			return false, nil
		}); err != nil {
			return errors.Wrap(err, "saving transferred starknet id")
		}
		if len(minted) > 0 {
			if err := tx.BulkSave(ctx, minted); err != nil {
				return errors.Wrap(err, "saving minted starknet id")
			}
		}
		if len(burned) > 0 {
			if _, err := tx.Exec(ctx, `DELETE FROM starknet_id WHERE starknet_id IN (?)`, pg.In(burned)); err != nil {
				return errors.Wrap(err, "saving burned starknet id")
			}
		}
	}

	return nil
}

func (s Store) addDomains(ctx context.Context, tx sdk.Transaction, blockCtx *BlockContext) error {
	if blockCtx.domains.Len() > 0 {
		if err := blockCtx.domains.Range(func(k string, v *storage.Domain) (bool, error) {
			_, err := tx.Exec(ctx, `INSERT INTO domain (address_id, address_hash, domain, owner, expiry)
			VALUES (?,?,?,?,?)
			ON CONFLICT (domain)
			DO 
			UPDATE SET address_id = excluded.address_id, address_hash = excluded.address_hash, owner = excluded.owner, expiry = excluded.expiry`,
				v.AddressId, v.AddressHash, v.Domain, v.Owner.String(), v.Expiry,
			)
			return false, err
		}); err != nil {
			return errors.Wrap(err, "saving domain")
		}
	}
	if blockCtx.transferredDomains.Len() > 0 {
		if err := blockCtx.transferredDomains.Range(func(s string, si *storage.Domain) (bool, error) {
			_, err := tx.Exec(ctx, `UPDATE domain SET owner = ? WHERE domain = ?`, si.Owner, si.Domain)
			return false, err
		}); err != nil {
			return errors.Wrap(err, "saving transferred domain")
		}
	}
	return nil
}

func (s Store) saveFields(ctx context.Context, tx sdk.Transaction, blockCtx *BlockContext) error {
	if blockCtx.fields.Len() == 0 {
		return nil
	}
	if err := blockCtx.fields.Range(func(k string, v *storage.Field) (bool, error) {
		_, err := tx.Exec(ctx, `INSERT INTO field (owner_id, name, namespace, value)
			VALUES (?,?,?,?)
			ON CONFLICT (namespace,owner_id,name)
			DO 
			UPDATE SET value = excluded.value`,
			v.OwnerId.String(), v.Name, v.Namespace, v.Value,
		)
		return false, err
	}); err != nil {
		return errors.Wrap(err, "saving field")
	}
	return nil
}
