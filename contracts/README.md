# Solidity Contracts: Oracle Registry & Settlement

This folder contains the core logic for the Decentralized Oracle Network. It manages node registration, staking, data requests, and fulfillment.

## Key Components

- **`OracleRegistry.sol`**: The main entry point. It handles:
  - `registerNode()`: Nodes must stake a minimum amount of ETH.
  - `requestData(symbol)`: Clients pay a bounty fee to trigger a request.
  - `fulfillRequest(...)`: Verifies signatures and distributes rewards/slashing.

## Architectural Decisions & Tradeoffs

### 1. Verification Logic: Looping `ecrecover`
I chose to verify signatures in a simple loop. 
- **Decision:** Loop through `honestNodes` and call `ecrecover`.
- **Tradeoff:** Higher gas costs as the number of nodes increases. However, the simplicity makes the contract much safer and easier to audit.

### 2. Staking & Slashing
Nodes are required to stake ETH to participate.
- **Why:** This ensures everyone has skin in the game. If a node reports data outside the 0.2% deviation range, they get slashed (10% of their stake). This creates a strong economic incentive for honesty.

### 3. Rewards Distribution
The `bountyFee` from the client is divided equally among all "honest" nodes (those within the deviation threshold).
- **Decision:** Equal split.
- **Tradeoff:** It doesn't account for individual trust scores yet, but it's fair and keeps the contract math simple.

## Development & Testing

### Scripts
- `deploy.ts`: Deploys the Registry and initializes settings.
- `verify.ts`: Verifies signatures against the contract on-chain.

### Testing
Wait until I've added the multi-node logic to run these:
```powershell
pnpm hardhat test
```
