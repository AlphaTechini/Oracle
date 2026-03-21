FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o fetcher .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/fetcher .

# Default .env overrides these
ENV WS_RPC_URL="ws://hardhat:8545"
ENV AGGREGATOR_RPC_URL="aggregator:4000"

CMD ["./fetcher"]
