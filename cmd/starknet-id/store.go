package main

import (
	"context"
	"time"

	"github.com/dipdup-io/starknet-id/internal/storage"
	"github.com/dipdup-io/starknet-id/internal/storage/postgres"
	sdk "github.com/dipdup-net/indexer-sdk/pkg/storage"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/uptrace/bun"
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
	since := time.Now()
	tx, err := postgres.BeginTransaction(ctx, s.pg.Transactable)
	if err != nil {
		return err
	}
	defer tx.Close(ctx)

	if !blockCtx.isEmpty() {
		if err := s.saveAddresses(ctx, tx, blockCtx); err != nil {
			return tx.HandleError(ctx, err)
		}
		if err := s.saveStarknetId(ctx, tx, blockCtx); err != nil {
			return tx.HandleError(ctx, err)
		}
		if err := s.saveSubdomains(ctx, tx, blockCtx); err != nil {
			return tx.HandleError(ctx, err)
		}
		if err := s.addDomains(ctx, tx, blockCtx); err != nil {
			return tx.HandleError(ctx, err)
		}
		if err := s.saveFields(ctx, tx, blockCtx); err != nil {
			return tx.HandleError(ctx, err)
		}
	}

	if err := tx.SaveState(ctx, blockCtx.state); err != nil {
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

func (s Store) saveAddresses(ctx context.Context, tx postgres.Transaction, blockCtx *BlockContext) error {
	if blockCtx.addresses.Len() == 0 {
		return nil
	}
	addresses := make([]*storage.Address, 0)
	if err := blockCtx.addresses.Range(func(k string, v *storage.Address) (bool, error) {
		addresses = append(addresses, v)
		return false, nil
	}); err != nil {
		return err
	}

	if err := tx.SaveAddress(ctx, addresses...); err != nil {
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
			if _, err := tx.Exec(ctx, `DELETE FROM starknet_id WHERE starknet_id IN (?)`, bun.In(burned)); err != nil {
				return errors.Wrap(err, "saving burned starknet id")
			}
		}
	}

	return nil
}

func (s Store) addDomains(ctx context.Context, tx sdk.Transaction, blockCtx *BlockContext) error {
	if blockCtx.domains.Len() > 0 {
		if err := blockCtx.domains.Range(func(k string, v *storage.Domain) (bool, error) {
			var (
				query         string
				timeIsZero    = v.Expiry.IsZero()
				addressIsNull = len(v.AddressHash) == 0
			)

			switch {
			case !timeIsZero && !addressIsNull:
				query = `INSERT INTO domain (address_id, address_hash, domain, owner, expiry)
				VALUES (?,?,?,?,?)
				ON CONFLICT (domain)
				DO 
				UPDATE SET address_id = excluded.address_id, address_hash = excluded.address_hash, owner = excluded.owner, expiry = excluded.expiry`
			case timeIsZero && !addressIsNull:
				query = `INSERT INTO domain (address_id, address_hash, domain, owner, expiry)
				VALUES (?,?,?,?,?)
				ON CONFLICT (domain)
				DO 
				UPDATE SET address_id = excluded.address_id, address_hash = excluded.address_hash`
			case !timeIsZero && addressIsNull:
				query = `INSERT INTO domain (address_id, address_hash, domain, owner, expiry)
				VALUES (?,?,?,?,?)
				ON CONFLICT (domain)
				DO 
				UPDATE SET owner = excluded.owner, expiry = excluded.expiry`
			default:
				return false, nil
			}

			_, err := tx.Exec(ctx, query,
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

func (s Store) saveSubdomains(ctx context.Context, tx sdk.Transaction, blockCtx *BlockContext) error {
	if blockCtx.subdomains.Len() == 0 {
		return nil
	}
	if err := blockCtx.subdomains.Range(func(k string, v *storage.Subdomain) (bool, error) {
		_, err := tx.Exec(ctx, `INSERT INTO subdomain (registration_height, registration_date, resolver_id, subdomain)
			VALUES (?,?,?,?)
			ON CONFLICT (subdomain)
			DO 
			UPDATE SET registration_height = excluded.registration_height, registration_date = excluded.registration_date, resolver_id = excluded.resolver_id`,
			v.RegistrationHeight, v.RegistrationDate, v.ResolverId, v.Subdomain,
		)
		return false, err
	}); err != nil {
		return errors.Wrap(err, "saving subdomains")
	}
	return nil
}
