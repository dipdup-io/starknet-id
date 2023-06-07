# ---------------------------------------------------------------------
#  The first stage container, for building the application
# ---------------------------------------------------------------------
FROM golang:1.19-alpine as builder

ENV CGO_ENABLED=0
ENV GO111MODULE=on
ENV GOOS=linux

RUN apk --no-cache add ca-certificates
RUN apk add --update git musl-dev gcc build-base

RUN mkdir -p $GOPATH/src/github.com/dipdup-io/starknet-id/

COPY ./go.* $GOPATH/src/github.com/dipdup-io/starknet-id/
WORKDIR $GOPATH/src/github.com/dipdup-io/starknet-id
RUN go mod download

COPY cmd/starknet-id cmd/starknet-id
COPY internal internal

WORKDIR $GOPATH/src/github.com/dipdup-io/starknet-id/cmd/starknet-id/
RUN go build -a -o /go/bin/starknet-id .

# ---------------------------------------------------------------------
#  The second stage container, for running the application
# ---------------------------------------------------------------------
FROM scratch

WORKDIR /app/starknet-id/

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/bin/starknet-id /go/bin/starknet-id
COPY ./build/dipdup.yml ./

ENTRYPOINT ["/go/bin/starknet-id", "-c", "dipdup.yml"]