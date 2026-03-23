package main

import (
	"strings"
	"testing"
)

// Tests the event parsing logic present in ingestion/main.go
func TestParseDataRequestedEvent(t *testing.T) {
	tests := []struct {
		name        string
		setupBuf    func() []byte
		wantSymbol  string
		wantName    string
		wantErrText string
	}{
		{
			name: "Happy Path",
			setupBuf: func() []byte {
				symbolStr := "ETH"
				nameStr := "Ethereum"
				dataBuf := make([]byte, 200)
				dataBuf[95] = byte(len(symbolStr))
				copy(dataBuf[96:], []byte(symbolStr))
				nameOffset := 96 + 32*((len(symbolStr)+31)/32) + 31
				dataBuf[nameOffset] = byte(len(nameStr))
				copy(dataBuf[nameOffset+1:], []byte(nameStr))
				return dataBuf
			},
			wantSymbol: "ETH",
			wantName:   "Ethereum",
		},
		{
			name: "Buffer Too Short",
			setupBuf: func() []byte {
				return make([]byte, 150) // Less than 160
			},
			wantErrText: "dataBuf too short",
		},
		{
			name: "nameData out of bounds",
			setupBuf: func() []byte {
				symbolStr := "ETH"
				dataBuf := make([]byte, 160)
				dataBuf[95] = byte(len(symbolStr))
				nameOffset := 96 + 32*((len(symbolStr)+31)/32) + 31 // 159
				dataBuf[nameOffset] = 10 // Name length is 10, but buffer is only 160 bytes. nameDataStart + nameLen = 160 + 10 = 170
				return dataBuf
			},
			wantErrText: "nameData out of bounds",
		},
		{
			name: "Symbol Length Clamping (>32)",
			setupBuf: func() []byte {
				// We claim the symbol is 50 bytes long, but it should be clamped to 32.
				symbolStr := strings.Repeat("A", 50)
				dataBuf := make([]byte, 200)
				dataBuf[95] = 50 // length > 32
				copy(dataBuf[96:], []byte(symbolStr)) // Copies 50 bytes

				// Calculate nameOffset as main.go does, with clamping to 32
				clampedLen := 32
				nameOffset := 96 + 32*((clampedLen+31)/32) + 31
				dataBuf[nameOffset] = 4
				copy(dataBuf[nameOffset+1:], []byte("Test"))
				return dataBuf
			},
			wantSymbol: strings.Repeat("A", 32), // Should only read 32 bytes
			wantName:   "Test",
		},
		{
			name: "Empty Strings",
			setupBuf: func() []byte {
				dataBuf := make([]byte, 160)
				dataBuf[95] = 0 // Empty symbol

				nameOffset := 96 + 32*((0+31)/32) + 31 // 96 + 0 + 31 = 127
				dataBuf[nameOffset] = 0 // Empty name
				return dataBuf
			},
			wantSymbol: "",
			wantName:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := tt.setupBuf()

			symbol, name, err := ParseDataRequestedEvent(buf)

			if tt.wantErrText != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErrText)
				}
				if !strings.Contains(err.Error(), tt.wantErrText) {
					t.Errorf("expected error containing %q, got %v", tt.wantErrText, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if symbol != tt.wantSymbol {
				t.Errorf("Expected symbol %q, got %q", tt.wantSymbol, symbol)
			}
			if name != tt.wantName {
				t.Errorf("Expected name %q, got %q", tt.wantName, name)
			}
		})
	}
}
