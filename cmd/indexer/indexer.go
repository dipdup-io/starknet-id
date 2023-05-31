package main

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dipdup-io/starknet-go-api/pkg/data"
	starknetid "github.com/dipdup-io/starknet-id/internal/starknet-id"
	"github.com/dipdup-io/starknet-id/internal/storage/postgres"
	"github.com/dipdup-io/starknet-indexer/pkg/grpc/pb"
	"github.com/dipdup-net/indexer-sdk/pkg/modules"
	"github.com/goccy/go-json"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// variables
var (
	ZeroAddress = data.Felt("0x0").Bytes()
)

// input name
const (
	InputName = "input"
)

// EventHandler -
type EventHandler func(ctx context.Context, event *pb.Event) error

// Indexer -
type Indexer struct {
	storage       postgres.Storage
	input         *modules.Input
	blockCtx      *BlockContext
	eventHandlers map[string]EventHandler
	store         Store

	wg *sync.WaitGroup
}

// NewIndexer -
func NewIndexer(pg postgres.Storage) *Indexer {
	indexer := &Indexer{
		storage:  pg,
		input:    modules.NewInput(InputName),
		blockCtx: newBlockContext(),
		store:    NewStore(pg),
		wg:       new(sync.WaitGroup),
	}
	indexer.eventHandlers = map[string]EventHandler{
		starknetid.EventTransfer:              indexer.parseTransferEvent,
		starknetid.EventAddrToDomainUpdate:    indexer.parseAddrToDomainUpdate,
		starknetid.EventDomainToAddrUpdate:    indexer.parseDomainToAddrUpdate,
		starknetid.EventStarknetIdUpdate:      indexer.parseStarknetIdUpdate,
		starknetid.EventDomainTransfer:        indexer.parseTransferDomain,
		starknetid.EventOnInftEquipped:        nil,
		starknetid.EventResetSubdomainsUpdate: nil,
		starknetid.EventVerifierDataUpdate:    nil,
	}

	return indexer
}

// Start -
func (indexer *Indexer) Start(ctx context.Context) {
	if err := indexer.init(ctx); err != nil {
		log.Err(err).Msg("state initialization")
		return
	}

	indexer.wg.Add(1)
	go indexer.listen(ctx)
}

// Name -
func (indexer *Indexer) Name() string {
	return "starknet_id_indexer"
}

// Height -
func (indexer *Indexer) Height() uint64 {
	return indexer.blockCtx.state.LastHeight
}

func (indexer *Indexer) init(ctx context.Context) error {
	state, err := indexer.storage.State.ByName(ctx, indexer.Name())
	switch {
	case err == nil:
		indexer.blockCtx.state = &state
		return nil
	case indexer.storage.State.IsNoRows(err):
		indexer.blockCtx.state.Name = indexer.Name()
		return indexer.storage.State.Save(ctx, indexer.blockCtx.state)
	default:
		return err
	}
}

// Input - returns input by name
func (indexer *Indexer) Input(name string) (*modules.Input, error) {
	switch name {
	case InputName:
		return indexer.input, nil
	default:
		return nil, errors.Wrap(modules.ErrUnknownInput, name)
	}
}

func (indexer *Indexer) listen(ctx context.Context) {
	defer indexer.wg.Done()

	input, err := indexer.Input(InputName)
	if err != nil {
		log.Err(err).Msg("unknown input")
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-input.Listen():
			if !ok {
				continue
			}

			switch typ := msg.(type) {
			case *pb.Subscription:
				switch {
				case typ.GetEndOfBlock():
					log.Info().
						Uint64("subscription", typ.Response.Id).
						Msg("end of block")

					if err := indexer.store.Save(ctx, indexer.blockCtx); err != nil {
						log.Err(err).Msg("saving data")
					}
					indexer.blockCtx.reset()
				case typ.Event != nil:
					if err := indexer.parseEvent(ctx, typ.Event); err != nil {
						log.Err(err).Msg("event parsing")
					}

					log.Info().
						Str("name", typ.Event.Name).
						Uint64("height", typ.Event.Height).
						Uint64("time", typ.Event.Time).
						Uint64("id", typ.Event.Id).
						Uint64("subscription", typ.Response.Id).
						Str("contract", fmt.Sprintf("0x%x", typ.Event.Contract)).
						Msg("event")

					if indexer.blockCtx.state.LastHeight < typ.Event.Height {
						indexer.blockCtx.state.LastHeight = typ.Event.Height
						indexer.blockCtx.state.LastTime = time.Unix(int64(typ.Event.Time), 0).UTC()
					}
				}
			default:
				log.Info().Msgf("unknown message: %T", typ)
			}
		}
	}
}

