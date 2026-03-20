import Redis from "ioredis";
import dotenv from "dotenv";

dotenv.config({ path: "../../.env" });

const redisUrl = process.env.REDIS_URL || "redis://localhost:6379";

export const redis = new Redis(redisUrl);

redis.on("error", (err) => {
  console.error("Redis connection error:", err);
});

redis.on("connect", () => {
  console.log("Connected to Redis State Layer");
});

/**
 * Fetch the latest price for a given symbol
 * Expects the ingestion engine to store prices as strings in Redis.
 */
export async function getLatestPrice(symbol: string): Promise<number | null> {
  const priceStr = await redis.get(`price:${symbol}`);
  if (!priceStr) return null;
  
  const price = parseFloat(priceStr);
  if (isNaN(price)) return null;

  // Smart contracts usually expect integers (e.g., price * 1e8 for 8 decimals)
  // We'll multiply by 1e8 here for precision.
  return Math.floor(price * 100000000);
}
