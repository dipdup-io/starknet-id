package main

import (
	"context"
	"math/big"
	"strings"
	"sync"
	"time"

	starknetid "github.com/dipdup-io/starknet-id/internal/starknet-id"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

// Tester -
type Tester struct {
	starknetIdApi starknetid.Api
	graphQlApi    GraphQlApi
	offset        int
	wg            *sync.WaitGroup
}

// NewTester -
func NewTester(cfg Config) *Tester {
	tester := new(Tester)
	tester.wg = new(sync.WaitGroup)
	tester.starknetIdApi = starknetid.NewApi(cfg.StarknetId)
	tester.graphQlApi = NewGraphQlApi(cfg.GraphQlApi)
	tester.offset = cfg.Start
	return tester
}

// Start -
func (t *Tester) Start(ctx context.Context) {
	t.wg.Add(1)
	go t.work(ctx)
}

func (t *Tester) work(ctx context.Context) {
	defer t.wg.Done()

	var (
		limit = 100
		end   bool
	)

	for !end {
		select {
		case <-ctx.Done():
			return
		default:
			domains, err := t.getActualDomains(ctx, limit, t.offset)
			if err != nil {
				log.Err(err).Msg("graphql")
				continue
			}

			for i := range domains {
				select {
				case <-ctx.Done():
					return
				default:
				}

				log.Info().Str("domain", domains[i].Domain).Msg("check...")
				resp, err := t.getDomain(ctx, domains[i].Domain)
				if err != nil {
					log.Err(err).Str("domain", domains[i].Domain).Msg("starknet id api")
					continue
				}
				if resp.Addr == "" {
					log.Error().Str("domain", domains[i].Domain).Msg("unknown domain")
					continue
				}

				siAddr, err := decimal.NewFromString(resp.Addr)
				if err != nil {
					log.Error().Str("starknet_id", resp.Addr).Msg("can't decode address")
					continue
				}
				addVal, ok := big.NewInt(0).SetString(strings.TrimPrefix(domains[i].Address, "\\x"), 16)
				if !ok {
					log.Error().Str("graphql", domains[i].Address).Msg("can't decode address")
					continue
				}
				gqAddr := decimal.NewFromBigInt(addVal, 0)

				if !gqAddr.Equal(siAddr) {
					log.Error().Str("starknet_id", siAddr.String()).Str("graphql", gqAddr.String()).Msg("unequal")
				}
			}

			t.offset += len(domains)
			end = len(domains) < limit
		}
	}
}

func (t *Tester) getActualDomains(ctx context.Context, limit, offset int) ([]ActualDomain, error) {
	gqCtx, cancelGQ := context.WithTimeout(ctx, time.Second*10)
	defer cancelGQ()

	return t.graphQlApi.ActualDomains(gqCtx, limit, offset)
}

func (t *Tester) getDomain(ctx context.Context, name string) (starknetid.DomainToAddrResponse, error) {
	gqCtx, cancelGQ := context.WithTimeout(ctx, time.Second*10)
	defer cancelGQ()

	return t.starknetIdApi.DomainToAddress(gqCtx, name)
}

// Close -
func (t *Tester) Close() error {
	t.wg.Wait()
	return nil
}
