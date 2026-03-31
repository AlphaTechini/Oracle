package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

// Mocking a price source for deterministic median testing
type MockSource struct {
	name  string
	price float64
	err   error
}

func (s MockSource) Name() string { return s.name }
func (s MockSource) FetchPrice(ctx context.Context, symbol string) (float64, error) {
	return s.price, s.err
}

func TestFetchMedianWithSources(t *testing.T) {
	sources := []PriceSource{
		MockSource{name: "Mock1", price: 100.5, err: nil},
		MockSource{name: "Mock2", price: 102.0, err: nil},
		MockSource{name: "Mock3", price: 99.0, err: nil},
		MockSource{name: "Mock4", price: 101.2, err: nil},
		MockSource{name: "Mock5", price: 105.0, err: nil},
	}

	median := FetchMedianWithSources("ETH", "Ethereum", sources)
	expected := uint64(10120000000) // 101.2 * 10^8
	if median != expected {
		t.Errorf("Expected median %d, got %d", expected, median)
	}
}

func TestSignPrice(t *testing.T) {
	// A known private key
	privKeyHex := "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

	var reqId [32]byte
	copy(reqId[:], []byte("test-request-id-padding-here-001"))

	price := uint64(10000000000) // 100.00

	sig, err := SignPrice(privKeyHex, reqId, price)
	if err != nil {
		t.Fatalf("SignPrice failed: %v", err)
	}

	if len(sig) != 65 {
		t.Errorf("Expected signature length 65, got %d", len(sig))
	}

	// Verify signature
	// We need to recreate the prefixed hash
	payload := make([]byte, 64)
	copy(payload[0:32], reqId[:])

	priceBN := make([]byte, 32)
	for i := 0; i < 8; i++ {
		priceBN[31-i] = byte(price >> (i * 8))
	}
	copy(payload[32:64], priceBN)

	hash := crypto.Keccak256Hash(payload)
	prefix := []byte("\x19Ethereum Signed Message:\n32")
	prefixedHash := crypto.Keccak256Hash(append(prefix, hash.Bytes()...))

	// Recover pubkey
	sigCopy := make([]byte, 65)
	copy(sigCopy, sig)
	sigCopy[64] -= 27 // Remove 27 from V for recovery

	pubKey, err := crypto.SigToPub(prefixedHash.Bytes(), sigCopy)
	if err != nil {
		t.Fatalf("Failed to recover pubkey: %v", err)
	}

	recoveredAddr := crypto.PubkeyToAddress(*pubKey)

	// Get original address
	privKey, _ := crypto.HexToECDSA(privKeyHex)
	originalAddr := crypto.PubkeyToAddress(privKey.PublicKey)

	if recoveredAddr.Hex() != originalAddr.Hex() {
		t.Errorf("Signature recovery mismatch. Expected %s, got %s", originalAddr.Hex(), recoveredAddr.Hex())
	}
}

// Test the individual fetcher with httptest
func TestBinanceSource(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"price": "123.45"}`))
	}))
	defer ts.Close()

	source := BinanceSource{BaseURL: ts.URL}
	if source.Name() != "Binance" {
		t.Errorf("Expected name Binance, got %s", source.Name())
	}

	price, err := source.FetchPrice(context.Background(), "ETH")
	if err != nil {
		t.Fatalf("FetchPrice failed: %v", err)
	}

	if price != 123.45 {
		t.Errorf("Expected price 123.45, got %f", price)
	}
}
