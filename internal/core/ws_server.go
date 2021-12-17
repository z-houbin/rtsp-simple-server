package core

import (
	"encoding/json"
	"fmt"
	"github.com/aler9/rtsp-simple-server/internal/conf"
	"github.com/aler9/rtsp-simple-server/internal/logger"
	"github.com/gorilla/websocket"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type WsServer struct {
	port int
	//前端webm数据监听
	cameraListener WsStatusListener
	//websocket      会话
	client map[string]*websocket.Conn
	conf   conf.Conf
	logger CPCLogger
	ws     *WsClient
}

type WsStatusListener interface {
	//OnConnect 客户端连接
	OnConnect(uuid string, kind string, dest string, ffmpegArgs string)
	//OnMessage 客户端消息
	OnMessage(uuid string, data []byte)
	//OnDisconnect 客户端断开
	OnDisconnect(uuid string)
}

var upGrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	//解决跨域问题
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (s *WsServer) run() {
	s.logger.Log(logger.Info, "ws 1")

	wsHandler := func(w http.ResponseWriter, r *http.Request) {
		s.logger.Log(logger.Info, "ws 4")

		uuid := strings.Split(r.URL.Path, "/")[1]
		kind := "default"
		dest := "rtsp:" + s.conf.RtspPushAddress + "/" + uuid
		conn, err := upGrader.Upgrade(w, r, nil)
		if err != nil {
			s.logger.Log(logger.Info, "upgrade failed", err.Error())
			return
		}

		//get uuid params
		s.client[uuid] = conn

		// notify android client
		go func() {
			s.logger.Log(logger.Info, "ws 5")
			s.notifyStreamReady(uuid, dest)
		}()

		//ff handler
		s.cameraListener.OnConnect(uuid, kind, dest, s.conf.FfmpegArgs)
		defer func(conn *websocket.Conn) {
			err := conn.Close()
			if err != nil {
				s.logger.Log(logger.Warn, err.Error())
			}
		}(conn)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				s.logger.Log(logger.Warn, err.Error())
				break
			}
			s.cameraListener.OnMessage(uuid, message)
		}
		s.logger.Log(logger.Info, "ws 6")
		s.cameraListener.OnDisconnect(uuid)
		// 通知cpc前端断开
		s.notifyStreamClose(uuid)
	}

	http.HandleFunc("/", wsHandler)
	s.logger.Log(logger.Info, "ws 2")

	_, err := os.Lstat("./cert.crt")
	if !os.IsNotExist(err) {
		// wss:
		if err := http.ListenAndServeTLS(":"+strconv.Itoa(s.port), "cert.crt", "cert.key", nil); err != nil {
			s.logger.Log(logger.Warn, "camera websocket listen err %s", err)
		}
	} else {
		if err := http.ListenAndServe(":"+strconv.Itoa(s.port), nil); err != nil {
			s.logger.Log(logger.Warn, "camera websocket listen err %s", err)
		}
	}
	s.logger.Log(logger.Info, "ws 3")
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
		err := client.WriteMessage(websocket.BinaryMessage, data)
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
