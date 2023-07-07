package storage

import (
	"github.com/dipdup-net/indexer-sdk/pkg/storage"
	"github.com/shopspring/decimal"
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
	// nolint
	tableName struct{} `pg:"field" comment:"Field table"`

	Id        uint64          `pg:",pk,notnull" comment:"Unique internal identity"`
	OwnerId   decimal.Decimal `pg:",type:numeric,use_zero" comment:"Starknet Id (token id)"`
	Namespace FieldNamespace  `pg:",type:SMALLINT" comment:"Kind of namespace"`
	Name      string          `comment:"Field name"`
	Value     []byte          `comment:"Field value"`

	Owner StarknetId `pg:"-" hasura:"table:starknet_id,field:owner_id,remote_field:id,type:oto,name:starknet_id"`
}

// TableName -
func (Field) TableName() string {
	return "field"
}
