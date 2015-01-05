package hydra

import (
	"fmt"
	"strconv"
	"strings"
)

type FileFormat int
type Converter int
type ConvertMap map[string]Converter

const (
	None FileFormat = iota
	LTSV
	JSON
)

const (
	ConvertBool Converter = iota + 1
	ConvertInt
	ConvertFloat
)

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
	convertMap := make(ConvertMap)
	for _, subdef := range strings.Split(config, ",") {
		def := strings.SplitN(subdef, ":", 2)
		if len(def) < 2 {
			continue
		}
		key := def[0]
		switch def[1] {
		case "bool":
			convertMap[key] = ConvertBool
		case "integer":
			convertMap[key] = ConvertInt
		case "float":
			convertMap[key] = ConvertFloat
		default:
		}
	}
	return convertMap
}

func ConvertTypes(data map[string]interface{}, convertMap ConvertMap) {
	for key, _value := range data {
		switch value := _value.(type) {
		default:
			continue
		case string:
			switch convertMap[key] {
			case ConvertBool:
				v, err := strconv.ParseBool(value)
				if err == nil {
					data[key] = v
				}
			case ConvertInt:
				v, err := strconv.Atoi(value)
				if err == nil {
					data[key] = v
				}
			case ConvertFloat:
				v, err := strconv.ParseFloat(value, 64)
				if err == nil {
					data[key] = v
				}
			}
		}
	}
}
