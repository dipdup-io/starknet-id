package storage

import (
	"context"

	"github.com/dipdup-net/indexer-sdk/pkg/storage"
)

// IAddress -
type IAddress interface {
	storage.Table[*Address]

	GetByHash(ctx context.Context, hash []byte) (Address, error)
}

// Address -
type Address struct {
	// nolint
	tableName struct{} `pg:"address" comment:"Address table"`

	Id      uint64  `pg:"id,notnull,type:bigint,pk" comment:"Unique internal identity"`
	Hash    []byte  `comment:"Starknet hash address"`
	Height  uint64  `comment:"Block number of the first address occurrence."`
	ClassId *uint64 `comment:"Internal class identity"`
}

// TableName -
func (Address) TableName() string {
	return "address"
}
