# Ingestion Fetcher Node

This component is responsible for retrieving real-time cryptocurrency price data from five external APIs concurrently, calculating an internal median, signing it, and sending it to the central Aggregator node.

## Architectural Tradeoffs & Decisions

1. **Concurrency Model (`sync.WaitGroup` vs Sequential):**
   - **Previous:** The previous implementation hit all 5 REST APIs sequentially. If one API hung, the node missed the strict 3-second DON SLA.
   - **Current:** We use standard library `sync.WaitGroup` intertwined with a strict `context.WithTimeout(2 * time.Second)`. All 5 APIs are hit simultaneously. If a single endpoint errors out or delays, it fails safely without dragging down the consensus window.

2. **Communication Protocol (`net/rpc` vs gRPC/HTTP):**
   - **Decision:** Shifted to standard library `net/rpc` over TCP. 
   - **Tradeoff:** While gRPC is the industry standard, it requires `protoc` cross-compilation toolchains which can be hostile to local Windows environments. Raw HTTP JSON incurs heavy parsing overhead. `net/rpc` provides identical binary performance natively out-of-the-box using Gob encoding, completely avoiding third-party protobuf dependencies while maintaining high performance.

3. **Event Ingestion (WebSockets vs Polling):**
   - **Decision:** Replaced the 10-second sleep polling model with EVM WebSocket Subscriptions (`ethclient.SubscribeFilterLogs`).
   - **Tradeoff:** WebSockets require persistent connections and can be fragile if the internal node drops. However, polling adds unacceptable artificial latency (up to 10s delay). WSS allows the node to begin fetching data the precise millisecond a client requests it on-chain.
