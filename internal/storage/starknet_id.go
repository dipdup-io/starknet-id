package storage

import (
	"github.com/dipdup-net/indexer-sdk/pkg/storage"
	"github.com/shopspring/decimal"
)

// IStarknetId -
type IStarknetId interface {
	storage.Table[*StarknetId]
}

// StarknetId -
type StarknetId struct {
	// nolint
	tableName struct{} `pg:"starknet_id" comment:"Starknet id table"`

	Id           uint64          `pg:",pk,notnull" comment:"Unique internal identity"`
	StarknetId   decimal.Decimal `pg:",unique,type:numeric,use_zero" comment:"Starknet Id (token id)"`
	OwnerAddress []byte          `comment:"Address hash of token owner"`
	OwnerId      uint64          `comment:"Owner identity of address"`

	Owner  Address `pg:"-" hasura:"table:address,field:owner_id,remote_field:id,type:oto,name:owner"`
	Fields []Field `pg:"-" hasura:"table:field,field:starknet_id,remote_field:owner_id,type:otm,name:fields"`
}

// TableName -
func (StarknetId) TableName() string {
	return "starknet_id"
}
