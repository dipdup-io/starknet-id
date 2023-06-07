package main

import (
	"bytes"
	"context"
	"sync"

	"github.com/goccy/go-json"

	starknetid "github.com/dipdup-io/starknet-id/internal/starknet-id"
	"github.com/dipdup-io/starknet-id/internal/storage"
	"github.com/dipdup-io/starknet-id/internal/storage/postgres"
	"github.com/dipdup-io/starknet-indexer/pkg/grpc/pb"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// EventHandler -
type EventHandler func(ctx context.Context, blockCtx *BlockContext, event *pb.Event) error

// Channel -
type Channel struct {
	name          string
	blockCtx      *BlockContext
	storage       postgres.Storage
	eventHandlers map[string]EventHandler
	store         Store
	failed        bool
	ch            chan *pb.Subscription
	wg            *sync.WaitGroup
}

// NewChannel -
func NewChannel(name string, pg postgres.Storage) Channel {
	ch := Channel{
		name:     name,
		storage:  pg,
		blockCtx: newBlockContext(),
		store:    NewStore(pg),
		ch:       make(chan *pb.Subscription, 1024*1024),
		wg:       new(sync.WaitGroup),
	}

	ch.eventHandlers = map[string]EventHandler{
		starknetid.EventTransfer:              ch.parseTransferEvent,
		starknetid.EventAddrToDomainUpdate:    ch.parseAddrToDomainUpdate,
		starknetid.EventDomainToAddrUpdate:    ch.parseDomainToAddrUpdate,
		starknetid.EventStarknetIdUpdate:      ch.parseStarknetIdUpdate,
		starknetid.EventDomainTransfer:        ch.parseTransferDomain,
		starknetid.EventVerifierDataUpdate:    ch.parseVerifierDataUpdate,
		starknetid.EventOnInftEquipped:        nil,
		starknetid.EventResetSubdomainsUpdate: nil,
	}

	return ch
}

// AddEvent -
func (channel Channel) Add(msg *pb.Subscription) {
	channel.ch <- msg
}

func (channel Channel) Start(ctx context.Context) {
	channel.wg.Add(1)
	go channel.listen(ctx)
}

func (channel Channel) listen(ctx context.Context) {
	defer channel.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-channel.ch:
			if channel.failed {
				continue
			}
			response := msg.GetResponse()
			switch {
			case msg.EndOfBlock != nil:
				log.Info().
					Uint64("subscription", response.GetId()).
					Uint64("height", msg.EndOfBlock.Height).
					Str("channel", channel.name).
					Msg("end of block")

				channel.blockCtx.updateState(channel.name, msg.EndOfBlock.Height)
				if err := channel.store.Save(ctx, channel.blockCtx); err != nil {
					log.Err(err).Msg("saving data")
					channel.failed = true
				}

			case msg.Event != nil:

				if err := channel.parseEvent(ctx, msg.Event); err != nil {
					log.Err(err).Msg("event parsing")
				}

				log.Debug().
					Str("name", msg.Event.Name).
					Uint64("height", msg.Event.Height).
					Uint64("time", msg.Event.Time).
					Uint64("id", msg.Event.Id).
					Uint64("subscription", msg.Response.Id).
					Str("channel", channel.name).
					Msg("new event")

			case msg.Address != nil:
				if err := channel.parseAddress(msg.Address); err != nil {
					log.Err(err).Msg("event parsing")
				}
				log.Debug().
					Uint64("height", msg.Address.Height).
					Uint64("id", msg.Address.Id).
					Uint64("subscription", msg.Response.Id).
					Str("channel", channel.name).
					Msg("new address")
			}
		}
	}
}

// Close -
func (channel Channel) Close() error {
	channel.wg.Wait()

	close(channel.ch)
	return nil
}

// Name -
func (channel Channel) Name() string {
	return channel.name
}

// State -
func (channel Channel) State() *storage.State {
	return channel.blockCtx.state
}

func (channel Channel) parseAddress(msg *pb.Address) error {
	channel.blockCtx.addAddress(msg)
	return nil
}

func (channel Channel) parseEvent(ctx context.Context, event *pb.Event) error {
	handler, ok := channel.eventHandlers[event.Name]
	if !ok {
		return errors.Errorf("unknown event handler: %s", event.Name)
	}
	if handler == nil {
		return nil
	}

	return handler(ctx, channel.blockCtx, event)
}

func (channel Channel) parseTransferEvent(ctx context.Context, blockCtx *BlockContext, event *pb.Event) error {
	var data starknetid.Transfer
	if err := json.Unmarshal(event.ParsedData, &data); err != nil {
		return errors.Wrap(err, "parsing data")
	}

	switch {
	case bytes.Equal(data.From.Bytes(), ZeroAddress):
		return blockCtx.addMintedStarknetId(ctx, channel.storage.Addresses, data)
	case bytes.Equal(data.To.Bytes(), ZeroAddress):
		return blockCtx.addBurnedStarknetId(ctx, channel.storage.Addresses, data)
	default:
		return blockCtx.addTransferedStarknetId(ctx, channel.storage.Addresses, data)
	}
}

func (channel Channel) parseAddrToDomainUpdate(ctx context.Context, blockCtx *BlockContext, event *pb.Event) error {
	var data starknetid.AddrToDomainUpdate
	if err := json.Unmarshal(event.ParsedData, &data); err != nil {
		return errors.Wrap(err, "parsing data")
	}
	return blockCtx.addDomains(ctx, channel.storage.Addresses, data.Domain, data.Address)
}

func (channel Channel) parseDomainToAddrUpdate(ctx context.Context, blockCtx *BlockContext, event *pb.Event) error {
	var data starknetid.DomainToAddrUpdate
	if err := json.Unmarshal(event.ParsedData, &data); err != nil {
		return errors.Wrap(err, "parsing data")
	}
	return blockCtx.addDomains(ctx, channel.storage.Addresses, data.Domain, data.Address)
}

func (channel Channel) parseStarknetIdUpdate(ctx context.Context, blockCtx *BlockContext, event *pb.Event) error {
	var data starknetid.StarknetIdUpdate
	if err := json.Unmarshal(event.ParsedData, &data); err != nil {
		return errors.Wrap(err, "parsing data")
	}
	return blockCtx.applyStaknetIdUpdate(data)
}

func (channel Channel) parseTransferDomain(ctx context.Context, blockCtx *BlockContext, event *pb.Event) error {
	var data starknetid.DomainTransfer
	if err := json.Unmarshal(event.ParsedData, &data); err != nil {
		return errors.Wrap(err, "parsing data")
	}
	return blockCtx.applyDomainTransfer(data)
}

func (channel Channel) parseVerifierDataUpdate(ctx context.Context, blockCtx *BlockContext, event *pb.Event) error {
	var data starknetid.VerifierDataUpdate
	if err := json.Unmarshal(event.ParsedData, &data); err != nil {
		return errors.Wrap(err, "parsing data")
	}

	return blockCtx.addField(data)
}
