package web

import (
	"card/db"
	"card/phoenix"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/go-ini/ini"
	"github.com/gorilla/websocket"
)

type WebSocketMessage struct {
	Type        string `json:"type"`
	AmountSat   int    `json:"amountSat"`
	PaymentHash string `json:"paymentHash"`
	Timestamp   int64  `json:"timestamp"`
	ExternalID  string `json:"externalId,omitempty"`
	PayerNote   string `json:"payerNote,omitempty"`
	PayerKey    string `json:"payerKey,omitempty"`
}

func (app *App) CreateHandler_Websocket() http.HandlerFunc {
	hostDomain := db.Db_get_setting(app.db_conn, "host_domain")
	return func(w http.ResponseWriter, r *http.Request) {

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

		// open a websocket connection to phoenix
		// to receive payment notifications
		// and pass each one on to the client

		cfg, err := ini.Load("/root/.phoenix/phoenix.conf")
		if err != nil {
			log.Error("failed to load phoenix config: ", err)
			return
		}

		hp := cfg.Section("").Key("http-password").String()

		// https://github.com/gorilla/websocket/issues/209#issuecomment-275419998
		h := http.Header{"Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte(":"+hp))}}
		c, _, err := websocket.DefaultDialer.Dial("ws://phoenix:9740/websocket", h)
		if err != nil {
			log.Info("phoenix not available : ", err.Error())
		} else {
			defer c.Close()

			done := make(chan struct{})

			go func() {
				defer close(done)
				for {
					_, message, err := c.ReadMessage()
					if err != nil {
						log.Info("websocket to phoenix is closing : ", err.Error())
						return
					}

					// `message` contains the websocket message as []byte from Phoenix server
					log.Info("message : ", string(message))

					// decode the JSON into a struct
					var webSocketMessage WebSocketMessage

					err = json.Unmarshal(message, &webSocketMessage)
					if err != nil {
						log.Error("websocket json unmarshal error: ", err)
						return
					}

					log.Info("webSocketMessage : ", webSocketMessage)

					// look up the payment_hash to look up the description using GetIncomingPayment
					incomingPayment, err := phoenix.GetIncomingPayment(webSocketMessage.PaymentHash)
					if err != nil {
						log.Error("phoenix GetIncomingPayment error: ", err)
						return
					}

					log.Info("incomingPayment : ", incomingPayment)

					// TODO: send JSON encoded data to add a tx row
					// TODO: send JSON encoded data to update the totals
					now := time.Now()
					now_string := now.Format("15:04:05")
					err = conn.WriteMessage(websocket.TextMessage,
						[]byte(now_string+" UTC, "+
							strconv.Itoa(incomingPayment.ReceivedSat)+" sats received, "+
							strconv.Itoa(incomingPayment.Fees)+" sats fees,"+
							" message: "+webSocketMessage.PayerNote))
					if err != nil {
						log.Warning("websocket write error :", err)
						return
					}
				}
			}()
		}

		for {
			// read message from client
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Info("websocket from client is closing : ", err.Error())
				return
			}

			// handle "ping" message
			if string(message) == "ping" {

				//send message to client
				err = conn.WriteMessage(websocket.TextMessage, []byte("pong"))
				if err != nil {
					log.Error("websocket write error: ", err)
					return
				}

				continue
			}

			// TODO: make sure authentication is implemented before adding more here!

			// show message if not handled
			log.Info("websocket from client - unhandled message : ", string(message))
		}
	}
}
