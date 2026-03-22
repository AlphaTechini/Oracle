import { ethers } from "ethers";
import dotenv from "dotenv";

dotenv.config({ path: "../../.env" });

const rpcUrl = process.env.RPC_URL || "http://127.0.0.1:8545";
export const provider = new ethers.JsonRpcProvider(rpcUrl);

// Dispatcher relayer wallet (pays the gas/bounty for client requests)
const privateKey = process.env.DISPATCHER_PRIVATE_KEY || "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80";
export const signer = new ethers.Wallet(privateKey, provider);

// OracleRegistry ABI
export const OracleABI = [
  "event DataRequested(bytes32 indexed reqId, string symbol, string name, uint256 bounty)",
  "event RequestFulfilled(bytes32 indexed reqId, uint256 consensusPrice, address aggregator)",
  "function requestData(string calldata symbol, string calldata name) external payable returns (bytes32)",
  "function requestDataBatch(string[] calldata symbols, string[] calldata names) external payable"
];

const contractAddress = process.env.ORACLE_CONTRACT_ADDRESS || ethers.ZeroAddress;
export const oracleContract = new ethers.Contract(contractAddress, OracleABI, signer);

// EIP-712 Domain and Types for Client Intent Verification
const domain = {
  name: "Oracle DON",
  version: "1",
  chainId: 31337, // Default Hardhat, should be dynamic in prod
  verifyingContract: contractAddress
};

interface TokenRequest {
  symbol: string;
  name: string;
}

const types = {
  DataRequest: [
    { name: "symbol", type: "string" },
    { name: "name", type: "string" },
    { name: "timestamp", type: "uint256" }
  ],
  BatchDataRequest: [
    { name: "tokens", type: "TokenRequest[]" },
    { name: "timestamp", type: "uint256" }
  ],
  TokenRequest: [
    { name: "symbol", type: "string" },
    { name: "name", type: "string" }
  ]
};

/**
 * Validates an EIP-712 signature for a batch request.
 */
export function verifyBatchClientIntent(tokens: TokenRequest[], timestamp: number, signature: string, expectedSigner: string): boolean {
  try {
    const value = { tokens, timestamp };
    const recoveredAddress = ethers.verifyTypedData(domain, types, value, signature);
    return recoveredAddress.toLowerCase() === expectedSigner.toLowerCase();
  } catch (err) {
    console.error("Batch EIP-712 Verification failed:", err);
    return false;
  }
}

/**
 * Validates an EIP-712 signature from a client.
 */
export function verifyClientIntent(symbol: string, name: string, timestamp: number, signature: string, expectedSigner: string): boolean {
  try {
    const value = { symbol, name, timestamp };
    const recoveredAddress = ethers.verifyTypedData(domain, types, value, signature);
    return recoveredAddress.toLowerCase() === expectedSigner.toLowerCase();
  } catch (err) {
    console.error("EIP-712 Verification failed:", err);
    return false;
  }
}

/**
 * Initiates an on-chain data request using the dispatcher's wallet.
 */
export async function requestOracleData(symbol: string, name: string): Promise<string> {
  try {
    console.log(`Submitting on-chain request for ${symbol} (${name})...`);
    
    // Dispatcher pays the bounty fee on behalf of the client
    const bountyFee = ethers.parseEther("0.01"); 
    const tx = await oracleContract.requestData(symbol, name, {
      value: bountyFee,
      gasLimit: 300000
    });

    console.log(`Request submitted! Hash: ${tx.hash}`);
    return tx.hash;
  } catch (err) {
    console.error(`Error requesting data for ${symbol}:`, err);
    throw err;
  }
}

/**
 * Initiates an on-chain batch data request.
 */
export async function requestOracleDataBatch(tokens: TokenRequest[]): Promise<string> {
  try {
    const symbols = tokens.map(t => t.symbol);
    const names = tokens.map(t => t.name);
    
    console.log(`Submitting on-chain batch request for ${symbols.join(", ")}...`);
    
    // Dispatcher pays the bounty fee per request
    const totalBounty = ethers.parseEther((0.01 * tokens.length).toString()); 
    const tx = await oracleContract.requestDataBatch(symbols, names, {
      value: totalBounty,
      gasLimit: 500000 + (100000 * tokens.length) // Dynamic gas based on batch size
    });

    console.log(`Batch request submitted! Hash: ${tx.hash}`);
    return tx.hash;
  } catch (err) {
    console.error(`Error requesting batch data:`, err);
    throw err;
  }
}
