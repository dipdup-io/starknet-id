package storage

import (
	"context"
	"time"

	"github.com/dipdup-net/indexer-sdk/pkg/storage"
	"github.com/uptrace/bun"
)

// ISubdomain -
type ISubdomain interface {
	storage.Table[*Subdomain]

	GetByResolverId(ctx context.Context, resolverId uint64) (result Subdomain, err error)
}

// Subdomain -
type Subdomain struct {
	bun.BaseModel `bun:"subdomain" comment:"Subdomain's table"`

	Id                 uint64    `bun:"id,pk,autoincrement"                                    comment:"Unique internal identity"`
	RegistrationHeight uint64    `comment:"Height of first event about subdomain registration"`
	RegistrationDate   time.Time `comment:"Date of first event about subdomain registration"`
	ResolverId         uint64    `comment:"Resolver's address id from main indexer"`
	Subdomain          string    `bun:",unique"                                                comment:"Subdomain string"`

	Resolver Address `bun:"-" hasura:"table:address,field:resolver_id,remote_field:id,type:oto,name:resolver"`
}

// TableName -
func (Subdomain) TableName() string {
	return "subdomain"
}
