-include .env
export $(shell sed 's/=.*//' .env)

starknet-id:
	cd cmd/starknet-id && go run . -c ../../build/dipdup.yml

build:
	docker-compose up -d -- build

lint:
	golangci-lint run

test:
	go test ./...