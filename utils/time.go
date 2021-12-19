package libutils

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

// 编码时间
func MarshalTime(t time.Time) (tbytes []byte) {
	tbytes, _ = t.UTC().MarshalBinary()
	tbytes = tbytes[1:13]
	return
}

// 解码时间
func UnmarshalTime(b interface{}) (t time.Time, err error) {
	var unix int64
	switch v := b.(type) {
	case int64:
		unix = v
	case string:
		if len(v) != 8 {
			err = errors.New("Unable to resolve time")
		}
		buf := bytes.NewBuffer([]byte(v))
		if err = binary.Read(buf, binary.BigEndian, &unix); err != nil {
			err = errors.New("Unable to resolve time")
			return
		}
	case []byte:
		// 15 位
		if len(v) == 15 {
			err = t.UnmarshalBinary(v)
			if err == nil {
				t = t.UTC()
			}
			return
		}

		// 12 未数
		if len(v) == 12 {
			v = bytes.Join([][]byte{[]byte{1}, v, []byte{255}, []byte{255}}, []byte{})
			err = t.UnmarshalBinary(v)
			return
		}

		// 8 位
		if len(v) == 8 {
			buf := bytes.NewBuffer(v)
			if err = binary.Read(buf, binary.BigEndian, &unix); err != nil {
				err = errors.New("Unable to resolve time")
				return
			}
		}
		err = errors.New("Unable to resolve time")
	default:
		return UnmarshalTime(fmt.Sprintf("%v", b))
	}
	t = time.Unix(unix, 0)
	return
}
