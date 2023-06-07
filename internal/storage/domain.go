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

	Id          uint64          `pg:",pk,comment:Unique internal identity"`
	AddressId   uint64          `pg:",comment:Address id from main indexer"`
	AddressHash []byte          `pg:",comment:Address hash"`
	Domain      string          `pg:",unique,comment:Domain string"`
	Owner       decimal.Decimal `pg:",type:numeric,use_zero,comment:Owner's starknet id"`
	Expiry      time.Time       `pg:",comment:Expiration time"`

	Address    Address    `pg:"-" hasura:"table:address,field:address_id,remote_field:id,type:oto,name:address"`
	StarknetId StarknetId `pg:"-" hasura:"table:starknet_id,field:owner,remote_field:id,type:oto,name:starknet_id"`
}

// TableName -
func (Domain) TableName() string {
	return "domain"
}
