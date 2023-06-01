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
	tableName struct{} `pg:"field,comment:Field table"`

	Id         uint64          `pg:",pk,notnull,comment:Unique internal identity"`
	StarknetId decimal.Decimal `pg:",type:numeric,use_zero,comment:Starknet Id (token id)"`
	Namespace  FieldNamespace  `pg:",type:SMALLINT,comment:Kind of namespace"`
	Name       string          `pg:",comment:Field name"`
	Value      []byte          `pg:",comment:Field value"`
}

// TableName -
func (Field) TableName() string {
	return "field"
}
