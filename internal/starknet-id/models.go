package starknetid

import "github.com/dipdup-io/starknet-go-api/pkg/data"

// addresses
const (
	AddressStarknetId = data.Felt("0x05dbdedc203e92749e2e746e2d40a768d966bd243df04a6b712e222bc040a9af")
	AddressNaming     = data.Felt("0x06ac597f8116f886fa1c97a23fa4e08299975ecaf6b598873ca6792b9bbfb678")
	AddressBraavos    = data.Felt("0x03448896d4a0df143f98c9eeccc7e279bf3c2008bda2ad2759f5b20ed263585f")
)

// event names
const (
	EventTransfer               = "Transfer"
	EventVerifierDataUpdate     = "VerifierDataUpdate"
	EventOnInftEquipped         = "on_inft_equipped"
	EventDomainToAddrUpdate     = "domain_to_addr_update"
	EventAddrToDomainUpdate     = "addr_to_domain_update"
	EventStarknetIdUpdate       = "starknet_id_update"
	EventDomainTransfer         = "domain_transfer"
	EventResetSubdomainsUpdate  = "reset_subdomains_update"
	EventDomainToResolverUpdate = "domain_to_resolver_update"
)

// Transfer -
type Transfer struct {
	From    data.Felt    `json:"from_"`
	To      data.Felt    `json:"to"`
	TokenId data.Uint256 `json:"tokenId"`
}

// VerifierDataUpdate -
type VerifierDataUpdate struct {
	StarknetId data.Felt `json:"starknet_id"`
	Field      data.Felt `json:"field"`
	Data       data.Felt `json:"data"`
	Verifier   data.Felt `json:"verifier"`
}

// OnInftEquipped -
type OnInftEquipped struct {
	InftContract data.Felt `json:"inft_contract"`
	InftId       data.Felt `json:"inft_id"`
	StarknetId   data.Felt `json:"starknet_id"`
}

// DomainToAddrUpdate -
type DomainToAddrUpdate struct {
	DomainLen data.Felt   `json:"domain_len"`
	Domain    []data.Felt `json:"domain"`
	Address   data.Felt   `json:"address"`
}

// AddrToDomainUpdate -
type AddrToDomainUpdate struct {
	DomainLen data.Felt   `json:"domain_len"`
	Domain    []data.Felt `json:"domain"`
	Address   data.Felt   `json:"address"`
}

// StarknetIdUpdate -
type StarknetIdUpdate struct {
	DomainLen data.Felt   `json:"domain_len"`
	Domain    []data.Felt `json:"domain"`
	Owner     data.Felt   `json:"owner"`
	Expiry    data.Felt   `json:"expiry"`
}

// DomainTransfer -
type DomainTransfer struct {
	DomainLen data.Felt   `json:"domain_len"`
	Domain    []data.Felt `json:"domain"`
	PrevOwner data.Felt   `json:"prev_owner"`
	NewOwner  data.Felt   `json:"new_owner"`
}

// ResetSubdomainsUpdate -
type ResetSubdomainsUpdate struct {
	DomainLen data.Felt   `json:"domain_len"`
	Domain    []data.Felt `json:"domain"`
}

// DomainToResolverUpdate -
type DomainToResolverUpdate struct {
	Domain    []data.Felt `json:"domain"`
	Resolver  data.Felt   `json:"resolver"`
	DomainLen data.Felt   `json:"domain_len"`
}
