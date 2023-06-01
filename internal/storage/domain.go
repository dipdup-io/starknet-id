package storage

import (
	"time"

	"github.com/dipdup-net/indexer-sdk/pkg/storage"
	"github.com/shopspring/decimal"
)

// IDomain -
type IDomain interface {
	storage.Table[*Domain]
}

// Domain -
type Domain struct {
	// nolint
	tableName struct{} `pg:"domain,comment:Domains table"`

	Id        uint64          `pg:",pk,comment:Unique internal identity"`
	AddressId uint64          `pg:",comment:Address id from main indexer"`
	Address   []byte          `pg:",comment:Address hash"`
	Domain    string          `pg:",unique,comment:Domain string"`
	Owner     decimal.Decimal `pg:",type:numeric,use_zero,comment:Owner's starknet id"`
	Expiry    time.Time       `pg:",comment:Expiration time"`
}

// TableName -
func (Domain) TableName() string {
	return "domain"
}
