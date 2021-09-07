package core

import (
	"fmt"
	"golang.org/x/net/websocket"
	"log"
)

type WsClient struct {
	url      string
	ws       *websocket.Conn
	listener WsStatusListener
}

func (w *WsClient) run() {
	origin := "http://127.0.0.1/"
	ws, err := websocket.Dial(w.url, "", origin)
	if err != nil {
		log.Fatal(err)
	}
	w.ws = ws

	for {
		var message []byte
		if err = websocket.Message.Receive(ws, &message); err != nil {
			fmt.Println("ws client can't receive")
			break
		}
		w.listener.OnMessage("", message)
	}
}
func (w *WsClient) send(data []byte) {
	if _, err := w.ws.Write(data); err != nil {
		log.Fatal(err)
	}
}

func newWsClient(url string, listener WsStatusListener) *WsClient {
	client := &WsClient{
		url:      url,
		listener: listener,
	}
	go client.run()
	return client
}
