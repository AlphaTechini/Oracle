# Go Aggregator: Network Consensus & Submission

The Aggregator is the router of the Oracle Network. It collects signed reports from multiple Fetcher Nodes, calculates the network-wide consensus, and submits the final transaction to the Solidity registry.

## Architectural Decisions & Tradeoffs

### 1. Simple Deviation Filtering (The 0.2% Rule)
I've implemented a strict filtering rule to ensure high-quality data.
- **Decision:** Any node whose price deviates by more than 0.2% from the network median is marked as "Slashed".
- **Tradeoff:** This might be tight for extremely volatile assets, but for major pairs (BTC/ETH), it effectively filters out noise and bad actors.

### 2. Aggregator Rotation Logic
The aggregator is not a single fixed server. 
- **Decision:** The Solidity contract deterministically picks which node should act as the aggregator for a specific `requestId`.
- **Tradeoff:** This requires nodes to monitor the chain and "know" when it's their turn to aggregate. It adds some complexity to the Go code but removes the central point of failure.

### 3. Replacing the Node.js Dispatcher
I moved the aggregation and submission logic from Node.js to Go.
- **Why:** Go's concurrency model (goroutines/channels) is better suited for handling dozens of node responses simultaneously. Plus, having the entire off-chain stack in one language (Go) makes it easier to manage dependencies and maintenance.

## Consensus Algorithm

1. **Collect:** Wait for node responses (up to 3 seconds).
2. **Sort:** Order all `internalMedianPrice` values.
3. **Calculate:** Find the Network Median.
4. **Filter:** Categorize into `HonestNodes` and `SlashedNodes`.
5. **Submit:** Execute `fulfillRequest(...)` on-chain.
