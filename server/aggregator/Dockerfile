FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o aggregator .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/aggregator .

ENV HTTP_RPC_URL="http://hardhat:8545"
ENV ORACLE_CONTRACT_ADDRESS="0x5FbDB2315678afecb367f032d93F642f64180aa3"

EXPOSE 4000
CMD ["./aggregator"]
