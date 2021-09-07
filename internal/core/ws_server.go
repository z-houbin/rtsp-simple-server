package core

import (
	"fmt"
	"golang.org/x/net/websocket"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type WsServer struct {
	port     int
	listener WsStatusListener
	client   map[string]*websocket.Conn
}

type WsStatusListener interface {
	//OnConnect 客户端连接
	OnConnect(uuid string)
	//OnMessage 客户端消息
	OnMessage(uuid string, data []byte)
	//OnDisconnect 客户端断开
	OnDisconnect(uuid string)
}

func (s WsServer) run() {
	wsHandler := func(ws *websocket.Conn) {
		var err error
		// get uuid params
		uuid := strings.ReplaceAll(ws.Request().URL.String(), "/", "")

		s.listener.OnConnect(uuid)
		// loop receive
		for {
			var message []byte
			if err = websocket.Message.Receive(ws, &message); err != nil {
				fmt.Println("Can't receive")
				break
			}
			s.listener.OnMessage(uuid, message)
		}
		s.listener.OnDisconnect(uuid)
	}
	http.Handle("/", websocket.Handler(wsHandler))
	if err := http.ListenAndServe(":"+strconv.Itoa(s.port), nil); err != nil {
		log.Fatal("ws server error:", err)
	}
}

func (s WsServer) send(uuid string, data []byte) {
	client, _ := s.client[uuid]
	if client != nil {
		err := websocket.Message.Send(client, data)
		if err != nil {
			log.Fatalf("send %s %s", uuid, err)
		}
	}
}

func RunCameraWebSocketServer(port int, listener WsStatusListener) *WsServer {
	server := &WsServer{port: port, listener: listener, client: map[string]*websocket.Conn{}}
	go server.run()
	return server
}
