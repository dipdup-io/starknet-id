package main

import (
	"context"
	"fmt"
	"time"

	"github.com/dipdup-io/starknet-id/internal/storage"
	"github.com/karlseguin/ccache/v2"
)

// Cache -
type Cache struct {
	*ccache.Cache

	subdomains storage.ISubdomain
}

// NewCache -
func NewCache(subdomains storage.ISubdomain) *Cache {
	return &Cache{
		Cache:      ccache.New(ccache.Configure().MaxSize(1000)),
		subdomains: subdomains,
	}
}

// SetSubdomain -
func (c *Cache) SetSubdomain(resolverId uint64, domain string) {
	c.Set(fmt.Sprintf("subdomain:%d", resolverId), domain+"."+rootDomain, time.Hour)
}

const rootDomain = "stark"

// GetSubdomain -
func (c *Cache) GetSubdomain(ctx context.Context, resolverId uint64) (string, error) {
	value, err := c.Fetch(fmt.Sprintf("subdomain:%d", resolverId), time.Hour, func() (interface{}, error) {
		sd, err := c.subdomains.GetByResolverId(ctx, resolverId)
		if err != nil {
			if c.subdomains.IsNoRows(err) {
				return rootDomain, nil
			}
			return "", err
		}
		return sd.Subdomain + "." + rootDomain, nil
	})
	if err != nil {
		return "", err
	}

	return value.Value().(string), nil
}
