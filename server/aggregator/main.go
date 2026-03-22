package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"log"
	"math"
	"math/big"
	"net"
	"net/rpc"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Shared types with the Fetcher for net/rpc
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

type Aggregator struct {
	mu            sync.Mutex
	reports       map[string][]NodeReport // hex(reqId) -> reports
	timers        map[string]bool         // keeps track if a timer is already running for reqId
	client        *ethclient.Client
	registryAddr  common.Address
	privateKey    *ecdsa.PrivateKey
	contractABI   abi.ABI
}

func NewAggregator(rpcURL, registryAddrHex, privKeyHex string) (*Aggregator, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}

	privKey, err := crypto.HexToECDSA(strings.TrimPrefix(privKeyHex, "0x"))
	if err != nil {
		return nil, err
	}

	// Minimal ABI for fulfillRequest(bytes32,uint256,address[],address[])
	abiJSON := `[{"inputs":[{"internalType":"bytes32","name":"reqId","type":"bytes32"},{"internalType":"uint256","name":"consensusPrice","type":"uint256"},{"internalType":"address[]","name":"honestNodes","type":"address[]"},{"internalType":"address[]","name":"slashedNodes","type":"address[]"}],"name":"fulfillRequest","outputs":[],"stateMutability":"nonpayable","type":"function"}]`
	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, err
	}

	return &Aggregator{
		reports:      make(map[string][]NodeReport),
		timers:       make(map[string]bool),
		client:       client,
		registryAddr: common.HexToAddress(registryAddrHex),
		privateKey:   privKey,
		contractABI:  parsedABI,
	}, nil
}

type AggregatorService struct {
	agg *Aggregator
}

// SubmitReport handles inbound net/rpc calls from ingestion nodes
func (s *AggregatorService) SubmitReport(report NodeReport, reply *SubmitResponse) error {
	reqIdHex := hex.EncodeToString(report.ReqId[:])

	s.agg.mu.Lock()
	s.agg.reports[reqIdHex] = append(s.agg.reports[reqIdHex], report)
	log.Printf("Received report for %x from %s (Price: %d)", report.ReqId, report.NodeAddress, report.Price)

	// If this is the FIRST report for this Request ID, start the 3-second Consensus Timer
	if !s.agg.timers[reqIdHex] {
		s.agg.timers[reqIdHex] = true
		go s.agg.startConsensusWindow(report.ReqId)
	}
	s.agg.mu.Unlock()

	reply.Success = true
	reply.Message = "Report Accepted into Consensus Pool"
	return nil
}

func (a *Aggregator) startConsensusWindow(reqId [32]byte) {
	reqIdHex := hex.EncodeToString(reqId[:])
	
	log.Printf("Consensus Window opened for %x (3 seconds)", reqId)
	time.Sleep(3 * time.Second) // Strict 3 second timeout SLA
	log.Printf("Consensus Window closed for %x. Calculating...", reqId)

	a.mu.Lock()
	reports := a.reports[reqIdHex]
	delete(a.reports, reqIdHex)
	delete(a.timers, reqIdHex)
	a.mu.Unlock()

	if len(reports) == 0 {
		return
	}

	// 1. Sort prices to find Median
	prices := make([]uint64, len(reports))
	for i, r := range reports {
		prices[i] = r.Price
	}
	sort.Slice(prices, func(i, j int) bool { return prices[i] < prices[j] })
	networkMedian := prices[len(prices)/2]

	// 2. Deviation Filtering > 0.2% (20 BPS)
	maxDev := networkMedian * 2 / 1000
	
	var honestNodes []common.Address
	var slashedNodes []common.Address

	for _, r := range reports {
		diff := uint64(math.Abs(float64(r.Price) - float64(networkMedian)))
		addr := common.HexToAddress(r.NodeAddress)
		if diff <= maxDev {
			honestNodes = append(honestNodes, addr)
		} else {
			slashedNodes = append(slashedNodes, addr)
		}
	}

	// 3. Submit Transaction to Blockchain
	a.submitToBlockchain(reqId, networkMedian, honestNodes, slashedNodes)
}

func (a *Aggregator) submitToBlockchain(reqId [32]byte, median uint64, honest []common.Address, slashed []common.Address) {
	log.Printf("Finalizing DON Consensus: Median=%d, Honest=%d, Slashed=%d", median, len(honest), len(slashed))

	// The DON Contract is optimized so the Aggregator's single ECDSA signature acts 
	// as the proof of threshold consensus. (msg.sender == getAggregator(reqId)).
	
	chainId, err := a.client.ChainID(context.Background())
	if err != nil {
		log.Printf("Failed to get chain ID: %v", err)
		return
	}

	auth, err := bind.NewKeyedTransactorWithChainID(a.privateKey, chainId)
	if err != nil {
		log.Printf("Failed to create transactor: %v", err)
		return
	}

	// Retrieve pending nonce
	nonce, err := a.client.PendingNonceAt(context.Background(), auth.From)
	if err != nil {
		log.Printf("Failed to get nonce: %v", err)
		return
	}
	auth.Nonce = big.NewInt(int64(nonce))
	auth.GasLimit = 3000000 // Safely estimate gas

	// Get latest gas prices
	gasPrice, err := a.client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Printf("Failed to suggest gas price: %v", err)
		return
	}
	auth.GasPrice = gasPrice

	// Bind Contract
	boundContract := bind.NewBoundContract(a.registryAddr, a.contractABI, a.client, a.client, a.client)

	// Call fulfillRequest(bytes32, uint256, address[], address[])
	// Convert [32]byte to bytes32, median to *big.Int
	medianInt := new(big.Int).SetUint64(median)
	
	tx, err := boundContract.Transact(auth, "fulfillRequest", reqId, medianInt, honest, slashed)
	if err != nil {
		log.Printf("Failed to send transaction: %v", err)
		return
	}

	log.Printf("Consensus Tx Sent! Hash: %s", tx.Hash().Hex())
}

func validateEnv() {
	required := []string{"HTTP_RPC_URL", "AGGREGATOR_PRIVATE_KEY", "ORACLE_CONTRACT_ADDRESS"}
	for _, env := range required {
		if os.Getenv(env) == "" {
			log.Printf("\x1b[33m[WARNING] Environment variable %s is not set. Using default value.\x1b[0m", env)
		}
	}
}

func main() {
	log.Println("DON Aggregator Node starting...")
	validateEnv()

	rpcURL := "http://127.0.0.1:8545"
	if envURL := os.Getenv("HTTP_RPC_URL"); envURL != "" {
		rpcURL = envURL
	}
	
	registryAddrHex := os.Getenv("ORACLE_CONTRACT_ADDRESS")
	if registryAddrHex == "" {
		registryAddrHex = "0x5FbDB2315678afecb367f032d93F642f64180aa3" 
	}

	privKeyHex := os.Getenv("AGGREGATOR_PRIVATE_KEY")
	if privKeyHex == "" {
		// Mock key for aggregator (Account #1 in hardhat)
		privKeyHex = "59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d" 
	}

	agg, err := NewAggregator(rpcURL, registryAddrHex, privKeyHex)
	if err != nil {
		log.Fatalf("Failed to init Aggregator: %v", err)
	}

	// Host net/rpc Server
	aggService := &AggregatorService{agg: agg}
	rpc.Register(aggService)

	port := os.Getenv("AGGREGATOR_PORT")
	if port == "" {
		port = "4000"
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("RPC Listener error: %v", err)
	}

	log.Printf("Aggregator net/rpc server listening on port %s", port)
	
	// Accept RPC connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept err: %v", err)
			continue
		}
		go rpc.ServeConn(conn)
	}
}
