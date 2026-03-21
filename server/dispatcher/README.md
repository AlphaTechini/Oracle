# Dispatcher Node.js API

The Dispatcher API acts as the client-facing gateway (Web2 to Web3) for to the Decentralized Oracle Network.

## Architectural Tradeoffs & Decisions

1. **Framework (Fastify vs Express):**
   - **Decision:** As per project guidelines, this API strictly uses Fastify.
   - **Tradeoff:** Fastify provides significantly higher throughput (requests per second) than Express natively, which is crucial for an Oracle gateway that might be spammed by Web2 clients.

2. **Client Authentication (EIP-712 Intent vs On-Chain Allowance):**
   - **Decision:** The API requires clients to submit an **EIP-712** typed signature with their request. The API verifies this signature *off-chain* before paying the on-chain bounty on behalf of the client (acting as a Relayer). 
   - **Tradeoff:** If we required on-chain `ERC20.approve` and `transferFrom`, clients would have to pay their own gas just to request data. By using the Relayer pattern protected by EIP-712, the API provides a seamless Web2 "gasless" experience for authorized clients, while mathematically blocking unauthorized bot spam from draining the relayer's gas pool.

3. **Client Subscriptions (Server-Sent Events vs WebSockets vs Polling):**
   - **Decision:** Chosen **Server-Sent Events (SSE)** via `fastify` raw responses over full bidirectional WebSockets.
   - **Tradeoff:** Since the client only needs to *listen* for the finalized price (unidirectional data flow: Server -> Client), SSE is much lighter than WebSockets, works natively over HTTP/1.1 and HTTP/2, and natively handles auto-reconnections in browsers (`EventSource`). Polling was discarded as it introduces artificial latency.
