package main

import (
	"log"
	"time"
)

func main() {
	log.Println("Starting High-Throughput Web3 Oracle Ingestion Engine...")

	InitRedis()

	symbols := []string{"tBTCUSD", "tETHUSD", "tSOLUSD"}
	
	// Create a goroutine to manage the websocket connection and processing
	go ConnectToBitfinex(symbols)

	// In a real application, we'd want to block forever and gracefully shut down
	// on SIGINT. Here, we just sleep forever.
	for {
		time.Sleep(1 * time.Hour)
	}
}
