import Fastify from "fastify";
import cors from "@fastify/cors";
import dotenv from "dotenv";
import { setupEventListeners } from "./events.js";

dotenv.config({ path: "../../.env" });

const fastify = Fastify({ logger: true });

async function start() {
  await fastify.register(cors);

  fastify.get("/health", async (request, reply) => {
    return { status: "ok", service: "dispatcher" };
  });

  fastify.get("/prices/:symbol", async (request: any, reply) => {
    const symbol = request.params.symbol;
    try {
      // Import on the fly or just use the redis client if imported globally
      // Actually we will just fetch using getLatestPrice (need to add import)
      const { getLatestPrice } = await import("./redis.js");
      const price = await getLatestPrice(symbol);
      return { symbol, price };
    } catch (err) {
      return reply.code(500).send({ error: "Failed to fetch price" });
    }
  });

  try {
    // Start listening to blockchain events
    await setupEventListeners();
    
    // Start HTTP server for healthchecks
    await fastify.listen({ port: 3001, host: "0.0.0.0" });
    fastify.log.info("Dispatcher running on port 3001");
  } catch (err) {
    fastify.log.error(err);
    process.exit(1);
  }
}

start();
