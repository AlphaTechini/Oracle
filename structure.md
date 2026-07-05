# Project Structure Guide

This guide provides a map of the Decentralized Oracle Network (DON) codebase, highlighting the major files that house the primary functions and logic.

## 🏗️ On-Chain Layer (Solidity)
The on-chain layer handles registration, request tracking, and settlement.
- **[OracleRegistry.sol](contracts/contracts/OracleRegistry.sol)**: The core smart contract. It manages node registration (staking), data request events, and the final fulfillment/settlement logic where rewards are distributed and malicious nodes are slashed.

## 📡 Off-Chain Data Ingestion (Go)
Fetcher nodes are responsible for retrieving real-time data from external sources.
- **[fetcher.go](server/ingestion/fetcher.go)**: Contains the logic for fetching prices from multiple sources (Binance, Coinbase, etc.), calculating the local median, and signing the report.

## 🤝 Aggregation & Consensus (Go)
The aggregator layer bridges the fetchers and the blockchain.
- **[aggregator/main.go](server/aggregator/main.go)**: Houses the consensus logic. It collects reports from multiple fetcher nodes, calculates the network-wide median, determines honest vs. slashed nodes based on deviation, and submits the final transaction to the `OracleRegistry` contract.

## 🚀 API Gateway (Node.js)
The dispatcher acts as the entry point for clients.
- **[dispatcher/index.ts](server/dispatcher/index.ts)**: The primary API server. It handles client requests (EIP-712 signed), triggers on-chain data requests, and provides a Server-Sent Events (SSE) stream for clients to subscribe to real-time fulfillment updates.

## 📊 Monitoring & UI (SvelteKit)
The dashboard provides a visual overview of the network.
- **[dashboard/src/routes/+page.svelte](dashboard/src/routes/+page.svelte)**: The main frontend page for tracking active price feeds and node health.

---

### Key Architectural Decisions
To understand *why* the project is structured this way, refer to the [Architectural Decisions & Tradeoffs](README.md#architectural-decisions--tradeoffs) section in the main README.
