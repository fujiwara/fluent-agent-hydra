package hydra

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/fujiwara/fluent-agent-hydra/fluent"
)

type FileFormat int

const (
	None FileFormat = iota
	LTSV
	JSON
)

type Converter interface {
	Convert(string) (interface{}, error)
}

type BoolConverter int
type IntConverter int
type FloatConverter int
type TimeConverter string

var (
	convertBool  BoolConverter
	convertInt   IntConverter
	convertFloat FloatConverter
)

func (c BoolConverter) Convert(v string) (interface{}, error) {
	return strconv.ParseBool(v)
}

func (c IntConverter) Convert(v string) (interface{}, error) {
	return strconv.Atoi(v)
}

func (c FloatConverter) Convert(v string) (interface{}, error) {
	return strconv.ParseFloat(v, 64)
}

func (c TimeConverter) Convert(v string) (time.Time, error) {
	return time.Parse(string(c), v)
}

type ConvertMap map[string]Converter

type RecordModifier struct {
	convertMap    ConvertMap
	timeParse     bool
	timeKey       string
	timeConverter TimeConverter
}

func (m *RecordModifier) Modify(r *fluent.TinyFluentRecord) {
	if m.convertMap != nil {
		m.convertMap.ConvertTypes(r.Data)
	}
	if !m.timeParse {
		return
	}
	if _t, ok := r.Data[m.timeKey]; ok {
		if t, ok := _t.(string); ok {
			if ts, err := m.timeConverter.Convert(t); err == nil {
				r.Timestamp = ts.Unix()
			}
		}
	}
}

func (f *FileFormat) UnmarshalText(text []byte) error {
	switch strings.ToLower(string(text)) {
	case "ltsv":
		*f = LTSV
	case "json":
		*f = JSON
	case "", "none":
		*f = None
	default:
		return fmt.Errorf("Invalid Format %s", string(text))
	}
	return nil
}

func (c *ConvertMap) UnmarshalText(text []byte) error {
	*c = NewConvertMap(string(text))
	return nil
}

func NewConvertMap(config string) ConvertMap {
	m := make(ConvertMap)
	for _, subdef := range strings.Split(config, ",") {
		def := strings.SplitN(subdef, ":", 2)
		if len(def) < 2 {
			continue
		}
		key := def[0]
		switch def[1] {
		case "bool":
			m[key] = convertBool
		case "integer":
			m[key] = convertInt
		case "float":
			m[key] = convertFloat
		default:
		}
	}
	return m
}

func (c ConvertMap) ConvertTypes(data map[string]interface{}) {
	for key, converter := range c {
		if _value, ok := data[key]; ok {
			switch value := _value.(type) {
			default:
				continue
			case string:
				v, err := converter.Convert(value)
				if err == nil {
					data[key] = v
				} else {
					log.Println(err)
				}
			}
		}
	}
}
