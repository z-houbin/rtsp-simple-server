package core

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
)

type FFHandler struct {
	connect map[string]*ffProcessor
}

//OnConnect imp WsStatusListener
func (h *FFHandler) OnConnect(uuid string) {
	fmt.Printf("ffhandler OnConnect:%s\n", uuid)
	processor := &ffProcessor{
		uuid: uuid,
	}
	processor.init(uuid)
	h.connect[uuid] = processor
}

//OnMessage imp WsStatusListener
func (h *FFHandler) OnMessage(uuid string, data []byte) {
	processor := h.connect[uuid]
	if processor == nil {
		fmt.Printf("connect lost %s\n", uuid)
		return
	}
	if processor.initSuccess {
		processor.handle(data)
	}
}

//OnDisconnect imp WsStatusListener
func (h *FFHandler) OnDisconnect(uuid string) {
	fmt.Printf("ffhandler OnDisconnect:%s \n", uuid)
	delete(h.connect, uuid)
}

type ffProcessor struct {
	uuid        string
	cmd         *exec.Cmd
	stdIn       io.WriteCloser
	initSuccess bool
}

func (p *ffProcessor) init(uuid string) {
	sysType := runtime.GOOS
	var rtspURL string
	//TODO 运行方式和地址待优化,cpc2_client OnAnnounce 地址从这个地址取的
	//TODO 异常处理待优化,可能导致整个rtsp停止运行
	var cmd *exec.Cmd
	if sysType == "windows" {
		// windows
		rtspURL = "rtsp://127.0.0.1:8554/" + uuid
		cmd = exec.Command("cmd", "/c",
			"ffmpeg", "-y",
			"-i", "-",
			"-c:v", "libx264",
			"-preset:v", "ultrafast",
			"-tune", "zerolatency",
			"-c:a", "aac",
			"-ar", "44100",
			"-b:a", "300k",
			"-async", "1",
			"-filter:v", "fps=20",
			"-use_wallclock_as_timestamps", "1",
			"-bufsize", "10240",
			"-g", "12",
			"-rtsp_transport", "tcp",
			"-f", "rtsp",
			rtspURL)
	} else {
		rtspURL = "rtsp://123.60.28.115:8554/" + uuid
		cmd = exec.Command("ffmpeg", "-y",
			"-i", "-",
			"-c:v", "libx264",
			"-preset:v", "ultrafast",
			"-tune", "zerolatency",
			"-c:a", "aac",
			"-ar", "44100",
			"-b:v", "2M",
			"-async", "1",
			"-filter:v", "fps=20",
			"-use_wallclock_as_timestamps", "1",
			"-bufsize", "10240",
			"-g", "12",
			"-rtsp_transport", "tcp",
			"-f", "rtsp",
			rtspURL,
		)
	}

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	stdIn, err := cmd.StdinPipe()
	if err != nil {
		log.Fatalln(err)
	}

	err = cmd.Start()
	if err != nil {
		log.Fatalln(err)
	}

	go func() {
		_ = cmd.Wait()
	}()

	p.cmd = cmd
	p.stdIn = stdIn
	p.initSuccess = true
}

func (p *ffProcessor) handle(data []byte) {
	_, _ = p.stdIn.Write(data)
}

func (p *ffProcessor) destroy() {
	if p.stdIn != nil {
		_ = p.stdIn.Close()
	}
}