// Output - returns output by name
func (indexer *Indexer) Output(name string) (*modules.Output, error) {
	return nil, errors.Wrap(modules.ErrUnknownOutput, name)
}

// AttachTo - attach input to output with name
func (indexer *Indexer) AttachTo(name string, input *modules.Input) error {
	output, err := indexer.Output(name)
	if err != nil {
		return err
	}
	output.Attach(input)
	return nil
}

// Close - gracefully stops module
func (indexer *Indexer) Close() error {
	indexer.wg.Wait()

	if err := indexer.input.Close(); err != nil {
		return err
	}

	return nil
}

func (indexer *Indexer) parseEvent(ctx context.Context, event *pb.Event) error {
	handler, ok := indexer.eventHandlers[event.Name]
	if !ok {
		return errors.Errorf("unknown event handler: %s", event.Name)
	}
	if handler == nil {
		return nil
	}

	return handler(ctx, event)
}

func (indexer *Indexer) parseTransferEvent(ctx context.Context, event *pb.Event) error {
	var data starknetid.Transfer
	if err := json.Unmarshal(event.ParsedData, &data); err != nil {
		return errors.Wrap(err, "parsing data")
	}

	switch {
	case bytes.Equal(data.From.Bytes(), ZeroAddress):
		indexer.blockCtx.addMintedStarknetId(data)
	case bytes.Equal(data.To.Bytes(), ZeroAddress):
		indexer.blockCtx.addBurnedStarknetId(data)
	default:
		indexer.blockCtx.addTransferedStarknetId(data)
	}

	return nil
}

func (indexer *Indexer) parseAddrToDomainUpdate(ctx context.Context, event *pb.Event) error {
	var data starknetid.AddrToDomainUpdate
	if err := json.Unmarshal(event.ParsedData, &data); err != nil {
		return errors.Wrap(err, "parsing data")
	}
	if err := indexer.blockCtx.addDomains(data.Domain, data.Address); err != nil {
		return errors.Wrap(err, "decoding domain")
	}
	return nil
}

func (indexer *Indexer) parseDomainToAddrUpdate(ctx context.Context, event *pb.Event) error {
	var data starknetid.DomainToAddrUpdate
	if err := json.Unmarshal(event.ParsedData, &data); err != nil {
		return errors.Wrap(err, "parsing data")
	}
	if err := indexer.blockCtx.addDomains(data.Domain, data.Address); err != nil {
		return errors.Wrap(err, "decoding domain")
	}
	return nil
}

func (indexer *Indexer) parseStarknetIdUpdate(ctx context.Context, event *pb.Event) error {
	var data starknetid.StarknetIdUpdate
	if err := json.Unmarshal(event.ParsedData, &data); err != nil {
		return errors.Wrap(err, "parsing data")
	}
	if err := indexer.blockCtx.applyStaknetIdUpdate(data); err != nil {
		return errors.Wrap(err, "decoding domain")
	}
	return nil
}

func (indexer *Indexer) parseTransferDomain(ctx context.Context, event *pb.Event) error {
	var data starknetid.DomainTransfer
	if err := json.Unmarshal(event.ParsedData, &data); err != nil {
		return errors.Wrap(err, "parsing data")
	}
	if err := indexer.blockCtx.applyDomainTransfer(data); err != nil {
		return errors.Wrap(err, "decoding domain")
	}
	return nil
}
