package storage

import (
	"context"
	"time"

	"github.com/dipdup-net/indexer-sdk/pkg/storage"
)

// ISubdomain -
type ISubdomain interface {
	storage.Table[*Subdomain]

	GetByResolverId(ctx context.Context, resolverId uint64) (result Subdomain, err error)
}

// Subdomain -
type Subdomain struct {
	// nolint
	tableName struct{} `pg:"subdomain,comment:Subdomain's table"`

	Id                 uint64    `pg:",pk,comment:Unique internal identity"`
	RegistrationHeight uint64    `pg:",comment:Height of first event about subdomain registration"`
	RegistrationDate   time.Time `pg:",comment:Date of first event about subdomain registration"`
	ResolverId         uint64    `pg:",comment:Resolver's address id from main indexer"`
	Subdomain          string    `pg:",unique,comment:Subdomain string"`

	Resolver Address `pg:"-" hasura:"table:address,field:resolver_id,remote_field:id,type:oto,name:resolver"`
}

// TableName -
func (Subdomain) TableName() string {
	return "subdomain"
}
