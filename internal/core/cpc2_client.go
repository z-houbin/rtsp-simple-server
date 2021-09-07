package core

import (
	"encoding/json"
	"fmt"
	"github.com/aler9/gortsplib"
)

type CPC2Client struct {
	api CpcApi
	ws  *WsClient
}

type respJSON struct {
	Uuid   string `json:"uuid"`
	Action string `json:"action"`
	Data   string `json:"data"`
}

func (c CPC2Client) OnAnnounce(ctx *gortsplib.ServerHandlerOnAnnounceCtx) {
	fmt.Printf("OnAnnounce %s\n", ctx.Path)

	if c.ws != nil {
		url := ctx.Req.URL
		//TODO 返回地址待优化
		str, err := json.Marshal(&respJSON{
			Action: "ACTION_LIVE_READY",
			Uuid:   ctx.Path,
			Data:   "rtsp://" + url.Host + url.Path,
		})
		if err != nil {
			fmt.Printf("")
		}
		c.ws.send([]byte(str))
	}
}

func (c CPC2Client) OnConnect(uuid string) {
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
