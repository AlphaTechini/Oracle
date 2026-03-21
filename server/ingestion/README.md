# Go Fetcher Node: Multi-Source Data Ingestion

The Fetcher Node is the backbone of the Oracle's data ingestion layer. Each node runs independently, fetching prices from multiple external APIs, calculating a median, and signing the result to prove its authenticity.

## Architectural Decisions & Tradeoffs

### 1. Hybrid Input: WebSockets + REST
I've designed the fetcher to be as fast as possible without sacrificing reliability.
- **Decision:** Use a persistent WebSocket for active markets (Binance, Bybit) but include a REST fallback.
- **Tradeoff:** Managing WebSocket connections in Go requires more boilerplate (goroutines, heartbeats), but it significantly reduces the "latency lag" between the exchange and the oracle.

### 2. Internal Median Calculation
Before signing, each node calculates its own median price from all 5 sources.
- **Why:** If one API is down or reporting a "fat finger" error, the median ensures the node's reported price remains accurate. It's the first line of defense against data corruption.

### 3. ECDSA Signing (`go-ethereum/crypto`)
Every reported price is cryptographically signed by the node's private key.
- **Decision:** Use standard ECDSA (secp256k1).
- **Why:** This allows the Solidity contract to verify exactly which node provided which data, preventing any "man-in-the-middle" attacks between the fetcher and the aggregator.

## Setup & Running

Individual nodes can be configured using environment variables:
- `PRIVATE_KEY`: The node's Ethereum private key.
- `NODE_ADDRESS`: The corresponding public address.
- `RPC_URL`: Connection to the Ethereum network.

```powershell
go run main.go
```
