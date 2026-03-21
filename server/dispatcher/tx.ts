import { ethers } from "ethers";
import dotenv from "dotenv";

dotenv.config({ path: "../../.env" });

const rpcUrl = process.env.RPC_URL || "http://127.0.0.1:8545";
export const provider = new ethers.JsonRpcProvider(rpcUrl);

// Use a private key for the dispatcher to sponsor client requests
const privateKey = process.env.DISPATCHER_PRIVATE_KEY || "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80";
export const signer = new ethers.Wallet(privateKey, provider);

// OracleRegistry ABI
export const OracleABI = [
  "event DataRequested(bytes32 indexed reqId, string symbol, uint256 bounty)",
  "event RequestFulfilled(bytes32 indexed reqId, uint256 consensusPrice, address aggregator)",
  "function requestData(string calldata symbol) external payable returns (bytes32)"
];

const contractAddress = process.env.ORACLE_CONTRACT_ADDRESS;

if (!contractAddress) {
  console.warn("ORACLE_CONTRACT_ADDRESS is not set. Broadcasting will fail.");
}

export const oracleContract = new ethers.Contract(
  contractAddress || ethers.ZeroAddress,
  OracleABI,
  signer
);

/**
 * Initiates an on-chain data request using the dispatcher's wallet.
 * The client asks the dispatcher, the dispatcher pays the bounty fee.
 */
export async function requestOracleData(symbol: string): Promise<string> {
  try {
    console.log(`Submitting on-chain request for ${symbol}...`);
    
    // We send the transaction with the required bounty fee
    const bountyFee = ethers.parseEther("0.01"); // Example bounty fee
    const tx = await oracleContract.requestData(symbol, {
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
