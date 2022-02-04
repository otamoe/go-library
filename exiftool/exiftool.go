package libexiftool

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"time"

	vasilemimetype "github.com/gabriel-vasile/mimetype"
	libcommand "github.com/otamoe/go-library/command"
	"github.com/rakyll/magicmime"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type (
	ExiftoolCommand struct {
		logger  *zap.Logger
		command *libcommand.Name
	}
)

func (exiftoolCommand *ExiftoolCommand) Get(ctx context.Context, header []byte) (res map[string]interface{}, err error) {
	// 使用 magic 检查 mime 类型
	var mimeDecoder *magicmime.Decoder
	if mimeDecoder, err = magicmime.NewDecoder(magicmime.MAGIC_MIME_TYPE | magicmime.MAGIC_SYMLINK | magicmime.MAGIC_ERROR); err != nil {
		exiftoolCommand.logger.Warn("magic", zap.Error(err))
		return
	}
	defer mimeDecoder.Close()
	var magicMimeType string
	if magicMimeType, err = mimeDecoder.TypeByBuffer(header); err != nil {
		exiftoolCommand.logger.Warn("magic", zap.Error(err))
		return
	}
	magicMimeType = strings.TrimSpace(strings.Split(magicMimeType, ";")[0])

	// 使用 go 检查 mime类型
	maxend := len(header)
	if maxend > 1024*8 {
		maxend = 1024 * 8
	}
	goMimeType := vasilemimetype.Detect(header[0:maxend]).String()
	goMimeType = strings.TrimSpace(strings.Split(goMimeType, ";")[0])

	if magicMimeType == "" && goMimeType == "" {
		err = errors.New("Mime type unknown")
		return
	}

	args := []string{
		"-a",
		"-n",
		"-json",
		"-",
	}

	stdout := bytes.NewBuffer([]byte{})
	stderr := bytes.NewBuffer([]byte{})

	// 使用 exiftool 检查数据
	cmd := exiftoolCommand.command.Run(ctx, os.TempDir(), bytes.NewBuffer(header), stdout, stderr, args...)
	select {
	case <-cmd.Wait():
		if err = cmd.Err(); err != nil {
			return
		}

		var stdoutb []byte
		if stdoutb, err = ioutil.ReadAll(stdout); err != nil {
			return
		}

		ress := []map[string]interface{}{}
		if err = json.Unmarshal(stdoutb, &ress); err != nil || len(ress) == 0 {
			var stderrb []byte
			if stderrb, _ = ioutil.ReadAll(stderr); err != nil && len(stderrb) != 0 {
				err = errors.New(string(stderrb))
			} else {
				err = fmt.Errorf("exiftool: unjson %s", err)
			}
			return
		}
		res = ress[0]
		res["MagicMimeType"] = magicMimeType
		res["GoMimeType"] = goMimeType
		return
	}
}

func NewExiftoolCommand(logger *zap.Logger, command *libcommand.Command) *ExiftoolCommand {
	exiftoolCommand := &ExiftoolCommand{
		logger:  logger.Named("exiftool"),
		command: command.Command("exiftool", runtime.NumCPU()*30, time.Second*10),
	}
	return exiftoolCommand
}

func New() fx.Option {
	return fx.Options(
		fx.Provide(NewExiftoolCommand),
	)
}
