package libffmpeg

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

type (
	Buffer struct {
		*bytes.Buffer

		logger *zap.Logger

		event FFmpegEvent

		total  time.Duration
		loaded time.Duration
		speed  float32
		time   time.Time
	}
)

func (b *Buffer) Write(p []byte) (n int, err error) {
	if n, err = b.Buffer.Write(p); err != nil {
		return
	}
	if b.event == nil {
		return
	}

	// 写入总共
	if b.total == 0 && b.Len() < 1024*8 {
		totalStr := b.Bytes()
		if s := bytes.LastIndex(totalStr, []byte("Duration: ")); s != -1 {
			totalStr = totalStr[s+10:]
			if n := bytes.Index(totalStr, []byte(",")); n != -1 {
				totalStr = totalStr[0:n]
				if d, e := ParseDuration(string(totalStr)); e == nil {
					b.total = d
				}
			}
		}
	}

	// 2 秒更新一次
	now := time.Now()
	if b.time.Add(time.Second * 2).After(now) {
		return
	}
	b.time = now

	// 已写入大小和 和百分比
	loadedStr := p
	if s := bytes.LastIndex(loadedStr, []byte(" time=")); s != -1 {
		loadedStr = loadedStr[s+6:]
		if n := bytes.Index(loadedStr, []byte(" bitrate=")); n != -1 {
			loadedStr = loadedStr[0:n]
			if d, e := ParseDuration(string(loadedStr)); e == nil {
				if b.total != 0 && d > b.total {
					d = b.total
				}
				b.loaded = d
			}
		}
	}

	// 写入速率
	speedStr := p
	if s := bytes.LastIndex(speedStr, []byte(" speed=")); s != -1 {
		speedStr = speedStr[s+7:]
		if n := bytes.Index(speedStr, []byte("x")); n != -1 {
			speedStr = speedStr[0:n]
			if speed, e := strconv.ParseFloat(string(speedStr), 10); e == nil && speed > 0 {
				b.speed = float32(speed)
			}
		}
	}

	err = b.event.Progress(b.total, b.loaded, b.speed)
	if err != nil {
		b.logger.Error("progress", zap.Error(err))
	}
	return
}

func ParseDuration(val string) (out time.Duration, err error) {
	s := strings.Split(val, ":")
	var d float64
	var x float64 = 1
	for i := len(s) - 1; i >= 0; i-- {
		v, e := strconv.ParseFloat(string(s[i]), 64)
		if e != nil {
			err = errors.New("Invalid duration")
			return
		}
		d += v * x
		x = x * 60
	}
	out = time.Duration(d * float64(time.Second))
	return
}
