import { ethers } from "ethers";
import dotenv from "dotenv";

dotenv.config({ path: "../../.env" });

const privateKey = process.env.ORACLE_PRIVATE_KEY;
if (!privateKey) {
  throw new Error("ORACLE_PRIVATE_KEY is not defined in the environment variables");
}

export const wallet = new ethers.Wallet(privateKey);

/**
 * Generates an ECDSA signature for the given request payload.
 */
export async function signPricePayload(
  requestId: string,
  symbol: string,
  price: number
): Promise<string> {
  // Message Hash = keccak256(abi.encodePacked(requestId, symbol, price))
  const messageHash = ethers.solidityPackedKeccak256(
    ["bytes32", "string", "uint256"],
    [requestId, symbol, price]
  );

  const messageBytes = ethers.getBytes(messageHash);
  const signature = await wallet.signMessage(messageBytes);

  return signature;
}
