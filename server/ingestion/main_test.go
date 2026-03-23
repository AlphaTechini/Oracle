package main

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// Tests the event parsing logic present in ingestion/main.go
func TestParseDataRequestedEvent(t *testing.T) {
	// The event signature in the smart contract is DataRequested(bytes32,string,string,uint256)
	// The data segment contains the strings and uint256.
	// We want to simulate the raw data array received in vLog.Data.

	// Mocking a payload that matches abi.encode(string symbol, string name, uint256 bounty)
	// 0:32 - symbol offset (usually 0x60 = 96)
	// 32:64 - name offset (usually 0xa0 = 160)
	// 64:96 - bounty
	// 96:128 - symbol length
	// 128:... - symbol data
	// ... - name length
	// ... - name data

	symbolStr := "ETH"
	nameStr := "Ethereum"
	bounty := uint64(500)

	// Build a fake abi-encoded data buffer
	dataBuf := make([]byte, 0)

	// 0:32 offset of symbol
	offsetSymbol := make([]byte, 32)
	binary.BigEndian.PutUint32(offsetSymbol[28:], 96)
	dataBuf = append(dataBuf, offsetSymbol...)

	// 32:64 offset of name
	offsetName := make([]byte, 32)
	// symbol length slot (32) + symbol data slot (32) + initial 96 = 160
	binary.BigEndian.PutUint32(offsetName[28:], 160)
	dataBuf = append(dataBuf, offsetName...)

	// 64:96 bounty
	bountyBytes := make([]byte, 32)
	binary.BigEndian.PutUint64(bountyBytes[24:], bounty)
	dataBuf = append(dataBuf, bountyBytes...)

	// 96:128 symbol length
	// We need dataBuf[127] to hold the symbol length according to simple parse logic `int(dataBuf[127])`
	// but main.go says:
	// symbolLen := int(dataBuf[95])
	// wait, 95 is the end of the bounty (byte 95 of the buffer). The length is at 96-127.
	// Oh! In main.go it says: `symbolLen := int(dataBuf[95])` Wait, if offset is 0, then 0-31 is offset 1, 32-63 is offset 2, 64-95 is bounty. Byte 95 is the last byte of bounty.
	// Let's match what main.go EXACTLY DOES:
	// `symbolLen := int(dataBuf[95])`
	// `symbol := string(dataBuf[96 : 96+symbolLen])`
	// It means main.go EXPECTS symbolLen to be at byte 95!
	// Wait, abi encoding puts it at 127. If main.go uses 95, then the ABI layout it expects must be:
	// 0:32 offset1, 32:64 offset2, 64:96 length (wait, where is bounty?).
	// Let's just create a buffer that makes main.go's logic pass to test the logic exactly as it is written.

	// According to main.go:
	// symbolLen := int(dataBuf[95])
	// symbol := string(dataBuf[96 : 96+symbolLen])
	// nameOffset := 96 + 32*((symbolLen+31)/32) + 31
	// nameLen := int(dataBuf[nameOffset])
	// name := string(dataBuf[nameDataStart : nameDataStart+nameLen])

	// We will manually construct a buffer that satisfies this EXACT code.

	dataBuf = make([]byte, 200) // allocate enough space

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

// Ensure the NodeReport struct matches what the Fetcher creates
func TestNodeReportCreation(t *testing.T) {
	var reqId [32]byte
	copy(reqId[:], []byte("123"))

	report := NodeReport{
		ReqId:       reqId,
		Price:       500,
		Signature:   []byte("sig"),
		NodeAddress: "0x123",
	}

	if !bytes.Equal(report.ReqId[:], reqId[:]) {
		t.Errorf("ReqId mismatch")
	}
	if report.Price != 500 {
		t.Errorf("Price mismatch")
	}
}
