package storage

import (
	"time"

	"github.com/dipdup-net/indexer-sdk/pkg/storage"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

// IDomain -
type IDomain interface {
	storage.Table[*Domain]
}

// Domain -
type Domain struct {
	bun.BaseModel `bun:"domain" comment:"Domains table"`

	Id          uint64          `bun:"id,pk,autoincrement" comment:"Unique internal identity"`
	AddressId   uint64          `comment:"Address id from main indexer"`
	AddressHash []byte          `comment:"Address hash"`
	Domain      string          `bun:",unique" comment:"Domain string"`
	Owner       decimal.Decimal `bun:",type:numeric,use_zero" comment:"Owner's starknet id"`
	Expiry      time.Time       `comment:"Expiration time"`

	Address    Address    `bun:"-" hasura:"table:address,field:address_id,remote_field:id,type:oto,name:address"`
	StarknetId StarknetId `bun:"-" hasura:"table:starknet_id,field:owner,remote_field:id,type:oto,name:starknet_id"`
}

// TableName -
func (Domain) TableName() string {
	return "domain"
}
