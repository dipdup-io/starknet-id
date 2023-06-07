package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/dipdup-io/starknet-go-api/pkg/data"
	"github.com/dipdup-io/starknet-go-api/pkg/encoding"
	starknetid "github.com/dipdup-io/starknet-id/internal/starknet-id"
	"github.com/dipdup-io/starknet-id/internal/storage"
	"github.com/dipdup-io/starknet-indexer/pkg/grpc/pb"
)

// BlockContext -
type BlockContext struct {
	domains            *syncMap[string, *storage.Domain]
	transferredDomains *syncMap[string, *storage.Domain]
	starknetIds        *syncMap[string, *TypeWithAction[*storage.StarknetId]]
	fields             *syncMap[string, *storage.Field]
	addresses          *syncMap[string, *storage.Address]

	state *storage.State
}

func newBlockContext() *BlockContext {
	return &BlockContext{
		domains:            newSyncMap[string, *storage.Domain](),
		transferredDomains: newSyncMap[string, *storage.Domain](),
		starknetIds:        newSyncMap[string, *TypeWithAction[*storage.StarknetId]](),
		fields:             newSyncMap[string, *storage.Field](),
		addresses:          newSyncMap[string, *storage.Address](),
		state:              new(storage.State),
	}
}

func (bc *BlockContext) isEmpty() bool {
	return bc.domains.Len() == 0 &&
		bc.fields.Len() == 0 &&
		bc.transferredDomains.Len() == 0 &&
		bc.addresses.Len() == 0 &&
		bc.starknetIds.Len() == 0
}

func (bc *BlockContext) reset() {
	bc.domains.Reset()
	bc.transferredDomains.Reset()
	bc.starknetIds.Reset()
	bc.fields.Reset()
	bc.addresses.Reset()
}

func (bc *BlockContext) findAddress(ctx context.Context, addresses storage.IAddress, hash []byte) (*storage.Address, error) {
	addr, err := addresses.GetByHash(ctx, hash)
	if err != nil {
		if addresses.IsNoRows(err) {
			key := hex.EncodeToString(hash)
			address, ok := bc.addresses.Get(key)
			if ok {
				return address, nil
			}
		}
		return nil, err
	}
	return &addr, nil
}

func (bc *BlockContext) addDomains(ctx context.Context, addresses storage.IAddress, domains []data.Felt, address data.Felt) error {
	hash := address.Bytes()
	addr, err := bc.findAddress(ctx, addresses, hash)
	if err != nil {
		return err
	}
	for i := range domains {
		decoded, err := starknetid.Decode(domains[i])
		if err != nil {
			return err
		}
		if item, ok := bc.domains.Get(decoded); ok {
			item.AddressHash = hash
			item.Domain = decoded
			item.AddressId = addr.Id
		} else {
			bc.domains.Set(decoded, &storage.Domain{
				AddressHash: hash,
				AddressId:   addr.Id,
				Domain:      decoded,
			})
		}
	}
	return nil
}

func (bc *BlockContext) applyStaknetIdUpdate(update starknetid.StarknetIdUpdate) error {
	for i := range update.Domain {
		decoded, err := starknetid.Decode(update.Domain[i])
		if err != nil {
			return err
		}
		expiry, err := update.Expiry.Uint64()
		if err != nil {
			return err
		}
		if item, ok := bc.domains.Get(decoded); ok {
			item.Expiry = time.Unix(int64(expiry), 0).UTC()
			item.Owner = update.Owner.Decimal()
		} else {
			bc.domains.Set(decoded, &storage.Domain{
				Expiry: time.Unix(int64(expiry), 0).UTC(),
				Owner:  update.Owner.Decimal(),
				Domain: decoded,
			})
		}
	}
	return nil
}

func (bc *BlockContext) applyDomainTransfer(update starknetid.DomainTransfer) error {
	for i := range update.Domain {
		decoded, err := starknetid.Decode(update.Domain[i])
		if err != nil {
			return err
		}
		bc.transferredDomains.Set(decoded, &storage.Domain{
			Domain: decoded,
			Owner:  update.NewOwner.Decimal(),
		})
	}
	return nil
}

func (bc *BlockContext) addMintedStarknetId(ctx context.Context, addresses storage.IAddress, transfer starknetid.Transfer) error {
	tokenId, err := transfer.TokenId.Decimal()
	if err != nil {
		return err
	}
	hash := transfer.To.Bytes()
	addr, err := bc.findAddress(ctx, addresses, hash)
	if err != nil {
		return err
	}

	sid := &storage.StarknetId{
		StarknetId:   tokenId,
		OwnerAddress: hash,
		OwnerId:      addr.Id,
	}
	bc.starknetIds.Set(transfer.TokenId.String(), NewTypeWithAction(sid, ActionInsert))
	return nil
}

