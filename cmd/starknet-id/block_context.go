package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dipdup-io/starknet-go-api/pkg/data"
	"github.com/dipdup-io/starknet-go-api/pkg/encoding"
	starknetid "github.com/dipdup-io/starknet-id/internal/starknet-id"
	"github.com/dipdup-io/starknet-id/internal/storage"
	"github.com/dipdup-io/starknet-indexer/pkg/grpc/pb"
	"github.com/pkg/errors"
)

// BlockContext -
type BlockContext struct {
	cache *Cache

	domains            *syncMap[string, *storage.Domain]
	transferredDomains *syncMap[string, *storage.Domain]
	starknetIds        *syncMap[string, *TypeWithAction[*storage.StarknetId]]
	fields             *syncMap[string, *storage.Field]
	addresses          *syncMap[string, *storage.Address]
	subdomains         *syncMap[string, *storage.Subdomain]

	addressRepo   storage.IAddress
	subdomainsMap map[string]string

	state *storage.State
}

func newBlockContext(
	subdomainRepo storage.ISubdomain,
	addressRepo storage.IAddress,
	subdomainsMap map[string]string,
) *BlockContext {
	return &BlockContext{
		cache:              NewCache(subdomainRepo),
		domains:            newSyncMap[string, *storage.Domain](),
		transferredDomains: newSyncMap[string, *storage.Domain](),
		starknetIds:        newSyncMap[string, *TypeWithAction[*storage.StarknetId]](),
		fields:             newSyncMap[string, *storage.Field](),
		addresses:          newSyncMap[string, *storage.Address](),
		subdomains:         newSyncMap[string, *storage.Subdomain](),
		addressRepo:        addressRepo,
		subdomainsMap:      subdomainsMap,
		state:              new(storage.State),
	}
}

func (bc *BlockContext) isEmpty() bool {
	return bc.domains.Len() == 0 &&
		bc.fields.Len() == 0 &&
		bc.transferredDomains.Len() == 0 &&
		bc.addresses.Len() == 0 &&
		bc.starknetIds.Len() == 0 &&
		bc.subdomains.Len() == 0
}

func (bc *BlockContext) reset() {
	bc.domains.Reset()
	bc.transferredDomains.Reset()
	bc.starknetIds.Reset()
	bc.fields.Reset()
	bc.addresses.Reset()
	bc.subdomains.Reset()
}

func (bc *BlockContext) findAddress(ctx context.Context, hash []byte) (*storage.Address, error) {
	addr, err := bc.addressRepo.GetByHash(ctx, hash)
	if err != nil {
		if bc.addressRepo.IsNoRows(err) {
			key := hex.EncodeToString(hash)
			address, ok := bc.addresses.Get(key)
			if ok {
				return address, nil
			}
			return &storage.Address{
				Hash: hash,
			}, nil
		}
		return nil, err
	}
	return &addr, nil
}

func (bc *BlockContext) getFullDomainName(ctx context.Context, domains []data.Felt, contract storage.Address) (string, error) {
	parts, err := bc.decodeDomainName(domains)
	if err != nil {
		return "", err
	}
	var (
		subdomain string
		ok        bool
	)

	if subdomain, ok = bc.subdomainsMap[hex.EncodeToString(contract.Hash)]; !ok {
		subdomain, err = bc.cache.GetSubdomain(ctx, contract.Id)
		if err != nil {
			return "", errors.Wrap(err, "get subdomain")
		}
	}

	parts = append(parts, subdomain)

	if ok {
		parts = append(parts, rootDomain)
	}

	return strings.Join(parts, "."), nil
}

func (bc *BlockContext) decodeDomainName(domains []data.Felt) ([]string, error) {
	parts := make([]string, len(domains))
	for i := range domains {
		decoded, err := starknetid.Decode(domains[i])
		if err != nil {
			return nil, err
		}
		parts[i] = decoded
	}
	return parts, nil
}

func (bc *BlockContext) addDomains(ctx context.Context, domains []data.Felt, address data.Felt, contract storage.Address) error {
	hash := address.Bytes()
	addr, err := bc.findAddress(ctx, hash)
	if err != nil {
		return err
	}

	domain, err := bc.getFullDomainName(ctx, domains, contract)
	if err != nil {
		return err
	}
	if item, ok := bc.domains.Get(domain); ok {
		item.AddressHash = hash
		item.Domain = domain
		item.AddressId = addr.Id
	} else {
		bc.domains.Set(domain, &storage.Domain{
			AddressHash: hash,
			AddressId:   addr.Id,
			Domain:      domain,
		})
	}

	return nil
}

func (bc *BlockContext) applyStaknetIdUpdate(update starknetid.StarknetIdUpdate) error {
	parts, err := bc.decodeDomainName(update.Domain)
	if err != nil {
		return err
	}
	parts = append(parts, rootDomain)
	domain := strings.Join(parts, ".")

	expiry, err := update.Expiry.Uint64()
	if err != nil {
		return err
	}

	if item, ok := bc.domains.Get(domain); ok {
		item.Expiry = time.Unix(int64(expiry), 0).UTC()
		item.Owner = update.Owner.Decimal()
	} else {
		bc.domains.Set(domain, &storage.Domain{
			Expiry: time.Unix(int64(expiry), 0).UTC(),
			Owner:  update.Owner.Decimal(),
			Domain: domain,
		})
	}
	return nil
}

func (bc *BlockContext) applyDomainTransfer(update starknetid.DomainTransfer) error {
	parts, err := bc.decodeDomainName(update.Domain)
	if err != nil {
		return err
	}
	parts = append(parts, rootDomain)
	domain := strings.Join(parts, ".")
	bc.transferredDomains.Set(domain, &storage.Domain{
		Domain: domain,
		Owner:  update.NewOwner.Decimal(),
	})
	return nil
}

func (bc *BlockContext) addSubdomain(ctx context.Context, event *pb.Event, update starknetid.DomainToResolverUpdate) error {
	parts, err := bc.decodeDomainName(update.Domain)
	if err != nil {
		return err
	}
	domain := strings.Join(parts, ".")
	hash := update.Resolver.Bytes()
	addr, err := bc.findAddress(ctx, hash)
	if err != nil {
		return err
	}
	bc.cache.SetSubdomain(addr.Id, domain)
	bc.subdomains.Set(domain, &storage.Subdomain{
		RegistrationHeight: event.Height,
		RegistrationDate:   time.Unix(int64(event.Time), 0),
		ResolverId:         addr.Id,
		Subdomain:          domain,
	})
	return nil
}

func (bc *BlockContext) addMintedStarknetId(ctx context.Context, transfer starknetid.Transfer) error {
	tokenId, err := transfer.TokenId.Decimal()
	if err != nil {
		return err
	}
	hash := transfer.To.Bytes()
	addr, err := bc.findAddress(ctx, hash)
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

func (bc *BlockContext) addBurnedStarknetId(ctx context.Context, transfer starknetid.Transfer) error {
	tokenId, err := transfer.TokenId.Decimal()
	if err != nil {
		return err
	}
	hash := transfer.From.Bytes()
	addr, err := bc.findAddress(ctx, hash)
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

func (bc *BlockContext) addTransferedStarknetId(ctx context.Context, transfer starknetid.Transfer) error {
	tokenId, err := transfer.TokenId.Decimal()
	if err != nil {
		return err
	}
	hash := transfer.To.Bytes()
	addr, err := bc.findAddress(ctx, hash)
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
