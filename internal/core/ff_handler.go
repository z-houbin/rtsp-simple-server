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
	"strings"
)

type FFHandler struct {
	connect map[string]*ffProcessor
	logger  CPCLogger
}

//OnConnect imp WsStatusListener
func (h *FFHandler) OnConnect(uuid string, kind string, dest string, ffmpegArgs string) {
	h.logger.Log(logger.Info, "ff.connect %s %s", uuid, util.TimeUtil{}.GetTimeStr())
	processor := &ffProcessor{
		uuid:   uuid,
		logger: h.logger,
	}
	processor.init(uuid, kind, dest, ffmpegArgs)
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

func (p *ffProcessor) init(uuid string, kind string, dest string, ffmpegArgs string) {
	p.logger.Log(logger.Info, "ff.init %s %s", uuid, util.TimeUtil{}.GetTimeStr())

	var cmdName string
	var cmdArgs []string

	if runtime.GOOS == "windows" {
		cmdName = "cmd"
		cmdArgs = append(cmdArgs, "/c", "ffmpeg")
	} else {
		cmdName = "ffmpeg"
	}

	if ffmpegArgs == "" {
		cmdArgs = append(cmdArgs,
			"-hide_banner",
			"-f", "webm",
			"-analyzeduration", "1000",
			"-i", "-", // 管道输入
			//"-c:v", "h264",
			//"-c:a", "opus",
			"-preset:v", "fast", //编码速度,影响视频质量 ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow, placedo
			"-tune", "zerolatency", //视频类型,表示零延迟
			"-b:v", "800k", //码率比特率,每秒处理的字节数,默认200kb
			"-async", "1",
			"-r", "15", //帧率,视频中每秒图片帧数,默认25,低于输入可能会丢帧
			"-use_wallclock_as_timestamps", "1", //用系统时间计时当成时间轴
			//"-bufsize", "10240", //设置码率控制缓冲区大小
			"-g", "12", //图片组大小
		)
	} else {
		cmdArgs = append(cmdArgs, strings.Split(ffmpegArgs, " ")...)
	}

	if kind == "aliyun" {
		cmdArgs = append(cmdArgs, "-vf", "crop=9/16*in_h:in_h,transpose=2")
	}

	cmdArgs = append(cmdArgs,
		"-rtsp_transport", "tcp", //rtsp传输协议
		"-f", "rtsp", //文件格式
		dest)

	p.logger.Log(logger.Info, "ff.processor.command %s %s", cmdArgs)

	cmd := exec.Command(cmdName, cmdArgs...)
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
