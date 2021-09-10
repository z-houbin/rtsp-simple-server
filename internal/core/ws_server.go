package core

import (
	"encoding/json"
	"fmt"
	"github.com/aler9/rtsp-simple-server/internal/conf"
	"github.com/aler9/rtsp-simple-server/internal/logger"
	"golang.org/x/net/websocket"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type WsServer struct {
	port     int
	listener WsStatusListener
	client   map[string]*websocket.Conn
	conf     conf.Conf
	logger   CPCLogger
	ws       *WsClient
}

type WsStatusListener interface {
	//OnConnect 客户端连接
	OnConnect(uuid string, rtspHost string)
	//OnMessage 客户端消息
	OnMessage(uuid string, data []byte)
	//OnDisconnect 客户端断开
	OnDisconnect(uuid string)
}

func (s *WsServer) run() {
	wsHandler := func(ws *websocket.Conn) {
		var err error
		// get uuid params
		uuid := strings.ReplaceAll(ws.Request().URL.String(), "/", "")
		s.client[uuid] = ws

		// notify android client
		go func() {
			s.notifyStreamReady(uuid)
		}()

		// ff handler
		s.listener.OnConnect(uuid, s.conf.RtspPushAddress)

		// loop receive
		for {
			var message []byte
			if err = websocket.Message.Receive(ws, &message); err != nil {
				s.logger.Log(logger.Warn, "camera websocket interrupt")
				break
			}
			s.listener.OnMessage(uuid, message)
		}
		s.listener.OnDisconnect(uuid)
	}

	http.Handle("/", websocket.Handler(wsHandler))

	_, err := os.Lstat("./cert.crt")
	if !os.IsNotExist(err) {
		// wss://
		if err := http.ListenAndServeTLS(":"+strconv.Itoa(s.port), "cert.crt", "cert.key", nil); err != nil {
			s.logger.Log(logger.Warn, "camera websocket listen err %s", err)
		}
	} else {
		if err := http.ListenAndServe(":"+strconv.Itoa(s.port), nil); err != nil {
			s.logger.Log(logger.Warn, "camera websocket listen err %s", err)
		}
	}
}

func (s *WsServer) notifyStreamReady(uuid string) {
	str, err := json.Marshal(&respJSON{
		Action: "ACTION_LIVE_READY",
		Uuid:   uuid,
		Data:   "rtsp://" + s.conf.RtspPushAddress + "/" + uuid,
	})
	if err != nil {
		fmt.Printf("")
	}
	if s.ws != nil {
		s.ws.send(str)
	}
}

func (s *WsServer) send(uuid string, data []byte) {
	client, _ := s.client[uuid]
	if client != nil {
		err := websocket.Message.Send(client, data)
		if err != nil {
			s.logger.Log(logger.Warn, "camera websocket send err %s %s", uuid, err)
		}
	}
}

func RunCameraWebSocketServer(config conf.Conf, listener WsStatusListener, logger CPCLogger) *WsServer {
	server := &WsServer{
		port:     config.CameraWebSocketPort,
		conf:     config,
		listener: listener,
		client:   map[string]*websocket.Conn{},
		logger:   logger,
	}
	go server.run()
	return server
}
