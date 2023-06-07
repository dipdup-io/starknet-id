CREATE OR REPLACE VIEW actual_domains AS
SELECT
    domain.id,
    domain.address_hash as address,
    domain.domain,
    domain.expiry,
    starknet_id.starknet_id,
    starknet_id.owner_address
FROM
    domain
left join starknet_id on owner = starknet_id.starknet_id
where expiry > current_timestamp;