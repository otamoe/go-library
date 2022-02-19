package libffmpeg

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	goLog "github.com/ipfs/go-log/v2"
	libcommand "github.com/otamoe/go-library/command"
	libutils "github.com/otamoe/go-library/utils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type (
	FFmpegCommand struct {
		transcodingLogger  *zap.Logger
		transcodingCommand *libcommand.Name

		screenshotLogger  *zap.Logger
		screenshotCommand *libcommand.Name

		keyframeLogger  *zap.Logger
		keyframeCommand *libcommand.Name
	}

	FFmpegEvent interface {
		Start(dir string) (err error)
		Progress(loaded time.Duration, total time.Duration, speed float32) (err error)
		Complete(out []byte, err error) (rerr error)
	}
	FFmpegStartFunc    func(dir string) (err error)
	FFmpegProgressFunc func(loaded time.Duration, total time.Duration, speed float32) (err error)
	FFmpegEndFunc      func(out []byte) (err error)

	Type uint8
)

const TypeTranscoding = Type(1)
const TypeScreenshot = Type(2)
const TypeKeyframe = Type(3)

var HttpAccept = "*/*"
var HttpAcceptLanguage = "en,en-GB;q=0.9,en-US;q=0.8"
var HttpUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/69.0.3497.100 Safari/537.36"

func (ffmpegCommand *FFmpegCommand) Get(ctx context.Context, typ Type, args []string, event FFmpegEvent) (out []byte, err error) {
	if args, err = ffmpegCommand.preprocessingArgs(args); err != nil {
		return
	}
	stdout := bytes.NewBuffer([]byte{})
	stderr := &Buffer{
		Buffer: bytes.NewBuffer([]byte{}),
		event:  event,
	}
	switch typ {
	case TypeKeyframe:
		stderr.logger = ffmpegCommand.keyframeLogger
	case TypeScreenshot:
		stderr.logger = ffmpegCommand.screenshotLogger
	default:
		stderr.logger = ffmpegCommand.transcodingLogger
	}

	// 临时目录
	dir := path.Join(os.TempDir(), "ffmpeg-"+string(libutils.RandByte(64, libutils.RandAlphaLowerNumber)))
	if err = os.MkdirAll(dir, 0755); err != nil {
		return
	}

	// 删除临时目录
	defer os.RemoveAll(dir)

	// 使用 ffmpeg 解析数据
	var cmd *libcommand.Run
	switch typ {
	case TypeKeyframe:
		cmd = ffmpegCommand.keyframeCommand.Run(ctx, dir, nil, stdout, stderr, args...)
	case TypeScreenshot:
		cmd = ffmpegCommand.screenshotCommand.Run(ctx, dir, nil, stdout, stderr, args...)
	default:
		cmd = ffmpegCommand.transcodingCommand.Run(ctx, dir, nil, stdout, stderr, args...)
	}

	// 已开始运行
	if err = event.Start(dir); err != nil {
		return
	}

	defer func() {
		if event != nil {
			err = event.Complete(out, err)
		}
	}()

	select {
	case <-cmd.Wait():
		if err = cmd.Err(); err != nil {
			return
		}

		if out, err = ioutil.ReadAll(stdout); err != nil {
			return
		}
		return
	}
}

func NewFFmpegCommand(command *libcommand.Command) *FFmpegCommand {
	ffmpegCommand := &FFmpegCommand{
		transcodingLogger:  goLog.Logger("ffmpeg.transcoding").Desugar(),
		transcodingCommand: command.Command("ffmpeg-transcoding", (runtime.NumCPU()/2)+1, time.Second*3600),

		screenshotLogger:  goLog.Logger("ffmpeg.screenshot").Desugar(),
		screenshotCommand: command.Command("ffmpeg-screenshot", runtime.NumCPU()*30, time.Second*30),

		keyframeLogger:  goLog.Logger("ffmpeg.keyframe").Desugar(),
		keyframeCommand: command.Command("ffmpeg-keyframe", runtime.NumCPU()*5, time.Second*600),
	}

	return ffmpegCommand
}

func New() fx.Option {
	return fx.Options(
		fx.Provide(NewFFmpegCommand),
	)
}

func (ffmpegCommand *FFmpegCommand) preprocessingArgs(args []string) (rargs []string, err error) {
	if len(args) == 0 {
		err = errors.New("Input field is empty")
		return
	}

	// 解析输入
	var argInput string
	for index, arg := range args {
		if arg == "-i" {
			if len(args) > (index + 1) {
				argInput = args[index+1]
			}
			break
		}
	}

	var u *url.URL
	if u, err = url.Parse(strings.Trim(argInput, "'")); err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") {
		err = errors.New("Input address is not a URL")
		return
	}

	// headers
	headers := []string{
		"Accept: " + HttpAccept,
		"Accept-Language: " + HttpAcceptLanguage,
		"DNT: 1",
	}

	if u.Scheme != "https" {
		headers = append(headers, "Upgrade-Insecure-Requests: 1")
	}

	var authType string
	if u.User == nil {
		authType = "0"
	} else {
		authType = "1"
	}

	rargs = append([]string{"-headers", strings.Join(headers, "\r\n"), "-user_agent", HttpUserAgent, "-auth_type", authType}, args...)

	return
}
