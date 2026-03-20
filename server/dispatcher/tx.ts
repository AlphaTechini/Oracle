import { ethers } from "ethers";
import { wallet } from "./signer.js";
import dotenv from "dotenv";

dotenv.config({ path: "../../.env" });

const rpcUrl = process.env.RPC_URL || "http://127.0.0.1:8545";
export const provider = new ethers.JsonRpcProvider(rpcUrl);

// Connect wallet to provider
export const signer = wallet.connect(provider);

// Provide the compiled ABI for the Oracle contract
const OracleABI = [
  "event OracleRequest(bytes32 indexed requestId, string symbol)",
  "function fulfill(bytes32 requestId, string calldata symbol, uint256 price, bytes calldata signature) external"
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
 * Broadcasts the transaction to the smart contract.
 * Includes basic nonce management and gas logic.
 */
export async function broadcastTx(
  requestId: string,
  symbol: string,
  price: number,
  signature: string
) {
  try {
    console.log(`Broadcasting price of ${price} for ${symbol}...`);
    
    // We send the transaction
    const tx = await oracleContract.fulfill(requestId, symbol, price, signature, {
      gasLimit: 300000 // basic gas limit for fulfillment
    });

    console.log(`Transaction submitted! Hash: ${tx.hash}`);
    
    // Optionally wait for confirmation
    const receipt = await tx.wait(1);
    console.log(`Transaction confirmed in block: ${receipt.blockNumber}`);
  } catch (err) {
    console.error(`Error broadcastinging TX for ${symbol}:`, err);
  }
}
