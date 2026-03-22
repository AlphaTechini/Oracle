import Fastify, { FastifyRequest, FastifyReply } from "fastify";
import cors from "@fastify/cors";
import dotenv from "dotenv";
import { setupEventListeners } from "./events.js";
import { requestOracleData, requestOracleDataBatch, verifyClientIntent, verifyBatchClientIntent } from "./tx.js";
import { EventEmitter } from "events";

dotenv.config({ path: "../../.env" });

const fastify = Fastify({ logger: true });
export const internalEventBus = new EventEmitter();

interface RequestBody {
  symbol: string;
  name: string;
  timestamp: number;
  signature: string;
  clientAddress: string;
}

interface BatchRequestBody {
  tokens: { symbol: string; name: string }[];
  timestamp: number;
  signature: string;
  clientAddress: string;
}

function validateEnv() {
  const required = ["RPC_URL", "DISPATCHER_PRIVATE_KEY", "ORACLE_CONTRACT_ADDRESS"];
  const missing = required.filter(k => !process.env[k]);
  if (missing.length > 0) {
    console.warn(`\x1b[33m[WARNING] Missing environment variables: ${missing.join(", ")}\x1b[0m`);
    console.warn("\x1b[33m[WARNING] Server may not function correctly with default values.\x1b[0m");
  }
}

async function start() {
  validateEnv();
  await fastify.register(cors);

  fastify.get("/health", async () => {
    return { status: "ok", service: "dispatcher-api" };
  });

  // Client Batch Request Endpoint
  fastify.post("/batch-request", async (request: FastifyRequest, reply: FastifyReply) => {
    const { tokens, timestamp, signature, clientAddress } = request.body as BatchRequestBody;
    
    if (!tokens || !Array.isArray(tokens) || tokens.length === 0 || !timestamp || !signature || !clientAddress) {
      return reply.code(400).send({ error: "Missing required fields (tokens[], timestamp, signature, clientAddress)" });
    }

    if (tokens.length > 30) {
      return reply.code(400).send({ error: "Batch size exceeds maximum limit of 30 tokens" });
    }

    // Verify EIP-712 Batch signature
    const isValid = verifyBatchClientIntent(tokens, timestamp, signature, clientAddress);
    if (!isValid) {
      fastify.log.warn(`Invalid Batch EIP-712 signature from ${clientAddress}`);
      return reply.code(401).send({ error: "Invalid Batch EIP-712 intent signature" });
    }
    
    try {
      fastify.log.info(`Verified batch intent. Sponsoring fetch for ${tokens.length} tokens`);
      const txHash = await requestOracleDataBatch(tokens);
      return { status: "pending", tokens: tokens.map(t => t.symbol), transactionHash: txHash };
    } catch (err) {
      fastify.log.error(err);
      return reply.code(500).send({ error: "Failed to request batch price on-chain" });
    }
  });

  // Client Request Endpoint (Protects bounty pool with EIP-712)
  fastify.post("/request", async (request: FastifyRequest, reply: FastifyReply) => {
    const { symbol, name, timestamp, signature, clientAddress } = request.body as RequestBody;
    
    if (!symbol || !name || !timestamp || !signature || !clientAddress) {
      return reply.code(400).send({ error: "Missing required EIP-712 fields (symbol, name, timestamp, signature, clientAddress)" });
    }

    // Protection: Verify the client intent natively before spending Oracle gas
    const isValid = verifyClientIntent(symbol, name, timestamp, signature, clientAddress);
    if (!isValid) {
      fastify.log.warn(`Invalid EIP-712 signature from ${clientAddress}`);
      return reply.code(401).send({ error: "Invalid EIP-712 intent signature" });
    }
    
    try {
      fastify.log.info(`Verified intent. Sponsoring fetch for ${symbol} (${name})`);
      const txHash = await requestOracleData(symbol, name);
      return { status: "pending", symbol, name, transactionHash: txHash };
    } catch (err) {
      fastify.log.error(err);
      return reply.code(500).send({ error: "Failed to request price on-chain" });
    }
  });

  // Server-Sent Events (SSE) Endpoint - Native Fastify Flow
  fastify.get("/subscribe/:symbol", async (request: FastifyRequest, reply: FastifyReply) => {
    const { symbol } = request.params as { symbol: string };
    
    reply.raw.setHeader('Content-Type', 'text/event-stream');
    reply.raw.setHeader('Cache-Control', 'no-cache');
    reply.raw.setHeader('Connection', 'keep-alive');
    reply.raw.setHeader('Access-Control-Allow-Origin', '*');

    // Send initial connection header
    reply.raw.write("retry: 10000\n\n");

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
    await setupEventListeners();
    await fastify.listen({ port: 3001, host: "0.0.0.0" });
    fastify.log.info("Dispatcher API running on port 3001");
  } catch (err) {
    fastify.log.error(err);
    process.exit(1);
  }
}

start();
