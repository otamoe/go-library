package libffprobe

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	goLog "github.com/ipfs/go-log/v2"
	libcommand "github.com/otamoe/go-library/command"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var HttpAccept = "*/*"
var HttpAcceptLanguage = "en,en-GB;q=0.9,en-US;q=0.8"
var HttpUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/69.0.3497.100 Safari/537.36"

var ErrInvalidData = errors.New("Invalid data found when processing input")

type (
	FFprobe struct {
		Streams []FFprobeStream `json:"streams"`
		Format  FFprobeFormat   `json:"format"`
	}

	FFprobeStream struct {
		Index          int    `json:"index"`
		CodecType      string `json:"codec_type"`
		CodecName      string `json:"codec_name"`
		CodecLongName  string `json:"codec_long_name"`
		CodecTag       string `json:"codec_tag"`
		CodecTagString string `json:"codec_tag_string"`
		CodecTimeBase  string `json:"codec_time_base"`

		Width  *int `json:"width"`
		Height *int `json:"height"`

		RFrameRate         *string                  `json:"r_frame_rate"`
		AvgFrameRate       *string                  `json:"avg_frame_rate"`
		TimeBase           *string                  `json:"time_base"`
		StartPts           *int                     `json:"start_pts"`
		StartTime          *float64                 `json:"start_time,string"`
		DurationTs         *int64                   `json:"duration_ts"`
		Duration           *float64                 `json:"duration,string"`
		BitRate            *int                     `json:"bit_rate,string"`
		MaxBitRate         *int                     `json:"max_bit_rate,string"`
		BitsPerRawSample   *int                     `json:"bits_per_raw_sample,string"`
		NbFrames           *int                     `json:"nb_frames,string"`
		Disposition        FFprobeStreamDisposition `json:"disposition"`
		Tags               map[string]string        `json:"tags"`
		SideDataList       []FFprobeSideDataList    `json:"side_data_list"`
		Profile            *string                  `json:"profile"`
		HasBFrames         *int                     `json:"has_b_frames"`
		SampleAspectRatio  *string                  `json:"sample_aspect_ratio"`
		DisplayAspectRatio *string                  `json:"display_aspect_ratio"`
		PixFmt             *string                  `json:"pix_fmt"`
		Level              *int                     `json:"level"`
		ColorRange         *string                  `json:"color_range"`
		ColorSpace         *string                  `json:"color_space"`
		ColorTransfer      *string                  `json:"color_transfer"`
		ColorPrimaries     *string                  `json:"color_primaries"`
		ChromaLocation     *string                  `json:"chroma_location"`
		IsAvc              *bool                    `json:"is_avc,string"`
		SampleFmt          *string                  `json:"sample_fmt"`
		SampleRate         *int                     `json:"sample_rate,string"`
		Channels           *int                     `json:"channels"`
		ChannelLayout      *string                  `json:"channel_layout"`
		BitsPerSample      *int                     `json:"bits_per_sample"`
	}
	FFprobeSideDataList struct {
		SideDataType string `json:"side_data_type"`
		Type         string `json:"type"`
		Projection   string `json:"projection"`
		Rotation     int    `json:"rotation"`
		Inverted     int    `json:"inverted"`
		Yaw          int    `json:"yaw"`
		Pitch        int    `json:"pitch"`
		Roll         int    `json:"roll"`
	}

	FFprobeStreamDisposition struct {
		Default         int `json:"default"`
		Dub             int `json:"dub"`
		Original        int `json:"original"`
		Comment         int `json:"comment"`
		Lyrics          int `json:"lyrics"`
		Karaoke         int `json:"karaoke"`
		Forced          int `json:"forced"`
		HearingImpaired int `json:"hearing_impaired"`
		VisualImpaired  int `json:"visual_impaired"`
		CleanEffects    int `json:"clean_effects"`
		AttachedPic     int `json:"attached_pic"`
		TimedThumbnails int `json:"timed_thumbnails"`
	}

	FFprobeFormat struct {
		NbStreams      int               `json:"nb_streams"`
		NbPrograms     int               `json:"nb_programs"`
		FormatName     string            `json:"format_name"`
		FormatLongName string            `json:"format_long_name"`
		StartTime      float64           `json:"start_time,string"`
		Duration       float64           `json:"duration,string"`
		BitRate        int               `json:"bit_rate,string"`
		ProbeScore     int               `json:"probe_score"`
		Tags           map[string]string `json:"tags"`
	}

	FFprobeCommand struct {
		logger  *zap.Logger
		command *libcommand.Name
	}
)

func (ffprobeCommand *FFprobeCommand) Get(ctx context.Context, u *url.URL) (res *FFprobe, err error) {
	if u == nil {
		err = errors.New("Input field is empty")
		return
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		err = errors.New("Input field is not URL")
		return
	}
	if u.Host == "" {
		err = errors.New("Input field is not URL")
		return
	}

	var authType string
	if u.User == nil {
		authType = "0"
	} else {
		authType = "1"
	}

	headers := []string{
		"Accept: " + HttpAccept,
		"Accept-Language: " + HttpAcceptLanguage,
		"DNT: 1",
	}
	if u.Scheme != "https" {
		headers = append(headers, "Upgrade-Insecure-Requests: 1")
	}

	args := []string{
		"-i", u.String(),
		"-auth_type", authType,
		"-headers", strings.Join(headers, "\r\n"),
		"-user_agent", HttpUserAgent,
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		"-show_error",
	}

	defer func() {
		if err != nil && strings.Index(err.Error(), ErrInvalidData.Error()) != -1 {
			err = ErrInvalidData
		}
	}()

	stdout := bytes.NewBuffer([]byte{})
	stderr := bytes.NewBuffer([]byte{})

	// 使用 ffprobe 解析数据
	cmd := ffprobeCommand.command.Run(ctx, os.TempDir(), nil, stdout, stderr, args...)
	select {
	case <-cmd.Wait():
		if err = cmd.Err(); err != nil {
			return
		}

		var stdoutb []byte
		if stdoutb, err = ioutil.ReadAll(stdout); err != nil {
			return
		}

		res = &FFprobe{}
		if err = json.Unmarshal(stdoutb, res); err != nil {
			ffprobeCommand.logger.Error("unjson", zap.String("stdout", string(stdoutb)))
			var stderrb []byte
			if stderrb, _ = ioutil.ReadAll(stderr); err != nil && len(stderrb) != 0 {
				err = errors.New(string(stderrb))
				ffprobeCommand.logger.Error("unjson", zap.String("stderr", string(stderrb)))
			} else {
				err = fmt.Errorf("ffprobe: unjson %s", err)
			}
			res = nil
			return
		}
		return
	}
}

func NewFFprobeCommand(command *libcommand.Command) *FFprobeCommand {
	ffprobeCommand := &FFprobeCommand{
		logger:  goLog.Logger("ffprobe").Desugar(),
		command: command.Command("ffprobe", runtime.NumCPU()*30, time.Second*60),
	}
	return ffprobeCommand
}

func New() fx.Option {
	return fx.Options(
		fx.Provide(NewFFprobeCommand),
	)
}
