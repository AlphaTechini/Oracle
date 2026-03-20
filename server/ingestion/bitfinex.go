package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const BitfinexWSURL = "wss://api-pub.bitfinex.com/ws/2"

// BitfinexMessage represents the initial subscription response
type BitfinexMessage struct {
	Event   string `json:"event"`
	Channel string `json:"channel"`
	Symbol  string `json:"symbol"`
	ChanID  int    `json:"chanId"`
}

// ChannelMap maps the channel ID to the trading symbol
var ChannelMap = make(map[int]string)

func ConnectToBitfinex(symbols []string) {
	for {
		log.Println("Connecting to Bitfinex WS...")
		c, _, err := websocket.DefaultDialer.Dial(BitfinexWSURL, nil)
		if err != nil {
			log.Println("Dial Error:", err)
			time.Sleep(5 * time.Second)
			continue
		}

		// Subscribe to ticker channels
		for _, sym := range symbols {
			subMsg := map[string]string{
				"event":   "subscribe",
				"channel": "ticker",
				"symbol":  sym,
			}
			err = c.WriteJSON(subMsg)
			if err != nil {
				log.Println("Subscribe Write Error:", err)
			}
		}

		// Read loop
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("Read Error (reconnecting):", err)
				c.Close()
				break // breaks the inner loop, reconnects the outer loop
			}
			processMessage(message)
		}
	}
}

func processMessage(msg []byte) {
	// Bitfinex sends JSON objects for events (like subscription confirmations)
	// and JSON arrays for data updates.

	if len(msg) > 0 && msg[0] == '{' {
		// Object: Decode as event
		var eventMsg BitfinexMessage
		if err := json.Unmarshal(msg, &eventMsg); err == nil {
			if eventMsg.Event == "subscribed" {
				ChannelMap[eventMsg.ChanID] = eventMsg.Symbol
				log.Printf("Subscribed to %s on channel %d\n", eventMsg.Symbol, eventMsg.ChanID)
			}
		}
		return
	}

	if len(msg) > 0 && msg[0] == '[' {
		// Array: Decode as data
		var data []interface{}
		if err := json.Unmarshal(msg, &data); err != nil {
			return
		}

		if len(data) < 2 {
			return
		}

		chanIDFloat, ok := data[0].(float64)
		if !ok {
			return
		}
		chanID := int(chanIDFloat)

		// Heartbeats are array [chanID, "hb"]
		if str, ok := data[1].(string); ok && str == "hb" {
			return
		}

		// Ticker data payload is [chanID, [bid, bidSize, ask, askSize, dailyChange, dailyChangePerc, lastPrice, volume, high, low]]
		payload, ok := data[1].([]interface{})
		if !ok || len(payload) < 7 {
			return
		}

		lastPrice, ok := payload[6].(float64)
		if !ok {
			return
		}

		symbol, ok := ChannelMap[chanID]
		if !ok {
			return
		}

		// Clean up symbol for standard formatting (e.g. tBTCUSD -> BTC)
		cleanSymbol := strings.TrimPrefix(symbol, "t")
		cleanSymbol = strings.TrimSuffix(cleanSymbol, "USD")

		// Update Redis
		err := UpdatePrice(cleanSymbol, lastPrice)
		if err != nil {
			log.Printf("Failed to update Redis for %s: %v\n", cleanSymbol, err)
		} else {
			fmt.Printf("Bitfinex Update -> %s: %f\n", cleanSymbol, lastPrice)
		}
	}
}
