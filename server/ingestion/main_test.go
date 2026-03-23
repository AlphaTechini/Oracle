package main

import (
	"testing"
)

// Tests the event parsing logic present in ingestion/main.go
func TestParseDataRequestedEvent(t *testing.T) {
	// The event signature in the smart contract is DataRequested(bytes32,string,string,uint256)
	// The data segment contains the strings and uint256.
	// We want to simulate the raw data array received in vLog.Data.

	// Constructing a payload matching the simple parsing logic in ParseDataRequestedEvent.
	// The function expects the symbol length at byte offset 95 and the symbol string at 96.
	// Then it expects the name length and string data following 32-byte chunks.

	symbolStr := "ETH"
	nameStr := "Ethereum"

	dataBuf := make([]byte, 200) // allocate enough space

	// Put symbol length at 95
	dataBuf[95] = byte(len(symbolStr))

	// Put symbol string at 96
	copy(dataBuf[96:], []byte(symbolStr))

	// Calculate nameOffset as main.go does
	symbolLen := int(dataBuf[95])
	nameOffset := 96 + 32*((symbolLen+31)/32) + 31

	// Put name length at nameOffset
	dataBuf[nameOffset] = byte(len(nameStr))

	// Put name string at nameOffset+1
	copy(dataBuf[nameOffset+1:], []byte(nameStr))


	// Test the extracted function ParseDataRequestedEvent
	symbol, name, err := ParseDataRequestedEvent(dataBuf)
	if err != nil {
		t.Fatalf("ParseDataRequestedEvent failed: %v", err)
	}

	if symbol != symbolStr {
		t.Errorf("Expected symbol %s, got %s", symbolStr, symbol)
	}

	if name != nameStr {
		t.Errorf("Expected name %s, got %s", nameStr, name)
	}
}
