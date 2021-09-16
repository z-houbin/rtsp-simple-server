package core

import (
	"encoding/json"
	"fmt"
	"github.com/aler9/rtsp-simple-server/internal/conf"
	"github.com/aler9/rtsp-simple-server/internal/logger"
	"golang.org/x/net/websocket"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type WsServer struct {
	port int
	// 前端webm数据监听
	cameraListener WsStatusListener
	// websocket 会话
	client map[string]*websocket.Conn
	conf   conf.Conf
	logger CPCLogger
	ws     *WsClient
}

type WsStatusListener interface {
	//OnConnect 客户端连接
	OnConnect(uuid string, kind string, dest string)
	//OnMessage 客户端消息
	OnMessage(uuid string, data []byte)
	//OnDisconnect 客户端断开
	OnDisconnect(uuid string)
}

func (s *WsServer) run() {
	wsHandler := func(ws *websocket.Conn) {
		var err error
		var uuid, kind, dest string

		// get uuid/dest params
		re := regexp.MustCompile(`^/([^/]+)/([^/]+)/(.*)$`)
		url := ws.Request().URL.String()
		match := re.FindStringSubmatch(url)

		if match != nil {
			uuid = match[1]
			kind = match[2]
			dest = "rtsp://" + match[3]
		} else {
			uuid = strings.ReplaceAll(url, "/", "")
			kind = "default"
			dest = "rtsp://" + s.conf.RtspPushAddress + "/" + uuid
		}

		// get uuid params
		s.client[uuid] = ws

		// notify android client
		go func() {
			s.notifyStreamReady(uuid, dest)
		}()

		// ff handler
		s.cameraListener.OnConnect(uuid, kind, dest)

		// loop receive
		for {
			var message []byte
			if err = websocket.Message.Receive(ws, &message); err != nil {
				s.logger.Log(logger.Warn, "camera websocket interrupt")
				break
			}
			s.cameraListener.OnMessage(uuid, message)
		}
		s.cameraListener.OnDisconnect(uuid)
		// 通知cpc前端断开
		s.notifyStreamClose(uuid)
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

func (s *WsServer) notifyStreamReady(uuid string, dest string) {
	str, err := json.Marshal(&respJSON{
		Action: "ACTION_LIVE_READY",
		Uuid:   uuid,
		Data:   dest,
	})
	if err != nil {
		fmt.Printf("")
	}
	if s.ws != nil {
		s.ws.send(str)
	}
}

func (s *WsServer) notifyStreamClose(uuid string) {
	str, err := json.Marshal(&respJSON{
		Action: "ACTION_LIVE_CLOSE",
		Uuid:   uuid,
		Data:   "",
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
		port:           config.CameraWebSocketPort,
		conf:           config,
		cameraListener: listener,
		client:         map[string]*websocket.Conn{},
		logger:         logger,
	}
	go server.run()
	return server
}
