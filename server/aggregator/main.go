package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// NodeReport represents the data sent by a Fetcher Node
type NodeReport struct {
	NodeAddress common.Address
	Price       uint64
	Signature   []byte
}

type Aggregator struct {
	mu           sync.Mutex
	reports      map[string][]NodeReport // reqId -> reports
	client       *ethclient.Client
	registryAddr common.Address
	privateKey   string
}

func NewAggregator(rpcURL, registryAddr, privKey string) (*Aggregator, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil { return nil, err }
	return &Aggregator{
		reports:      make(map[string][]NodeReport),
		client:       client,
		registryAddr: common.HexToAddress(registryAddr),
		privateKey:   privKey,
	}, nil
}

// AddReport adds a report from a node to the aggregator's memory
func (a *Aggregator) AddReport(reqId string, report NodeReport) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.reports[reqId] = append(a.reports[reqId], report)
}

// ProcessConsensus handles the Network Median calculation and filtering
func (a *Aggregator) ProcessConsensus(reqId [32]byte) (uint64, []common.Address, []common.Address, [][]byte) {
	reqIdStr := fmt.Sprintf("%x", reqId)
	a.mu.Lock()
	reports := a.reports[reqIdStr]
	delete(a.reports, reqIdStr) // clear memory
	a.mu.Unlock()

	if len(reports) == 0 { return 0, nil, nil, nil }

	// 1. Sort prices
	prices := make([]uint64, len(reports))
	for i, r := range reports { prices[i] = r.Price }
	sort.Slice(prices, func(i, j int) bool { return prices[i] < prices[j] })

	// 2. Calculate Network Median
	networkMedian := prices[len(prices)/2]

	// 3. Deviation Filtering (0.2%)
	// Max Deviation = NetworkMedian * 2 / 1000
	maxDev := networkMedian * 2 / 1000
	
	var honestNodes []common.Address
	var slashedNodes []common.Address
	var signatures [][]byte

	for _, r := range reports {
		diff := uint64(math.Abs(float64(r.Price) - float64(networkMedian)))
		if diff <= maxDev {
			honestNodes = append(honestNodes, r.NodeAddress)
			signatures = append(signatures, r.Signature)
		} else {
			slashedNodes = append(slashedNodes, r.NodeAddress)
		}
	}

	return networkMedian, honestNodes, slashedNodes, signatures
}

// SubmitConsensus sends the finalize transaction to the contract
func (a *Aggregator) SubmitConsensus(reqId [32]byte, median uint64, honest []common.Address, slashed []common.Address, sigs [][]byte) error {
	// In a real app, I'd use the abigen-generated bindings. 
	// For this task, I'll describe the steps to use TransactOpts.
	
	log.Printf("Submitting Consensus for %x: Median=%d, Honest=%d, Slashed=%d", reqId, median, len(honest), len(slashed))
	
	// Implementation note: 
	// 1. Parse Private Key
	// 2. Create TransactOpts
	// 3. Call registry.FulfillRequest(reqId, median, honest, slashed, sigs)
	
	return nil
}

func main() {
	log.Println("Go Aggregator Node starting...")
	// Skeleton main loop
	for {
		time.Sleep(1 * time.Hour)
	}
}
