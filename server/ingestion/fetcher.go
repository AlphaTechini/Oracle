package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

// PriceSource represents a common interface for different data providers
type PriceSource interface {
	FetchPrice(ctx context.Context, symbol string) (float64, error)
	Name() string
}

// Global scaling factor: 10^8
const PriceScale = 100000000

// --- Implementations of PriceSource ---

type BinanceSource struct{}
func (s BinanceSource) Name() string { return "Binance" }
func (s BinanceSource) FetchPrice(ctx context.Context, symbol string) (float64, error) {
	baseCurrency := os.Getenv("BASE_CURRENCY")
	if baseCurrency == "" {
		baseCurrency = "USDT"
	}
	url := fmt.Sprintf("https://api.binance.com/api/v3/ticker/price?symbol=%s%s", symbol, baseCurrency)
	resp, err := http.Get(url)
	if err != nil { return 0, err }
	defer resp.Body.Close()
	var data struct { Price string `json:"price"` }
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil { return 0, err }
	var price float64
	fmt.Sscanf(data.Price, "%f", &price)
	return price, nil
}

type BitfinexSource struct{}
func (s BitfinexSource) Name() string { return "Bitfinex" }
func (s BitfinexSource) FetchPrice(ctx context.Context, symbol string) (float64, error) {
	// Bitfinex uses USD for spot, symbols prefixed with 't'
	url := fmt.Sprintf("https://api-pub.bitfinex.com/v2/ticker/t%sUSD", symbol)
	resp, err := http.Get(url)
	if err != nil { return 0, err }
	defer resp.Body.Close()
	var data []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil { return 0, err }
	if len(data) < 7 { return 0, fmt.Errorf("invalid bitfinex response") }
	return data[6].(float64), nil
}

type CoinbaseSource struct{}
func (s CoinbaseSource) Name() string { return "Coinbase" }
func (s CoinbaseSource) FetchPrice(ctx context.Context, symbol string) (float64, error) {
	url := fmt.Sprintf("https://api.coinbase.com/v2/prices/%s-USD/spot", symbol)
	resp, err := http.Get(url)
	if err != nil { return 0, err }
	defer resp.Body.Close()
	var data struct { Data struct { Amount string `json:"amount"` } `json:"data"` }
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil { return 0, err }
	var price float64
	fmt.Sscanf(data.Data.Amount, "%f", &price)
	return price, nil
}

type CoinGeckoSource struct{
	TokenName string
}
func (s CoinGeckoSource) Name() string { return "CoinGecko" }
func (s CoinGeckoSource) FetchPrice(ctx context.Context, symbol string) (float64, error) {
	// Use the passed name if available, otherwise fallback to symbol lowercase
	id := strings.ToLower(s.TokenName)
	if id == "" {
		id = strings.ToLower(symbol)
	}
	
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd", id)
	resp, err := http.Get(url)
	if err != nil { return 0, err }
	defer resp.Body.Close()
	var data map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil { return 0, err }
	
	if val, ok := data[id]["usd"]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("id %s not found in coingecko response", id)
}

type BybitSource struct{}
func (s BybitSource) Name() string { return "Bybit" }
func (s BybitSource) FetchPrice(ctx context.Context, symbol string) (float64, error) {
	baseCurrency := os.Getenv("BASE_CURRENCY")
	if baseCurrency == "" {
		baseCurrency = "USDT"
	}
	url := fmt.Sprintf("https://api.bybit.com/v5/market/tickers?category=spot&symbol=%s%s", symbol, baseCurrency)
	resp, err := http.Get(url)
	if err != nil { return 0, err }
	defer resp.Body.Close()
	var data struct { Result struct { List []struct { LastPrice string `json:"lastPrice"` } `json:"list"` } `json:"result"` }
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil { return 0, err }
	if len(data.Result.List) == 0 { return 0, fmt.Errorf("no bybit data") }
	var price float64
	fmt.Sscanf(data.Result.List[0].LastPrice, "%f", &price)
	return price, nil
}

// FetchMedian fetches prices concurrently from all sources and returns the median scaled by 10^8
func FetchMedian(symbol string, name string) uint64 {
	sources := []PriceSource{
		BinanceSource{}, 
		BitfinexSource{}, 
		CoinbaseSource{}, 
		CoinGeckoSource{TokenName: name}, 
		BybitSource{},
	}
	var prices []float64

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, s := range sources {
		wg.Add(1)
		go func(source PriceSource) {
			defer wg.Done()
			
			price, err := source.FetchPrice(ctx, symbol)
			if err != nil {
				log.Printf("[%s] Fetch Error: %v", source.Name(), err)
				return
			}
			
			mu.Lock()
			prices = append(prices, price)
			mu.Unlock()
		}(s)
	}

	wg.Wait()

	if len(prices) == 0 { 
		return 0 
	}
	
	sort.Float64s(prices)
	median := prices[len(prices)/2]
	return uint64(math.Round(median * PriceScale))
}

// SignPrice signs the (reqId, price) using the node's private key
func SignPrice(privKeyHex string, reqId [32]byte, price uint64) ([]byte, error) {
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privKeyHex, "0x"))
	if err != nil { return nil, err }

	// Solidity abi.encodePacked(reqId, uint256(price)) result is 64 bytes
	// [32 bytes reqId][32 bytes price]
	payload := make([]byte, 64)
	copy(payload[0:32], reqId[:])
	
	// Price as uint256 (32 bytes, big-endian)
	priceBN := make([]byte, 32)
	for i := 0; i < 8; i++ {
		priceBN[31-i] = byte(price >> (i * 8))
	}
	copy(payload[32:64], priceBN)

	hash := crypto.Keccak256Hash(payload)
	
	// Ethereum Signed Message Prefix
	prefix := []byte("\x19Ethereum Signed Message:\n32")
	prefixedHash := crypto.Keccak256Hash(append(prefix, hash.Bytes()...))

	signature, err := crypto.Sign(prefixedHash.Bytes(), privateKey)
	if err != nil { return nil, err }
	
	// Add 27 to V (Ethereum recovery ID)
	signature[64] += 27
	return signature, nil
}
