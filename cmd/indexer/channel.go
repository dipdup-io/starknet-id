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
type EventHandler func(blockCtx *BlockContext, event *pb.Event) error

// Channel -
type Channel struct {
	name          string
	blockCtx      *BlockContext
	eventHandlers map[string]EventHandler
	store         Store
	ch            chan *pb.Subscription
	wg            *sync.WaitGroup
}

// NewChannel -
func NewChannel(name string, pg postgres.Storage) Channel {
	ch := Channel{
		name:     name,
		blockCtx: newBlockContext(),
		store:    NewStore(pg),
		ch:       make(chan *pb.Subscription, 1024),
		wg:       new(sync.WaitGroup),
	}

	ch.eventHandlers = map[string]EventHandler{
		starknetid.EventTransfer:              parseTransferEvent,
		starknetid.EventAddrToDomainUpdate:    parseAddrToDomainUpdate,
		starknetid.EventDomainToAddrUpdate:    parseDomainToAddrUpdate,
		starknetid.EventStarknetIdUpdate:      parseStarknetIdUpdate,
		starknetid.EventDomainTransfer:        parseTransferDomain,
		starknetid.EventVerifierDataUpdate:    parseVerifierDataUpdate,
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
			switch {
			case msg.GetEndOfBlock():
				log.Info().
					Uint64("subscription", msg.Response.Id).
					Str("channel", channel.name).
					Msg("end of block")

				if err := channel.store.Save(ctx, channel.blockCtx); err != nil {
					log.Err(err).Msg("saving data")
				}
			case msg.Event != nil:
				if err := channel.parseEvent(msg.Event); err != nil {
					log.Err(err).Msg("event parsing")
				}

				log.Debug().
					Str("name", msg.Event.Name).
					Uint64("height", msg.Event.Height).
					Uint64("time", msg.Event.Time).
					Uint64("id", msg.Event.Id).
					Uint64("subscription", msg.Response.Id).
					Str("channel", channel.name).
					Msg("event")

				if msg.Event.Height > channel.blockCtx.state.LastHeight {
					channel.blockCtx.updateState(channel.name, msg.Event.Height, msg.Event.Time)
					if err := channel.store.Save(ctx, channel.blockCtx); err != nil {
						log.Err(err).Msg("saving data")
					}
				}
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

func (channel Channel) parseEvent(event *pb.Event) error {
	handler, ok := channel.eventHandlers[event.Name]
	if !ok {
		return errors.Errorf("unknown event handler: %s", event.Name)
	}
	if handler == nil {
		return nil
	}

	return handler(channel.blockCtx, event)
}

func parseTransferEvent(blockCtx *BlockContext, event *pb.Event) error {
	var data starknetid.Transfer
	if err := json.Unmarshal(event.ParsedData, &data); err != nil {
		return errors.Wrap(err, "parsing data")
	}

	switch {
	case bytes.Equal(data.From.Bytes(), ZeroAddress):
		return blockCtx.addMintedStarknetId(data)
	case bytes.Equal(data.To.Bytes(), ZeroAddress):
		return blockCtx.addBurnedStarknetId(data)
	default:
		return blockCtx.addTransferedStarknetId(data)
	}
}

func parseAddrToDomainUpdate(blockCtx *BlockContext, event *pb.Event) error {
	var data starknetid.AddrToDomainUpdate
	if err := json.Unmarshal(event.ParsedData, &data); err != nil {
		return errors.Wrap(err, "parsing data")
	}
	return blockCtx.addDomains(data.Domain, data.Address)
}

func parseDomainToAddrUpdate(blockCtx *BlockContext, event *pb.Event) error {
	var data starknetid.DomainToAddrUpdate
	if err := json.Unmarshal(event.ParsedData, &data); err != nil {
		return errors.Wrap(err, "parsing data")
	}
	return blockCtx.addDomains(data.Domain, data.Address)
}

func parseStarknetIdUpdate(blockCtx *BlockContext, event *pb.Event) error {
	var data starknetid.StarknetIdUpdate
	if err := json.Unmarshal(event.ParsedData, &data); err != nil {
		return errors.Wrap(err, "parsing data")
	}
	return blockCtx.applyStaknetIdUpdate(data)
}

func parseTransferDomain(blockCtx *BlockContext, event *pb.Event) error {
	var data starknetid.DomainTransfer
	if err := json.Unmarshal(event.ParsedData, &data); err != nil {
		return errors.Wrap(err, "parsing data")
	}
	return blockCtx.applyDomainTransfer(data)
}

func parseVerifierDataUpdate(blockCtx *BlockContext, event *pb.Event) error {
	var data starknetid.VerifierDataUpdate
	if err := json.Unmarshal(event.ParsedData, &data); err != nil {
		return errors.Wrap(err, "parsing data")
	}

	return blockCtx.addField(data)
}
