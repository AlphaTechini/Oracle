import Fastify, { FastifyRequest, FastifyReply } from "fastify";
import cors from "@fastify/cors";
import dotenv from "dotenv";
import { setupEventListeners } from "./events.js";
import { requestOracleData } from "./tx.js";
import { EventEmitter } from "events";

dotenv.config({ path: "../../.env" });

const fastify = Fastify({ logger: true });

// Event emitter to broadcast on-chain events to SSE clients
export const internalEventBus = new EventEmitter();

async function start() {
  await fastify.register(cors);

  fastify.get("/health", async (request: FastifyRequest, reply: FastifyReply) => {
    return { status: "ok", service: "dispatcher-api" };
  });

  // Client Request Endpoint: Initiates an On-Chain Oracle Request
  fastify.post("/request", async (request: any, reply: FastifyReply) => {
    const { symbol } = request.body as any;
    if (!symbol) return reply.code(400).send({ error: "Symbol required" });
    
    try {
      fastify.log.info(`Client requested data for ${symbol}`);
      // The dispatcher pays the bounty fee on behalf of the Web2 client (for demo purposes)
      const txHash = await requestOracleData(symbol);
      return { status: "pending", symbol, transactionHash: txHash };
    } catch (err) {
      fastify.log.error(err);
      return reply.code(500).send({ error: "Failed to request price on-chain" });
    }
  });

  // SSE Subscription Endpoint: Clients listen for fulfillment events
  fastify.get("/subscribe/:symbol", (request: any, reply: FastifyReply) => {
    const symbol = request.params.symbol;
    reply.raw.writeHead(200, {
      "Content-Type": "text/event-stream",
      "Cache-Control": "no-cache",
      "Connection": "keep-alive",
      "Access-Control-Allow-Origin": "*",
    });

    const onFulfilled = (data: { reqId: string; symbol: string; price: number }) => {
      if (data.symbol === symbol || symbol === "ALL") {
        reply.raw.write(`data: ${JSON.stringify(data)}\n\n`);
      }
    };

    internalEventBus.on("RequestFulfilled", onFulfilled);

    request.raw.on("close", () => {
      internalEventBus.removeListener("RequestFulfilled", onFulfilled);
    });
  });

  try {
    // Start listening to blockchain events from the DO Network
    await setupEventListeners();
    
    // Start HTTP server
    await fastify.listen({ port: 3001, host: "0.0.0.0" });
    fastify.log.info("Client Dispatcher API running on port 3001");
  } catch (err) {
    fastify.log.error(err);
    process.exit(1);
  }
}

start();
