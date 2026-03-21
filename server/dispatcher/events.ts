import { oracleContract } from "./tx.js";
import { internalEventBus } from "./index.js";

export async function setupEventListeners() {
  console.log("Setting up Oracle RequestFulfilled event listener...");

  oracleContract.on("RequestFulfilled", async (reqId: string, consensusPrice: bigint, aggregator: string) => {
    console.log(`\n[EVENT] RequestFulfilled! Request ID: ${reqId}, Price: ${consensusPrice.toString()}`);

    try {
      // In a real app we'd map reqId back to the symbol. For demo SSE purposes,
      // we'll broadcast "ALL" symbol updates as well as the reqId.
      internalEventBus.emit("RequestFulfilled", {
        reqId: reqId,
        symbol: "ALL", // Simplified for SSE 
        price: Number(consensusPrice)
      });
      
    } catch (err) {
      console.error(`[ERROR] Failed to process fulfillment event:`, err);
    }
  });

  console.log("Listening for RequestFulfilled events on-chain...");
}
