# Aggregator Node

The Aggregator acts as the consensus router for the Decentralized Oracle Network (DON).

## Architectural Tradeoffs & Decisions

1. **Protocol (RPC vs HTTP):**
   - **Decision:** The nodes connect to the Aggregator using `net/rpc` over standard TCP instead of REST endpoints.
   - **Tradeoff:** It removes the need to parse JSON overhead on every single node report, crucial for a high-throughput network handling thousands of data points a second.

2. **Consensus Timing (Channel Multiplexing vs Batching):**
   - **Decision:** As soon as the *first* node reports a price for a specific Request, a strict 3-second `time.Sleep` goroutine is spawned to define a consensus window.
   - **Tradeoff:** This ensures the Oracle responds strictly within the SLA defined in the PRD (3 seconds). If nodes are too slow to hit the 3-second hard deadline, they are simply excluded from that round's snapshot.

3. **Signature Validation (Aggregator Authority vs Fully decentralized Smart Contract Verifiers):**
   - **Decision:** The network uses a trusted Aggregator pattern where the on-chain contract simply verifies the final price was signed by the registered round-robin Aggregator, rather than the contract verifying all 5 individual node signatures (`ecrecover` loop).
   - **Tradeoff:** Fully decentralized validation of an array of ECDSA signatures on the EVM costs thousands of gas ($O(n)$ footprint), leading to unscalable networks. Validating a single Aggregator (which proves off-chain consensus was reached) costs only $O(1)$ gas. This aligns perfectly with off-chain computation models like CCIP and OCR.
