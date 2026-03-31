package main

import (
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// We can't easily test the full Aggregator struct due to its dependency on the ethclient
// unless we mock a server. However, we can test the exact logic used in SubmitReport
// and startConsensusWindow by extracting or replicating it.

func TestSubmitReportLogic(t *testing.T) {
	hookCalled := make(chan bool, 1)

	agg := &Aggregator{
		reports: make(map[string][]NodeReport),
		timers:  make(map[string]bool),
		// client and other eth dependencies can be nil for this test
		// because we are only testing the in-memory tracking logic
		// before consensus window processing.
		startConsensusHook: func(reqId [32]byte) {
			hookCalled <- true
		},
	}

	service := &AggregatorService{agg: agg}

	var reqId [32]byte
	copy(reqId[:], []byte("req-1"))
	reqIdHex := hex.EncodeToString(reqId[:])

	report1 := NodeReport{ReqId: reqId, Price: 1000, NodeAddress: common.HexToAddress("0x1").Hex()}
	report2 := NodeReport{ReqId: reqId, Price: 1005, NodeAddress: common.HexToAddress("0x2").Hex()}

	var reply SubmitResponse

	err := service.SubmitReport(report1, &reply)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reply.Success {
		t.Errorf("Expected report to be accepted")
	}

	// Wait a tiny bit for the goroutine to write to the channel
	<-hookCalled

	agg.mu.Lock()
	startedTimer1 := agg.timers[reqIdHex]
	agg.mu.Unlock()

	if !startedTimer1 {
		t.Errorf("Expected timer state to be true on first report")
	}

	err = service.SubmitReport(report2, &reply)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	agg.mu.Lock()
	reportsCount := len(agg.reports[reqIdHex])
	agg.mu.Unlock()

	if reportsCount != 2 {
		t.Errorf("Expected 2 reports, got %d", reportsCount)
	}
}

func TestCalculateConsensus(t *testing.T) {
	// Tests the extracted CalculateConsensus logic
	reports := []NodeReport{
		{Price: 100000, NodeAddress: common.HexToAddress("0x1").Hex()},
		{Price: 100100, NodeAddress: common.HexToAddress("0x2").Hex()},
		{Price: 100200, NodeAddress: common.HexToAddress("0x3").Hex()}, // Median will be 100100
		{Price: 105000, NodeAddress: common.HexToAddress("0x4").Hex()}, // Outlier (slashed) > 0.2% deviation
		{Price: 99990, NodeAddress: common.HexToAddress("0x5").Hex()},
	}

	networkMedian, honestNodes, slashedNodes := CalculateConsensus(reports)

	if networkMedian != 100100 {
		t.Errorf("Expected median 100100, got %d", networkMedian)
	}

	// 100100 * 2 / 1000 = 200.2 -> 200 max deviation.
	// Honest: 0x1 (100000, diff 100), 0x2 (100100, diff 0), 0x3 (100200, diff 100), 0x5 (99990, diff 110)
	// Slashed: 0x4 (105000, diff 4900)

	if len(honestNodes) != 4 {
		t.Errorf("Expected 4 honest nodes, got %d", len(honestNodes))
	}

	if len(slashedNodes) != 1 {
		t.Fatalf("Expected 1 slashed node, got %d", len(slashedNodes))
	}

	slashedAddr := common.HexToAddress("0x4")
	if slashedNodes[0] != slashedAddr {
		t.Errorf("Expected slashed node to be %s, got %s", slashedAddr.Hex(), slashedNodes[0].Hex())
	}
}

func TestCalculateConsensusEmpty(t *testing.T) {
	var reports []NodeReport
	median, honest, slashed := CalculateConsensus(reports)
	if median != 0 || honest != nil || slashed != nil {
		t.Errorf("Expected empty returns for empty reports, got %d %v %v", median, honest, slashed)
	}
}
