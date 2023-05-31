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
	tableName struct{} `pg:"starknet_id,comment:Starknet id table"`

	Id           uint64          `pg:",pk,notnull,comment:Unique internal identity"`
	StarknetId   decimal.Decimal `pg:",unique,type:numeric,use_zero,comment:Starknet Id (token id)"`
	OwnerAddress []byte          `pg:",comment:Address hash of token owner"`
}

// TableName -
func (StarknetId) TableName() string {
	return "starknet_id"
}
