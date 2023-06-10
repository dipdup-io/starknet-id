package main

import (
	"context"
	"crypto/rand"
	"math/big"
	"sort"
	"strings"
	"sync"
	"time"

	starknetid "github.com/dipdup-io/starknet-id/internal/starknet-id"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

type testPart struct {
	offset int
	limit  int
}

func randomParts(count, end, limit int) ([]testPart, error) {
	threasholds := make([]int, count)
	for i := 0; i < count; i++ {
		value, err := rand.Int(rand.Reader, big.NewInt(int64(end)))
		if err != nil {
			return nil, err
		}
		threasholds[i] = int(value.Int64())
	}

	sort.Ints(threasholds)

	parts := make([]testPart, count)
	for i := 0; i < count; i++ {
		parts[i].limit = limit
		parts[i].offset = threasholds[i]
	}
	return parts, nil
}

// Tester -
type Tester struct {
	starknetIdApi starknetid.Api
	graphQlApi    GraphQlApi
	parts         []testPart
	wg            *sync.WaitGroup
}

// NewTester -
func NewTester(cfg Config) (*Tester, error) {
	tester := new(Tester)
	tester.wg = new(sync.WaitGroup)
	tester.starknetIdApi = starknetid.NewApi(cfg.StarknetId)
	tester.graphQlApi = NewGraphQlApi(cfg.GraphQlApi)

	parts, err := randomParts(cfg.Parts, 200000, 100)
	if err != nil {
		return nil, err
	}
	tester.parts = parts

	return tester, nil
}

// Start -
func (t *Tester) Start(ctx context.Context) {
	for i := range t.parts {
		t.wg.Add(1)
		go t.work(ctx, t.parts[i])
	}
}

func (t *Tester) work(ctx context.Context, part testPart) {
	defer t.wg.Done()

	var (
		processed = 0
		failed    = 0
	)

	domains, err := t.getActualDomains(ctx, part.limit, part.offset)
	if err != nil {
		log.Err(err).Msg("graphql")
		return
	}

	for i := range domains {
		select {
		case <-ctx.Done():
			log.Info().Int("failed", failed).Int("processed", processed).Msg("report")
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
		domainAddr := strings.TrimPrefix(domains[i].Address, "\\x")
		gqAddr := decimal.NewFromInt(0)
		if domainAddr != "" {
			addVal, ok := big.NewInt(0).SetString(domainAddr, 16)
			if !ok {
				log.Error().Str("graphql", domains[i].Address).Msg("can't decode address")
				continue
			}
			gqAddr = decimal.NewFromBigInt(addVal, 0)
		}

		if !gqAddr.Equal(siAddr) {
			log.Error().Str("starknet_id", siAddr.String()).Str("graphql", gqAddr.String()).Msg("unequal")
			failed += 1
		}

		processed += 1
	}

	log.Info().Int("failed", failed).Int("processed", processed).Msg("report")
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