func (bc *BlockContext) addBurnedStarknetId(ctx context.Context, addresses storage.IAddress, transfer starknetid.Transfer) error {
	tokenId, err := transfer.TokenId.Decimal()
	if err != nil {
		return err
	}
	hash := transfer.From.Bytes()
	addr, err := bc.findAddress(ctx, addresses, hash)
	if err != nil {
		return err
	}

	sid := &storage.StarknetId{
		StarknetId:   tokenId,
		OwnerAddress: hash,
		OwnerId:      addr.Id,
	}
	if item, ok := bc.starknetIds.Get(transfer.TokenId.String()); ok {
		if item.Action == ActionUpdate {
			item.Action = ActionDelete
		}
	} else {
		bc.starknetIds.Set(transfer.TokenId.String(), NewTypeWithAction(sid, ActionDelete))
	}
	return nil
}

func (bc *BlockContext) addTransferedStarknetId(ctx context.Context, addresses storage.IAddress, transfer starknetid.Transfer) error {
	tokenId, err := transfer.TokenId.Decimal()
	if err != nil {
		return err
	}
	hash := transfer.To.Bytes()
	addr, err := bc.findAddress(ctx, addresses, hash)
	if err != nil {
		return err
	}

	sid := &storage.StarknetId{
		StarknetId:   tokenId,
		OwnerAddress: hash,
		OwnerId:      addr.Id,
	}

	if item, ok := bc.starknetIds.Get(transfer.TokenId.String()); ok {
		item.Data = sid
	} else {
		bc.starknetIds.Set(transfer.TokenId.String(), NewTypeWithAction(sid, ActionUpdate))
	}
	return nil
}

func (bc *BlockContext) addField(update starknetid.VerifierDataUpdate) error {
	starknetId := update.StarknetId.Decimal()
	key := fmt.Sprintf("%s_%s_%d", starknetId.String(), update.Field.String(), storage.FieldNamespaceVerifier)
	bc.fields.Set(key, &storage.Field{
		OwnerId:   starknetId,
		Namespace: storage.FieldNamespaceVerifier,
		Name:      update.Field.ToAsciiString(),
		Value:     encoding.MustDecodeHex(update.Data.String()),
	})
	return nil
}

func (bc *BlockContext) addAddress(address *pb.Address) {
	key := hex.EncodeToString(address.GetHash())
	bc.addresses.Set(key, &storage.Address{
		Id:      address.GetId(),
		Hash:    address.GetHash(),
		Height:  address.GetHeight(),
		ClassId: address.ClassId,
	})
}

func (bc *BlockContext) updateState(name string, height uint64) {
	bc.state.LastHeight = height
	bc.state.LastTime = time.Now().UTC()
	bc.state.Name = name
}

type syncMap[K comparable, V any] struct {
	m  map[K]V
	mx *sync.RWMutex
}

func newSyncMap[K comparable, V any]() *syncMap[K, V] {
	return &syncMap[K, V]{
		make(map[K]V), new(sync.RWMutex),
	}
}

// Get -
func (m *syncMap[K, V]) Get(key K) (V, bool) {
	m.mx.RLock()
	defer m.mx.RUnlock()

	val, ok := m.m[key]
	return val, ok
}

// Set -
func (m *syncMap[K, V]) Set(key K, value V) {
	m.mx.Lock()
	m.m[key] = value
	m.mx.Unlock()
}

// Range -
func (m *syncMap[K, V]) Range(handler func(K, V) (bool, error)) error {
	m.mx.RLock()
	defer m.mx.RUnlock()

	for key, value := range m.m {
		stop, err := handler(key, value)
		if err != nil {
			return err
		}
		if stop {
			return nil
		}
	}
	return nil
}

// Delete -
func (m *syncMap[K, V]) Delete(key K) {
	m.mx.Lock()
	delete(m.m, key)
	m.mx.Unlock()
}

// Reset -
func (m *syncMap[K, V]) Reset() {
	m.mx.Lock()
	for key := range m.m {
		delete(m.m, key)
	}
	m.mx.Unlock()
}

// Len -
func (m *syncMap[K, V]) Len() int {
	m.mx.RLock()
	defer m.mx.RUnlock()
	return len(m.m)
}

// GetOrCreate -
func (m *syncMap[K, V]) GetOrCreate(key K, constructor func() V) V {
	m.mx.Lock()
	defer m.mx.Unlock()

	if val, ok := m.m[key]; ok {
		return val
	} else {
		if constructor != nil {
			m.m[key] = constructor()
		} else {
			var v V
			m.m[key] = v
		}
		return m.m[key]
	}
}
