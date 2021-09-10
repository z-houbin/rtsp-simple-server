package core

import (
	"fmt"
	"github.com/aler9/rtsp-simple-server/internal/logger"
	"github.com/aler9/rtsp-simple-server/internal/util"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
)

type FFHandler struct {
	connect map[string]*ffProcessor
	logger  CPCLogger
}

//OnConnect imp WsStatusListener
func (h *FFHandler) OnConnect(uuid string, rtspHost string) {
	h.logger.Log(logger.Info, "ff.connect %s %s", uuid, util.TimeUtil{}.GetTimeStr())
	processor := &ffProcessor{
		uuid:   uuid,
		logger: h.logger,
	}
	processor.init(uuid, rtspHost)
	h.connect[uuid] = processor
}

//OnMessage imp WsStatusListener
func (h *FFHandler) OnMessage(uuid string, data []byte) {
	processor := h.connect[uuid]
	if processor == nil {
		fmt.Printf("connect lost %s", uuid)
		return
	}
	if processor.initSuccess {
		processor.handle(data)
	}
}

//OnDisconnect imp WsStatusListener
func (h *FFHandler) OnDisconnect(uuid string) {
	h.logger.Log(logger.Info, "ff disconnect %s %s", uuid, util.TimeUtil{}.GetTimeStr())
	h.connect[uuid].destroy()
	delete(h.connect, uuid)
}

type ffProcessor struct {
	uuid        string
	cmd         *exec.Cmd
	stdIn       io.WriteCloser
	initSuccess bool
	logger      CPCLogger
}

func (p *ffProcessor) init(uuid string, rtspHost string) {
	p.logger.Log(logger.Info, "ff.init %s %s", uuid, util.TimeUtil{}.GetTimeStr())

	sysType := runtime.GOOS
	var rtspURL = "rtsp://" + rtspHost + "/" + uuid
	var cmd *exec.Cmd
	if sysType == "windows" {
		// windows
		// 全局参数 - 输入文件参数 - 输入文件 - 输出文件参数 - 输出文件
		cmd = exec.Command("cmd", "/c",
			"ffmpeg",
			"-f", "webm",
			"-analyzeduration", "1000",
			"-i", "-", // 管道输入
			//"-c:v", "h264",
			//"-c:a", "opus",
			"-preset:v", "ultrafast", //编码速度,影响视频质量 ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow, placedo
			"-tune", "zerolatency", //视频类型,表示零延迟
			"-b:v", "800k", //码率比特率,每秒处理的字节数,默认200kb
			"-async", "1",
			"-r", "15", //帧率,视频中每秒图片帧数,默认25,低于输入可能会丢帧
			"-use_wallclock_as_timestamps", "1", //用系统时间计时当成时间轴
			//"-bufsize", "10240", //设置码率控制缓冲区大小
			"-g", "12", //图片组大小
			"-rtsp_transport", "tcp", //rtsp传输协议
			"-f", "rtsp", //文件格式
			rtspURL)
	} else {
		cmd = exec.Command("ffmpeg",
			"-f", "webm",
			"-analyzeduration", "1000",
			"-i", "-", // 管道输入
			//"-c:v", "h264",
			//"-c:a", "opus",
			"-preset:v", "ultrafast", //编码速度,影响视频质量 ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow, placedo
			"-tune", "zerolatency", //视频类型,表示零延迟
			"-b:v", "800k", //码率比特率,每秒处理的字节数,默认200kb
			"-async", "1",
			"-r", "15", //帧率,视频中每秒图片帧数,默认25,低于输入可能会丢帧
			"-use_wallclock_as_timestamps", "1", //用系统时间计时当成时间轴
			//"-bufsize", "10240", //设置码率控制缓冲区大小
			"-g", "12", //图片组大小
			"-rtsp_transport", "tcp", //rtsp传输协议
			"-f", "rtsp", //文件格式
			rtspURL)
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
		p.logger.Log(logger.Info, "ff.processor.finish %s %s", uuid, util.TimeUtil{}.GetTimeStr())
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
	_ = p.cmd.Process.Kill()
}
