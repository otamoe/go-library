package libutils

import (
	"net"
	"net/http"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"unicode/utf8"
)

type (
	QueryField struct {
		Name     string
		Operator string
	}
)

var (
	saveFilterRegex    = regexp.MustCompile("[[:cntrl:][:punct:][:space:]\\pC\\pP\\pZ\\pM\\pS]+")
	saveSepRegexWeight = regexp.MustCompile("([[:ascii:]]+)")
	saveSepRegexInc    = regexp.MustCompile("([[:digit:]]+|[[:alpha:]]+|\\pN+|\\pL)")
)

func NameOfFunction(f interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

func IsEmpty(val interface{}) bool {
	return val == nil || val == 0 || val == "" || val == false
}

func IsMobile(req *http.Request) bool {
	ua := req.Header.Get("User-Agent")
	if ua == "" {
		return false
	}
	if strings.Index(ua, "Mobile") != -1 {
		return true
	}
	if strings.Index(ua, "Android") != -1 {
		return true
	}
	if strings.Index(ua, "Silk/") != -1 {
		return true
	}
	if strings.Index(ua, "Kindle") != -1 {
		return true
	}
	if strings.Index(ua, "BlackBerry") != -1 {
		return true
	}
	if strings.Index(ua, "Opera Mini") != -1 {
		return true
	}
	if strings.Index(ua, "Opera Mobi") != -1 {
		return true
	}
	return false
}

func ClientIP(req *http.Request, forwarded bool) string {
	if forwarded {
		clientIP := req.Header.Get("X-Forwarded-For")
		clientIP = strings.TrimSpace(strings.Split(clientIP, ",")[0])
		if clientIP == "" {
			clientIP = strings.TrimSpace(req.Header.Get("X-Real-Ip"))
		}
		if clientIP != "" {
			return clientIP
		}
	}
	if ip, _, err := net.SplitHostPort(strings.TrimSpace(req.RemoteAddr)); err == nil {
		return ip
	}
	return ""
}

func QueryFields(query string) (fields []*QueryField) {
	query = saveFilterRegex.ReplaceAllString(query, " ")

	// 增加
	for _, name := range strings.Fields(saveSepRegexInc.ReplaceAllString(query, " $1 ")) {
		field := &QueryField{
			Name:     name,
			Operator: "+",
		}
		fields = append(fields, field)
	}

	// > 运算
	for _, name := range strings.Fields(saveSepRegexWeight.ReplaceAllString(query, " $1 ")) {
		field := &QueryField{
			Name:     name,
			Operator: ">",
		}
		fields = append(fields, field)
	}
	return
}

func MaskPart(val string) string {
	if net.ParseIP(val) != nil {
		return MaskPartIP(val)
	}

	if strings.Index(val, "@") != -1 {
		return MaskPartEmail(val)
	}
	if len(val) <= 3 {
		return "***"
	}

	if len(val) <= 6 {
		return val[0:1] + "***" + val[len(val)-1:1]
	}

	if len(val) <= 9 {
		return val[0:2] + "***" + val[len(val)-2:2]
	}

	return val[0:3] + "***" + val[len(val)-3:3]
}

func MaskPartIP(ip string) string {
	if ip == "" {
		ip = "***"
	} else if strings.Index(ip, ":") != -1 {
		split := strings.Split(ip, ":")
		ip = strings.Join([]string{split[0], "***", split[len(split)-1]}, ":")
	} else if strings.Index(ip, ".") != -1 {
		split := strings.Split(ip, ".")
		ip = strings.Join([]string{split[0], "*", "*", split[len(split)-1]}, ".")
	} else {
		ip = "***"
	}
	return ip
}

func MaskPartEmail(email string) string {
	if email == "" {
		email = "***"
	} else if strings.Index(email, "@") != -1 {
		split := strings.Split(email, "@")
		domain := split[len(split)-1]
		name := split[0]
		if len(name) <= 3 {
			name = "***"
		} else if len(name) <= 6 {
			name = name[0:1] + "***" + name[len(name)-1:1]
		} else if len(name) <= 9 {
			name = name[0:2] + "***" + name[len(name)-2:2]
		} else {
			name = name[0:3] + "***" + name[len(name)-3:3]
		}
		if len(domain) <= 3 {
			domain = "***"
		} else if len(domain) <= 6 {
			domain = domain[0:1] + "***" + domain[len(domain)-1:1]
		} else if len(domain) <= 9 {
			domain = domain[0:2] + "***" + domain[len(domain)-2:2]
		} else {
			domain = domain[0:3] + "***" + domain[len(domain)-3:3]
		}
		email = strings.Join([]string{name, domain}, "@")
	} else {
		email = "***"
	}
	return email
}

func Utf8SubstrAndFill(value string, i int, n int, fill string) string {
	if value == "" || i >= n {
		return ""
	}
	var end string
	if !utf8.ValidString(value) {
		size := len(value)
		if i >= size {
			return ""
		}
		if size > n {
			end = fill
			n -= len(end)
		} else if n > size {
			n = size
		}
		if i >= n {
			return ""
		}
		return value[i:n] + end
	}

	runes := []rune(value)
	size := len(runes)
	if i >= size {
		return ""
	}
	if size > n {
		end = fill
		n -= utf8.RuneCountInString(end)
	} else if n > size {
		n = size
	}
	if i >= n {
		return ""
	}
	return string(runes[i:n]) + end
}

func Utf8LengthLimitPointer(s *string, n int) *string {
	if s == nil || *s == "" {
		return nil
	}
	v := Utf8LengthLimit(*s, n)
	s = &v
	return s
}

func Utf8LengthLimit(s string, n int) string {
	return Utf8SubstrAndFill(s, 0, n, "...")
}
