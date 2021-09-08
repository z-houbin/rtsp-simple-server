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
func (h *FFHandler) OnConnect(uuid string, rtspHost string) {
	fmt.Printf("ffhandler OnConnect:%s\n", uuid)
	processor := &ffProcessor{
		uuid: uuid,
	}
	processor.init(uuid, rtspHost)
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

func (p *ffProcessor) init(uuid string, rtspHost string) {
	sysType := runtime.GOOS
	var rtspURL = "rtsp://" + rtspHost + "/" + uuid
	var cmd *exec.Cmd
	if sysType == "windows" {
		// windows
		// 全局参数 - 输入文件参数 - 输入文件 - 输出文件参数 - 输出文件
		cmd = exec.Command("cmd", "/c",
			"ffmpeg",
			"-i", "-", // 管道输入
			"-c:v", "libx265",
			"-c:a", "copy", //直接复制不重新编码
			"-preset:v", "medium", //编码速度,影响视频质量
			"-tune", "zerolatency", //视频类型,表示零延迟
			"-b:v", "1500k", //码率比特率,每秒处理的字节数,默认200kb
			"-async", "1",
			"-r", "20", //帧率,视频中每秒图片帧数,默认25,低于输入可能会丢帧
			"-use_wallclock_as_timestamps", "1", //用系统时间计时当成时间轴
			"-bufsize", "10240", //设置码率控制缓冲区大小
			"-g", "12", //图片组大小
			"-rtsp_transport", "tcp", //rtsp传输协议
			"-f", "rtsp", //文件格式
			rtspURL)
	} else {
		cmd = exec.Command("ffmpeg",
			"-i", "-", // 管道输入
			"-c:v", "libx265",
			"-c:a", "copy", //直接复制不重新编码
			"-preset:v", "medium", //编码速度,影响视频质量
			"-tune", "zerolatency", //视频类型,表示零延迟
			"-b:v", "1500k", //码率比特率,每秒处理的字节数,默认200kb
			"-async", "1",
			"-r", "20", //帧率,视频中每秒图片帧数,默认25,低于输入可能会丢帧
			"-use_wallclock_as_timestamps", "1", //用系统时间计时当成时间轴
			"-bufsize", "10240", //设置码率控制缓冲区大小
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
