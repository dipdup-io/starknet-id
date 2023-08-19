package storage

import (
	"github.com/dipdup-net/indexer-sdk/pkg/storage"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

// FieldNamespace -
type FieldNamespace int

// FieldNamespace values
const (
	FieldNamespaceVerifier FieldNamespace = iota + 1
	FieldNamespaceUser
)

// IField -
type IField interface {
	storage.Table[*Field]
}

// Field
type Field struct {
	bun.BaseModel `bun:"field" comment:"Field table"`

	Id        uint64          `bun:"id,pk,autoincrement" comment:"Unique internal identity"`
	OwnerId   decimal.Decimal `bun:",type:numeric,use_zero" comment:"Starknet Id (token id)"`
	Namespace FieldNamespace  `bun:",type:SMALLINT" comment:"Kind of namespace"`
	Name      string          `comment:"Field name"`
	Value     []byte          `comment:"Field value"`

	Owner StarknetId `bun:"-" hasura:"table:starknet_id,field:owner_id,remote_field:id,type:oto,name:starknet_id"`
}

// TableName -
func (Field) TableName() string {
	return "field"
}
