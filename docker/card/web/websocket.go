package web

import (
	"card/db"
	"card/phoenix"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/go-ini/ini"
	"github.com/gorilla/websocket"
)

type wsPaymentEvent struct {
	Type        string `json:"type"`
	AmountSat   int    `json:"amountSat"`
	PaymentHash string `json:"paymentHash"`
	Timestamp   int64  `json:"timestamp"`
}

type WebSocketMessage struct {
	Type        string `json:"type"`
	AmountSat   int    `json:"amountSat"`
	PaymentHash string `json:"paymentHash"`
	Timestamp   int64  `json:"timestamp"`
	ExternalID  string `json:"externalId,omitempty"`
	PayerNote   string `json:"payerNote,omitempty"`
	PayerKey    string `json:"payerKey,omitempty"`
}

// startPhoenixListener opens a single websocket to Phoenix and broadcasts
// incoming payment events to all connected admin clients via the hub.
// It runs once and reconnects are not needed (Phoenix connection is long-lived).
func (app *App) startPhoenixListener() {
	cfg, err := ini.Load("/root/.phoenix/phoenix.conf")
	if err != nil {
		log.Error("failed to load phoenix config: ", err)
		return
	}

	hp := cfg.Section("").Key("http-password").String()
	h := http.Header{"Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte(":" + hp))}}
	c, _, err := websocket.DefaultDialer.Dial("ws://phoenix:9740/websocket", h)
	if err != nil {
		log.Info("phoenix websocket not available: ", err.Error())
		return
	}

	go func() {
		defer c.Close()
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Info("websocket to phoenix is closing: ", err.Error())
				return
			}

			log.Info("phoenix ws message: ", string(message))

			var wsMsg WebSocketMessage
			err = json.Unmarshal(message, &wsMsg)
			if err != nil {
				log.Error("websocket json unmarshal error: ", err)
				continue
			}

			incomingPayment, err := phoenix.GetIncomingPayment(wsMsg.PaymentHash)
			if err != nil {
				log.Error("phoenix GetIncomingPayment error: ", err)
				continue
			}

			event := wsPaymentEvent{
				Type:        "payment_received",
				AmountSat:   incomingPayment.ReceivedSat,
				PaymentHash: incomingPayment.PaymentHash,
				Timestamp:   incomingPayment.CompletedAt / 1000,
			}

			eventJSON, err := json.Marshal(event)
			if err != nil {
				log.Error("websocket json marshal error: ", err)
				continue
			}

			app.hub.broadcast(eventJSON)
		}
	}()
}

// startChannelPoller polls Phoenix channel status every 30s and broadcasts
// a channel_update event when any channel state changes.
func (app *App) startChannelPoller() {
	go func() {
		var lastStates map[string]string

		for {
			channels, err := phoenix.ListChannels()
			if err != nil {
				log.Warn("channel poll error: ", err)
				time.Sleep(30 * time.Second)
				continue
			}

			states := make(map[string]string, len(channels))
			for _, ch := range channels {
				states[ch.ChannelID] = ch.State
			}

			if lastStates != nil {
				changed := len(states) != len(lastStates)
				if !changed {
					for id, state := range states {
						if lastStates[id] != state {
							changed = true
							break
						}
					}
				}
				if changed {
					event := map[string]string{"type": "channel_update"}
					eventJSON, err := json.Marshal(event)
					if err == nil {
						app.hub.broadcast(eventJSON)
					}
				}
			}

			lastStates = states
			time.Sleep(30 * time.Second)
		}
	}()
}

func (app *App) CreateHandler_Websocket() http.HandlerFunc {
	hostDomain := db.Db_get_setting(app.db_conn, "host_domain")

	// Start Phoenix listener once (shared across all clients)
	var once sync.Once

	return func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() {
			app.startPhoenixListener()
			app.startChannelPoller()
		})

		var upgrader = websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true // allow non-browser clients
				}
				return origin == "https://"+hostDomain
			},
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Error("websocket upgrade error: ", err)
			return
		}

		log.Info("websocket from client is open")
		defer conn.Close()

		// Subscribe to hub broadcasts
		ch := app.hub.subscribe()
		defer app.hub.unsubscribe(ch)

		// Forward hub messages to client
		done := make(chan struct{})
		go func() {
			defer close(done)
			for msg := range ch {
				err := conn.WriteMessage(websocket.TextMessage, msg)
				if err != nil {
					log.Warn("websocket write error:", err)
					return
				}
			}
		}()

		// Read from client (ping/pong keepalive)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Info("websocket from client is closing: ", err.Error())
				return
			}

			if string(message) == "ping" {
				err = conn.WriteMessage(websocket.TextMessage, []byte("pong"))
				if err != nil {
					log.Error("websocket write error: ", err)
					return
				}
				continue
			}

			log.Info("websocket from client - unhandled message: ", string(message))
		}
	}
}
