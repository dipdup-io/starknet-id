package storage

import (
	"context"

	"github.com/dipdup-net/indexer-sdk/pkg/storage"
	"github.com/uptrace/bun"
)

// IAddress -
type IAddress interface {
	storage.Table[*Address]

	GetByHash(ctx context.Context, hash []byte) (Address, error)
}

// Address -
type Address struct {
	bun.BaseModel `bun:"address" comment:"Address table"`

	Id      uint64  `bun:"id,notnull,type:bigint,pk" comment:"Unique internal identity"`
	Hash    []byte  `comment:"Starknet hash address"`
	Height  uint64  `comment:"Block number of the first address occurrence."`
	ClassId *uint64 `bun:",nullzero" comment:"Internal class identity"`
}

// TableName -
func (Address) TableName() string {
	return "address"
}
