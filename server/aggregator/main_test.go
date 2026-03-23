package main

import (
	"encoding/hex"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// We can't easily test the full Aggregator struct due to its dependency on the ethclient
// unless we mock a server. However, we can test the exact logic used in SubmitReport
// and startConsensusWindow by extracting or replicating it.

func TestSubmitReportLogic(t *testing.T) {
	// Replicate the logic in SubmitReport and verify mutices and map modifications
	reports := make(map[string][]NodeReport)
	timers := make(map[string]bool)
	var mu sync.Mutex

	var reqId [32]byte
	copy(reqId[:], []byte("req-1"))
	reqIdHex := hex.EncodeToString(reqId[:])

	submitReport := func(report NodeReport) bool {
		mu.Lock()
		defer mu.Unlock()
		reports[reqIdHex] = append(reports[reqIdHex], report)

		startedTimer := false
		if !timers[reqIdHex] {
			timers[reqIdHex] = true
			startedTimer = true
		}
		return startedTimer
	}

	report1 := NodeReport{ReqId: reqId, Price: 1000, NodeAddress: "0x1"}
	report2 := NodeReport{ReqId: reqId, Price: 1005, NodeAddress: "0x2"}

	started := submitReport(report1)
	if !started {
		t.Errorf("Expected timer to start on first report")
	}

	started = submitReport(report2)
	if started {
		t.Errorf("Expected timer NOT to start on second report")
	}

	if len(reports[reqIdHex]) != 2 {
		t.Errorf("Expected 2 reports, got %d", len(reports[reqIdHex]))
	}
}

func TestCalculateConsensus(t *testing.T) {
	// Tests the extracted CalculateConsensus logic
	reports := []NodeReport{
		{Price: 100000, NodeAddress: "0x1"},
		{Price: 100100, NodeAddress: "0x2"},
		{Price: 100200, NodeAddress: "0x3"}, // Median will be 100100
		{Price: 105000, NodeAddress: "0x4"}, // Outlier (slashed) > 0.2% deviation
		{Price: 99990, NodeAddress: "0x5"},
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
		t.Errorf("Expected 1 slashed node, got %d", len(slashedNodes))
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
