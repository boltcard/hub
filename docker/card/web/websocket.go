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
}

func WebsocketHandler(w http.ResponseWriter, r *http.Request) {

	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	util.Check(err)

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
	util.Check(err)
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Warning("websocket read error :", err)
				return
			}

			// `message` contains the websocket message as []byte from Phoenix server

			// TODO:
			// decode the JSON into a struct
			// look up the payment_hash to look up the description using GetIncomingPayment

			var webSocketMessage WebSocketMessage

			err = json.Unmarshal(message, &webSocketMessage)
			util.Check(err)

			log.Info("webSocketMessage : ", webSocketMessage)

			incomingPayment, err := phoenix.GetIncomingPayment(webSocketMessage.PaymentHash)
			util.Check(err)

			log.Info("incomingPayment : ", incomingPayment)

			// message_string := string(message)
			now := time.Now()
			now_string := now.Format("15:04:05")
			err = conn.WriteMessage(websocket.TextMessage,
				[]byte(now_string+" UTC, "+
					strconv.Itoa(incomingPayment.ReceivedSat)+" sats received, "+
					strconv.Itoa(incomingPayment.Fees)+" sats fees"))
			if err != nil {
				log.Warning("websocket write error :", err)
				return
			}
		}
	}()

	for {
		// read message from client
		_, message, err := conn.ReadMessage()
		util.Check(err)

		// show message
		//log.Info("websocket rx : ", string(message))

		if string(message) == "ping" {

			//send message to client
			err = conn.WriteMessage(websocket.TextMessage, []byte("pong"))
			util.Check(err)
		}
	}
}
