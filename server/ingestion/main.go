package main

import (
	"log"
	"time"
)

func main() {
	log.Println("Starting Decentralized Oracle Fetcher Node...")

	// Example: In a loop, fetch median every 10 seconds for a request
	// In a real DON, this would be triggered by a contract event
	symbol := "BTC"
	
	for {
		log.Printf("--- New Rounds for %s ---", symbol)
		
		// 1. Fetch Internal Median from 5 sources
		median := FetchMedian(symbol)
		log.Printf("Internal Median for %s: %d (scaled)", symbol, median)

		// 2. Sign the data (Simulating a requestId)
		var dummyReqId [32]byte
		copy(dummyReqId[:], "req_123")
		
		// Note: USE_ENV_KEY would be loaded from .env
		privKey := "0xabc123..." // Placeholder for node's key
		
		sig, err := SignPrice(privKey, dummyReqId, median)
		if err != nil {
			log.Printf("Signing Error: %v", err)
		} else {
			log.Printf("Signed Median successfully. Sig length: %d", len(sig))
		}

		time.Sleep(10 * time.Second)
	}
}
