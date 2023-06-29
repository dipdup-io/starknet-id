# Starknet ID indexer

A DipDup Vertical component intended to enrich account data with [Starknet ID](https://www.starknet.id/) domains and identities.

## Features

* Domains and subdomains (currently Braavos and Xplorer) with names decoded
* Actual domains view: returns all non-expired domains
* Starknet ID owner and metadata fields (name + namespace + raw value)

## Public instances

Public deployments with reasonable rate limits are available for testing and prototyping:
* [Starknet mainnet](https://play.dipdup.io/?endpoint=https://starknet-id.dipdup.net/v1/graphql) `https://starknet-id.dipdup.net/v1/graphql` 

## Usage examples

### Lookup Starknet domains by address

```graphql
query DomainsByAddress {
  domain(
    where: {address_hash: {_eq: "\\x072d4f3fa4661228ed0c9872007fc7e12a581e000fad7b8f3e3e5bf9e6133207"}}
  ) {
    domain
    expiry
    owner
    address_hash
  }
}
```

### Resolve contract address by domain

```graphql
query ResolveAddress {
  domain(where: {domain: {_eq: "uniswap.stark"}}) {
    domain
    address_hash
    expiry
    owner
  }
}
```

### Get domain records

```grapgql
query GetDomainRecords {
  domain(where: {domain: {_regex: ".*.fricoben.stark"}}) {
    domain
    address_hash
    expiry
    owner
  }
}
```

### Query Starknet IDs by owner address

```graphql
query StarknetIdsByOwner {
  starknet_id(
    where: {owner_address: {_eq: "\\x049792ef13b7ecd3de26dc7ac1787f04f0e8c4658b877a75d93151ac903308b1"}}
  ) {
    fields {
      name
      namespace
      value
    }
    starknet_id
    owner_address
  }
}

```

## About

DipDup Vertical for Starknet is a federated API including the following services:

- [x] Generic Starknet indexer
- [x] Starknet ID indexer
- [ ] Token metadata resolver
- [ ] Aggregated market data
- [ ] Chain/dapp/contract analytics
- [ ] Starknet search engine

Project is supported by Starkware and Starknet Foundation via [OnlyDust platform](https://app.onlydust.xyz/projects/e1b6d080-7f15-4531-9259-10c3dae26848)

