package web

import (
	"card/phoenix"
	"card/util"
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

func WebsocketHandler(w http.ResponseWriter, r *http.Request) {

	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // TODO: authenticate
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	util.Check(err)

	log.Info("websocket from client is open")

	defer conn.Close()

	// open a websocket connection to phoenix
	// to receive payment notifications
	// and pass each one on to the client

	cfg, err := ini.Load("/root/.phoenix/phoenix.conf")
	util.Check(err)

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
				util.Check(err)

				log.Info("webSocketMessage : ", webSocketMessage)

				// look up the payment_hash to look up the description using GetIncomingPayment
				incomingPayment, err := phoenix.GetIncomingPayment(webSocketMessage.PaymentHash)
				util.Check(err)

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
			util.Check(err)

			continue
		}

		// TODO: make sure authentication is implemented before adding more here!

		// show message if not handled
		log.Info("websocket from client - unhandled message : ", string(message))
	}
}
