package web

import (
	"card/util"
	"encoding/base64"
	"log"
	"net/http"

	"github.com/go-ini/ini"
	"github.com/gorilla/websocket"
)

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
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %ss", message)
			err = conn.WriteMessage(websocket.TextMessage, message)
		}
	}()

	for {
		// read message from client
		_, message, err := conn.ReadMessage()
		util.Check(err)

		// show message
		log.Printf("Received message: %s", message)

		//send message to client
		err = conn.WriteMessage(websocket.TextMessage, message)
		util.Check(err)
	}
}
