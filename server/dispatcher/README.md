# Node.js Dispatcher: Client API Gateway

The Dispatcher serves as the bridge between Web2 clients (like a frontend UI) and the Web3 Decentralized Oracle Network. 

In a true Chainlink-like architecture, clients need a way to easily trigger jobs or subscribe to data feeds without managing raw blockchain transactions themselves. This API Gateway provides that interface.

## Core Features

### 1. Requesting Data (`POST /request`)
Clients can hit this endpoint to request a price update for a specific symbol. 
- **Action:** The Dispatcher uses its own funded wallet to call `requestData()` on the `OracleRegistry` contract, paying the `bountyFee` on behalf of the client.
- **Result:** Emits a `DataRequested` event on-chain, waking up the Go Fetcher Nodes.

### 2. Subscribing to Fulfillments (`GET /subscribe/:symbol`)
Clients can establish a Server-Sent Events (SSE) connection to listen for price updates in real-time.
- **Action:** The Dispatcher listens to the `RequestFulfilled` event from the `OracleRegistry` contract.
- **Result:** Pushes the finalized, consensus-agreed price back to the Web2 client via the open connection.

## Setup & Running

Copy your `.env` variables, ensuring `DISPATCHER_PRIVATE_KEY` and `ORACLE_CONTRACT_ADDRESS` are set.

```powershell
pnpm install
pnpm dev
```
