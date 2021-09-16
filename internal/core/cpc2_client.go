package core

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/aler9/gortsplib"
	"github.com/aler9/rtsp-simple-server/internal/logger"
	"github.com/aler9/rtsp-simple-server/internal/util"
	"log"
	"os/exec"
	"runtime"
)

type CPC2Client struct {
	api      CpcApi
	ws       *WsClient
	rtspHost string
	logger   CPCLogger
	play     bool
}

type CPCLogger interface {
	Log(logger.Level, string, ...interface{})
}

type respJSON struct {
	Uuid   string `json:"uuid"`
	Action string `json:"action"`
	Data   string `json:"data"`
}

func (c CPC2Client) OnAnnounce(ctx *gortsplib.ServerHandlerOnAnnounceCtx) {
	c.logger.Log(logger.Info, "OnAnnounce %s %s", ctx.Path, util.TimeUtil{}.GetTimeStr())
	// skip
	if c.ws != nil && false {
		url := ctx.Req.URL
		str, err := json.Marshal(&respJSON{
			Action: "ACTION_LIVE_READY",
			Uuid:   ctx.Path,
			Data:   "rtsp://" + c.rtspHost + url.Path,
		})
		if err != nil {
			fmt.Printf("")
		}
		c.ws.send([]byte(str))
	}
}

func (c *CPC2Client) OnSetup(ctx *gortsplib.ServerHandlerOnSetupCtx) {
	if c.play {
		return
	}
	c.play = true
	c.logger.Log(logger.Info, "OnSetup %s %s", ctx.Path, util.TimeUtil{}.GetTimeStr())
	// test
	if runtime.GOOS != "windows" {
		return
	}
	go func() {
		cmd := exec.Command("cmd", "/c", "ffplay -stats -infbuf  rtsp://127.0.0.1:8554/f1f7caeed5269c71ba4adf24d6e4a65f")
		errPipe, _ := cmd.StderrPipe()
		sc := bufio.NewScanner(errPipe)
		go func() {
			for sc.Scan() {
				line := sc.Text()
				c.logger.Log(logger.Info, "ffplay line %s %s", line, util.TimeUtil{}.GetTimeStr())
			}
		}()
		err := cmd.Start()
		if err != nil {
			log.Fatalln(err)
			return
		}
	}()
}

func (c CPC2Client) OnConnect(uuid string, kind string, dest string) {
	fmt.Printf("cpc2.OnConnect %s\n", uuid)
}

func (c CPC2Client) OnMessage(uuid string, data []byte) {
	fmt.Printf("cpc2.OnMessage %s %s\n", uuid, string(data))
}

func (c CPC2Client) OnDisconnect(uuid string) {
	fmt.Printf("cpc2.OnDisconnect %s \n", uuid)
}

type CpcApi interface {
	//OnRpcGetLiveStatus 获取指定流状态 未实现
	OnRpcGetLiveStatus(uuid string) string
	//OnRpcGetLiveList 获取流媒体列表 未实现
	OnRpcGetLiveList() string
	//OnRpcReqDisconnect 断开指定连接 未实现
	OnRpcReqDisconnect(uuid string) string
}
