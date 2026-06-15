package web

import (
	"card/db"
	"card/phoenix"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
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

const phoenixMaxBackoff = 30 * time.Second

// phoenixBackoff returns how long to wait before the next reconnect attempt,
// given the number of consecutive failures (0 means the last attempt
// succeeded). It grows exponentially (1s, 2s, 4s, …) capped at
// phoenixMaxBackoff so a flapping Phoenix never produces a hot reconnect loop.
func phoenixBackoff(failures int) time.Duration {
	if failures <= 0 {
		return 0
	}
	if failures > 6 { // 1<<6 == 64s already exceeds the cap; also guards against shift overflow
		return phoenixMaxBackoff
	}
	d := time.Duration(1<<(failures-1)) * time.Second
	if d > phoenixMaxBackoff {
		return phoenixMaxBackoff
	}
	return d
}

// interruptibleSleep waits for d, returning early if stop is closed so shutdown
// is not blocked by a pending backoff delay.
func interruptibleSleep(d time.Duration, stop <-chan struct{}) {
	if d <= 0 {
		return
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-stop:
	case <-t.C:
	}
}

// reconnectLoop calls connect, which is expected to block while the connection
// is alive and return when it fails to establish or drops. After each return
// it waits backoff(consecutiveFailures) before reconnecting; a successful
// connection (connect returns nil) resets the failure count. The loop exits
// when stop is closed.
func reconnectLoop(stop <-chan struct{}, connect func() error, backoff func(int) time.Duration) {
	failures := 0
	for {
		select {
		case <-stop:
			return
		default:
		}

		if err := connect(); err != nil {
			failures++
		} else {
			failures = 0
		}

		select {
		case <-stop:
			return
		default:
		}
		interruptibleSleep(backoff(failures), stop)
	}
}

// startPhoenixListener keeps a websocket open to Phoenix and broadcasts
// incoming payment events to all connected admin clients via the hub.
// It reconnects automatically: Phoenix may not be ready when the card service
// boots (a cold-start race) and may restart later, so a single dial is not
// enough — connectAndServePhoenix is retried with exponential backoff until
// app.stop is closed.
func (app *App) startPhoenixListener() {
	go reconnectLoop(app.stop, app.connectAndServePhoenix, phoenixBackoff)
}

// connectAndServePhoenix loads the Phoenix config, dials its websocket and
// blocks reading payment events until the connection fails to establish or
// drops, returning the error that ended it (nil on a clean close). It never
// retries itself — reconnectLoop owns the backoff and reconnection.
func (app *App) connectAndServePhoenix() error {
	cfg, err := ini.Load("/root/.phoenix/phoenix.conf")
	if err != nil {
		log.Info("phoenix config not available, will retry: ", err.Error())
		return err
	}

	hp := cfg.Section("").Key("http-password").String()
	h := http.Header{"Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte(":"+hp))}}
	c, _, err := websocket.DefaultDialer.Dial("ws://phoenix:9740/websocket", h)
	if err != nil {
		log.Info("phoenix websocket not available, will retry: ", err.Error())
		return err
	}

	log.Info("phoenix websocket listener connected")
	defer c.Close()

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Info("phoenix websocket closed, will reconnect: ", err.Error())
			return err
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

		// Mark any matching receipt as paid (for lightning address payments)
		if incomingPayment.IsPaid {
			db.Db_set_receipt_paid(app.db_write, incomingPayment.PaymentHash, "websocket")
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

// startReceiptPoller checks for unsettled receipts every 30s and marks them
// paid if Phoenix confirms the payment. This is a backstop for any payments
// missed by the WebSocket listener (e.g. during restarts).
func (app *App) startReceiptPoller() {
	go func() {
		for {
			time.Sleep(30 * time.Second)

			unpaid := db.Db_select_unpaid_receipts(app.db_read)
			for _, r := range unpaid {
				incoming, err := phoenix.GetIncomingPayment(r.PaymentHash)
				if err != nil {
					continue
				}
				if incoming.IsPaid {
					db.Db_set_receipt_paid(app.db_write, incoming.PaymentHash, "poller")
					log.Info("receipt poller settled: ", incoming.PaymentHash)

					event := wsPaymentEvent{
						Type:        "payment_received",
						AmountSat:   incoming.ReceivedSat,
						PaymentHash: incoming.PaymentHash,
						Timestamp:   incoming.CompletedAt / 1000,
					}
					eventJSON, err := json.Marshal(event)
					if err == nil {
						app.hub.broadcast(eventJSON)
					}
				}
			}
		}
	}()
}

func (app *App) CreateHandler_Websocket() http.HandlerFunc {
	hostDomain := db.Db_get_setting(app.db_read, "host_domain")

	return func(w http.ResponseWriter, r *http.Request) {
		// Authenticate admin session cookie
		c, err := r.Cookie("admin_session_token")
		if err != nil {
			http.Error(w, "not authenticated", http.StatusUnauthorized)
			return
		}
		adminSessionToken := db.Db_get_setting(app.db_read, "admin_session_token")
		if subtle.ConstantTimeCompare([]byte(c.Value), []byte(adminSessionToken)) != 1 {
			http.Error(w, "invalid session", http.StatusUnauthorized)
			return
		}
		sessionCreatedStr := db.Db_get_setting(app.db_read, "admin_session_created")
		if sessionCreatedStr != "" {
			sessionCreated, err := strconv.ParseInt(sessionCreatedStr, 10, 64)
			if err != nil || time.Now().Unix()-sessionCreated > 24*60*60 {
				http.Error(w, "session expired", http.StatusUnauthorized)
				return
			}
		}

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
