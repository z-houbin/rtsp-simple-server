package core

import (
	"github.com/aler9/rtsp-simple-server/internal/logger"
	"golang.org/x/net/websocket"
	"log"
	"time"
)

type WsClient struct {
	url      string
	ws       *websocket.Conn
	listener WsStatusListener
	logger   CPCLogger
}

func (w *WsClient) run() {
	w.logger.Log(logger.Info, "connect cpc2 live ws...")
	origin := "http://127.0.0.1/"
	ws, err := websocket.Dial(w.url, "", origin)
	if err != nil {
		log.Fatal(err)
	}
	w.ws = ws

	// 定时ping
	go func() {
		timer1 := time.NewTicker(10 * time.Second)
		for {
			select {
			case <-timer1.C:
				w.send([]byte("{\"ping\":\"true\"}"))
			}
		}
	}()

	w.logger.Log(logger.Info, "connect cpc2 live ws success")

	for {
		var message []byte
		if err = websocket.Message.Receive(ws, &message); err != nil {
			w.logger.Log(logger.Warn, "[%s] cpc ws client interrupt")
			break
		}
		w.listener.OnMessage("", message)
	}

	w.logger.Log(logger.Info, "connect cpc2 live ws disconnected")
}
func (w *WsClient) send(data []byte) {
	if _, err := w.ws.Write(data); err != nil {
		log.Fatal(err)
	}
}

func newWsClient(url string, listener WsStatusListener, cpcLogger CPCLogger) *WsClient {
	client := &WsClient{
		url:      url,
		listener: listener,
		logger:   cpcLogger,
	}
	go client.run()
	return client
}
