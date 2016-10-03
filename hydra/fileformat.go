package hydra

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fujiwara/fluent-agent-hydra/fluent"
)

type FileFormat int

const (
	FormatNone FileFormat = iota
	FormatLTSV
	FormatJSON
	FormatRegexp
)

const (
	ConvertTypeString = iota
	ConvertTypeInt
	ConvertTypeFloat
	ConvertTypeBool
)

type ConvertType int

type Converter interface {
	Convert(string) (interface{}, error)
}

type TimeFormat string

type BoolConverter int
type IntConverter int
type FloatConverter int
type TimeConverter TimeFormat

type Regexp struct {
	*regexp.Regexp
}

var (
	TimeFormatApache = TimeFormat("02/Jan/2006:15:04:05 -0700")
	TimeFormatNginx  = TimeFormat("02/Jan/2006:15:04:05 -0700")
	TimeFormatSyslog = TimeFormat("Jan 02 15:04:05")
	TimeFormatUnix   = TimeFormat("unix")
	TimeEpoch        = time.Unix(0, 0)
)

var (
	RegexpApache      = regexp.MustCompile(`^(?P<host>[^ ]*) [^ ]* (?P<user>[^ ]*) \[(?P<time>[^\]]*)\] "(?P<method>\S+)(?: +(?P<path>[^\"]*?)(?: +\S*)?)?" (?P<code>[^ ]*) (?P<size>[^ ]*)(?: "(?P<referer>[^\"]*)" "(?P<agent>[^\"]*)")?$`)
	RegexpApacheError = regexp.MustCompile(`^\[[^ ]* (?P<time>[^\]]*)\] \[(?P<level>[^\]]*)\](?: \[pid (?P<pid>[^\]]*)\])?( \[client (?P<client>[^\]]*)\])? (?P<message>.*)$`)
	RegexpNginx       = regexp.MustCompile(`^(?P<remote>[^ ]*) (?P<host>[^ ]*) (?P<user>[^ ]*) \[(?P<time>[^\]]*)\] "(?P<method>\S+)(?: +(?P<path>[^\"]*?)(?: +\S*)?)?" (?P<code>[^ ]*) (?P<size>[^ ]*)(?: "(?P<referer>[^\"]*)" "(?P<agent>[^\"]*)")?$`)
	RegexpSyslog      = regexp.MustCompile(`(?P<time>[^ ]*\s*[^ ]* [^ ]*) (?P<host>[^ ]*) (?P<ident>[a-zA-Z0-9_\/\.\-]*)(?:\[(?P<pid>[0-9]+)\])?(?:[^\:]*\:)? *(?P<message>.*)$`)
)

var (
	convertBool  BoolConverter
	convertInt   IntConverter
	convertFloat FloatConverter
)

func (c BoolConverter) Convert(v string) (interface{}, error) {
	return strconv.ParseBool(v)
}

func (c IntConverter) Convert(v string) (interface{}, error) {
	return strconv.ParseInt(v, 10, 64)
}

func (c FloatConverter) Convert(v string) (interface{}, error) {
	return strconv.ParseFloat(v, 64)
}

func (c TimeConverter) Convert(v string) (time.Time, error) {
	if TimeFormat(c) == TimeFormatUnix {
		_v := strings.SplitN(v, ".", 2)
		var sec, nsec int64
		var err error
		sec, err = strconv.ParseInt(_v[0], 10, 64)
		if err != nil {
			return TimeEpoch, err
		}
		if len(_v) == 2 {
			nsec, err = strconv.ParseInt(_v[1], 10, 64)
			if err != nil {
				nsec = 0
			}
		}
		return time.Unix(sec, nsec), nil
	} else {
		return time.Parse(string(c), v)
	}
}

type ConvertMap struct {
	TypeMap      map[string]ConvertType
	ConverterMap map[string]Converter
}

type RecordModifier struct {
	convertMap    ConvertMap
	timeParse     bool
	timeKey       string
	timeConverter TimeConverter
}

func (m *RecordModifier) Modify(r *fluent.TinyFluentRecord) {
	if m.convertMap.ConverterMap != nil {
		m.convertMap.ConvertTypes(r.Data)
	}
	if !m.timeParse {
		return
	}
	if _t, ok := r.Data[m.timeKey]; ok {
		if t, ok := _t.(string); ok {
			if ts, err := m.timeConverter.Convert(t); err == nil {
				r.Timestamp = ts
			}
		}
	}
}

func (f *FileFormat) UnmarshalText(text []byte) error {
	switch strings.ToLower(string(text)) {
	case "ltsv":
		*f = FormatLTSV
	case "json":
		*f = FormatJSON
	case "regexp":
		*f = FormatRegexp
	case "", "none":
		*f = FormatNone
	default:
		return fmt.Errorf("Invalid Format %s", string(text))
	}
	return nil
}

func (r *Regexp) UnmarshalText(text []byte) error {
	var err error
	s := string(text)
	switch strings.ToLower(s) {
	case "apache":
		r.Regexp = RegexpApache
	case "apache_error":
		r.Regexp = RegexpApacheError
	case "nginx":
		r.Regexp = RegexpNginx
	case "syslog":
		r.Regexp = RegexpSyslog
	default:
		r.Regexp, err = regexp.Compile(s)
	}
	return err
}

func (c *ConvertMap) UnmarshalText(text []byte) error {
	*c = NewConvertMap(string(text))
	return nil
}

func NewConvertMap(config string) ConvertMap {
	var m ConvertMap
	m.TypeMap = make(map[string]ConvertType)
	m.ConverterMap = make(map[string]Converter)
	for _, subdef := range strings.Split(config, ",") {
		def := strings.SplitN(subdef, ":", 2)
		if len(def) < 2 {
			continue
		}
		key := def[0]
		switch def[1] {
		case "bool":
			m.TypeMap[key] = ConvertTypeBool
			m.ConverterMap[key] = convertBool
		case "integer":
			m.TypeMap[key] = ConvertTypeInt
			m.ConverterMap[key] = convertInt
		case "float":
			m.TypeMap[key] = ConvertTypeFloat
			m.ConverterMap[key] = convertFloat
		default:
		}
	}
	return m
}

func (c ConvertMap) ConvertTypes(data map[string]interface{}) {
	for key, converter := range c.ConverterMap {
		if _value, ok := data[key]; ok {
			switch value := _value.(type) {
			default:
				continue
			case float64:
				if c.TypeMap[key] == ConvertTypeInt {
					data[key] = int64(value)
				}
			case float32:
				if c.TypeMap[key] == ConvertTypeInt {
					data[key] = int64(value)
				}
			case int:
				if c.TypeMap[key] == ConvertTypeInt {
					data[key] = int64(value)
				}
			case int32:
				if c.TypeMap[key] == ConvertTypeInt {
					data[key] = int64(value)
				}
			case string:
				if v, err := converter.Convert(value); err == nil {
					data[key] = v
				}
			}
		}
	}
}

func (t *TimeFormat) UnmarshalText(text []byte) error {
	switch strings.ToLower(string(text)) {
	case "apache":
		*t = TimeFormatApache
	case "nginx":
		*t = TimeFormatApache
	case "syslog":
		*t = TimeFormatSyslog
	default:
		*t = TimeFormat(text)
	}
	return nil
}
