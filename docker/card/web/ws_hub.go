package web

import (
	"encoding/json"
	"sync"

	log "github.com/sirupsen/logrus"
)

// wsHub broadcasts messages to all connected websocket clients.
type wsHub struct {
	mu      sync.RWMutex
	clients map[chan []byte]struct{}
}

func newWsHub() *wsHub {
	return &wsHub{clients: make(map[chan []byte]struct{})}
}

func (h *wsHub) subscribe() chan []byte {
	ch := make(chan []byte, 16)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *wsHub) unsubscribe(ch chan []byte) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
}

func (h *wsHub) broadcast(msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- msg:
		default:
			// skip slow clients
		}
	}
}

func (app *App) broadcastPaymentSent(amountSat int, paymentHash string, timestamp int64) {
	event := wsPaymentEvent{
		Type:        "payment_sent",
		AmountSat:   amountSat,
		PaymentHash: paymentHash,
		Timestamp:   timestamp,
	}
	eventJSON, err := json.Marshal(event)
	if err != nil {
		log.Error("broadcastPaymentSent marshal error: ", err)
		return
	}
	app.hub.broadcast(eventJSON)
}
