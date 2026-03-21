package main

import (
	"context"
	"log"
	"net/rpc"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Shared types with the Aggregator for net/rpc
type NodeReport struct {
	ReqId       [32]byte
	Price       uint64
	Signature   []byte
	NodeAddress string
}

type SubmitResponse struct {
	Success bool
	Message string
}

func main() {
	log.Println("Starting Decentralized Oracle Fetcher Node...")

	// 1. Configuration (In production, load via .env / os.Getenv)
	rpcURL := "ws://127.0.0.1:8545" // Target local Anvil/Hardhat node
	if envURL := os.Getenv("WS_RPC_URL"); envURL != "" {
		rpcURL = envURL
	}
	
	privKeyHex := os.Getenv("NODE_PRIVATE_KEY")
	if privKeyHex == "" {
		// Mock key for local testing
		privKeyHex = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80" 
	}
	
	privKey, _ := crypto.HexToECDSA(strings.TrimPrefix(privKeyHex, "0x"))
	nodeAddress := crypto.PubkeyToAddress(privKey.PublicKey).Hex()

	contractAddressHex := os.Getenv("ORACLE_CONTRACT_ADDRESS")
	if contractAddressHex == "" {
		contractAddressHex = "0x5FbDB2315678afecb367f032d93F642f64180aa3" // Default Hardhat local deploy
	}
	contractAddr := common.HexToAddress(contractAddressHex)

	aggregatorAddress := os.Getenv("AGGREGATOR_RPC_URL")
	if aggregatorAddress == "" {
		aggregatorAddress = "localhost:4000"
	}

	// 2. Setup Ethereum WebSocket Subscription
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to Ethereum Client via WSS: %v", err)
	}

	// keccak256("DataRequested(bytes32,string,uint256)")
	eventSignature := []byte("DataRequested(bytes32,string,uint256)")
	eventTopic := crypto.Keccak256Hash(eventSignature)

	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddr},
		Topics:    [][]common.Hash{{eventTopic}},
	}

	logs := make(chan types.Log)
	sub, err := client.SubscribeFilterLogs(context.Background(), query, logs)
	if err != nil {
		log.Fatalf("Failed to subscribe to logs: %v", err)
	}
	log.Printf("-- Fetcher [%s] listening for DataRequested on %s --", nodeAddress, contractAddressHex)

	for {
		select {
		case err := <-sub.Err():
			log.Fatalf("Subscription error: %v", err)
		case vLog := <-logs:
			// Parse the event (DataRequested)
			if len(vLog.Topics) < 2 {
				continue
			}
			reqId := vLog.Topics[1]

			// Note: The fully unABI-decoded symbol is stored in the data segment.
			// Since we just need the symbol, for string calldata it has dynamic offset.
			// For simplicity in this DON without abigen, we parse the raw data buffer.
			// The data layout is: Offset(32), Bounty(32), Length(32), StringData.
			dataBuf := vLog.Data
			if len(dataBuf) < 128 {
				continue
			}
			symbolLen := int(dataBuf[95]) // simple parse for short symbols
			symbol := string(dataBuf[96 : 96+symbolLen])

			log.Printf("Received Request! ReqId: %x, Symbol: %s", reqId.Bytes(), symbol)

			// 3. Fetch Prices Concurrently
			median := FetchMedian(symbol)
			log.Printf("Calculated internal median for %s: %d", symbol, median)

			if median == 0 {
				log.Printf("Failed to fetch median for %s", symbol)
				continue
			}

			// 4. Sign and Transmit
			var reqIdArray [32]byte
			copy(reqIdArray[:], reqId.Bytes())
			
			sig, err := SignPrice(privKeyHex, reqIdArray, median)
			if err != nil {
				log.Printf("Signing error: %v", err)
				continue
			}

			report := NodeReport{
				ReqId:       reqIdArray,
				Price:       median,
				Signature:   sig,
				NodeAddress: nodeAddress,
			}

			// Dial Aggregator via standard Go net/rpc
			client, err := rpc.DialHTTP("tcp", aggregatorAddress)
			if err != nil {
				log.Printf("Aggregator offline at %s: %v", aggregatorAddress, err)
				continue
			}

			var reply SubmitResponse
			err = client.Call("AggregatorService.SubmitReport", report, &reply)
			if err != nil {
				log.Printf("RPC Error: %v", err)
			} else {
				log.Printf("Submitted report. Aggregator response: %s", reply.Message)
			}
			client.Close()
		}
	}
}
