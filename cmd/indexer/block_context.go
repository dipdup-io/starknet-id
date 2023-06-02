package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/dipdup-io/starknet-go-api/pkg/data"
	starknetid "github.com/dipdup-io/starknet-id/internal/starknet-id"
	"github.com/dipdup-io/starknet-id/internal/storage"
)

// BlockContext -
type BlockContext struct {
	domains                *syncMap[string, *storage.Domain]
	transferredDomains     *syncMap[string, *storage.Domain]
	mintedStarknetIds      *syncMap[string, *storage.StarknetId]
	burnedStarknetIds      *syncMap[string, *storage.StarknetId]
	transferredStarknetIds *syncMap[string, *storage.StarknetId]
	fields                 *syncMap[string, *storage.Field]

	state *storage.State
}

func newBlockContext() *BlockContext {
	return &BlockContext{
		domains:                newSyncMap[string, *storage.Domain](),
		transferredDomains:     newSyncMap[string, *storage.Domain](),
		mintedStarknetIds:      newSyncMap[string, *storage.StarknetId](),
		burnedStarknetIds:      newSyncMap[string, *storage.StarknetId](),
		transferredStarknetIds: newSyncMap[string, *storage.StarknetId](),
		fields:                 newSyncMap[string, *storage.Field](),
		state:                  new(storage.State),
	}
}

func (bc *BlockContext) isEmpty() bool {
	return bc.domains.Len() == 0 &&
		bc.burnedStarknetIds.Len() == 0 &&
		bc.fields.Len() == 0 &&
		bc.mintedStarknetIds.Len() == 0 &&
		bc.transferredDomains.Len() == 0 &&
		bc.transferredStarknetIds.Len() == 0
}

func (bc *BlockContext) reset() {
	bc.domains.Reset()
	bc.transferredDomains.Reset()
	bc.mintedStarknetIds.Reset()
	bc.burnedStarknetIds.Reset()
	bc.transferredStarknetIds.Reset()
	bc.fields.Reset()
}

func (bc *BlockContext) addDomains(domains []data.Felt, address data.Felt) error {
	hash := address.Bytes()
	for i := range domains {
		decoded, err := starknetid.Decode(domains[i])
		if err != nil {
			return err
		}
		if item, ok := bc.domains.Get(decoded); ok {
			item.Address = hash
			item.Domain = decoded
		} else {
			bc.domains.Set(decoded, &storage.Domain{
				Address: hash,
				Domain:  decoded,
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

func (bc *BlockContext) addMintedStarknetId(transfer starknetid.Transfer) error {
	tokenId, err := transfer.TokenId.Decimal()
	if err != nil {
		return err
	}
	bc.mintedStarknetIds.Set(transfer.TokenId.String(), &storage.StarknetId{
		StarknetId:   tokenId,
		OwnerAddress: transfer.To.Bytes(),
	})
	return nil
}

func (bc *BlockContext) addBurnedStarknetId(transfer starknetid.Transfer) error {
	tokenId, err := transfer.TokenId.Decimal()
	if err != nil {
		return err
	}
	bc.burnedStarknetIds.Set(transfer.TokenId.String(), &storage.StarknetId{
		StarknetId:   tokenId,
		OwnerAddress: transfer.From.Bytes(),
	})
	return nil
}

func (bc *BlockContext) addTransferedStarknetId(transfer starknetid.Transfer) error {
	tokenId, err := transfer.TokenId.Decimal()
	if err != nil {
		return err
	}
	bc.transferredStarknetIds.Set(transfer.TokenId.String(), &storage.StarknetId{
		StarknetId:   tokenId,
		OwnerAddress: transfer.To.Bytes(),
	})
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

func (bc *BlockContext) addField(update starknetid.VerifierDataUpdate) error {
	starknetId := update.StarknetId.Decimal()
	key := fmt.Sprintf("%s_%s_%d", starknetId.String(), update.Field.String(), storage.FieldNamespaceVerifier)
	bc.fields.Set(key, &storage.Field{
		StarknetId: starknetId,
		Namespace:  storage.FieldNamespaceVerifier,
		Name:       update.Field.ToAsciiString(),
		Value:      update.Data.Bytes(),
	})
	return nil
}

func (bc *BlockContext) updateState(name string, height, ts uint64) {
	bc.state.LastHeight = height
	bc.state.LastTime = time.Unix(int64(ts), 0).UTC()
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
