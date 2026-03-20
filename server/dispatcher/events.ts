import { oracleContract } from "./tx.js";
import { signPricePayload } from "./signer.js";
import { getLatestPrice } from "./redis.js";
import { broadcastTx } from "./tx.js";

export async function setupEventListeners() {
  console.log("Setting up OracleRequest event listener...");

  oracleContract.on("OracleRequest", async (requestId: string, symbol: string) => {
    console.log(`\n[EVENT] OracleRequest received! Request ID: ${requestId}, Symbol: ${symbol}`);

    try {
      // 1. Fetch from Redis
      const price = await getLatestPrice(symbol);
      
      if (!price) {
        console.error(`[ERROR] No recent price found in Redis for ${symbol}`);
        return;
      }

      console.log(`[INFO] Latest price for ${symbol} retrieved: ${price}`);

      // 2. Sign Payload
      const signature = await signPricePayload(requestId, symbol, price);
      console.log(`[INFO] Payload signed successfully.`);

      // 3. Broadcast Transaction
      await broadcastTx(requestId, symbol, price, signature);
      
    } catch (err) {
      console.error(`[ERROR] Failed to process OracleRequest for ${symbol}:`, err);
    }
  });

  console.log("Listening for OracleRequest events...");
}
