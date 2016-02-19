package ltsv

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"
)

var (
	Replacer = strings.NewReplacer(
		"\t", "\\t",
		"\n", "\\n",
		"\r", "\\r",
	)
)

func replaceBytes(b []byte) []byte {
	b = bytes.Replace(b, []byte{'\n'}, []byte{'\\', 'n'}, -1)
	b = bytes.Replace(b, []byte{'\r'}, []byte{'\\', 'r'}, -1)
	b = bytes.Replace(b, []byte{'\t'}, []byte{'\\', 't'}, -1)
	return b
}

type Encoder struct {
	writer io.Writer
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{writer: w}
}

func (w *Encoder) Encode(data interface{}) error {
	first := true
	switch record := data.(type) {
	default:
		return fmt.Errorf("unsupported type")
	case map[string]interface{}:
		keys := make([]string, 0, len(record))
		for key := range record {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if !first {
				_, err := fmt.Fprint(w.writer, "\t")
				if err != nil {
					return err
				}
			} else {
				first = false
			}
			_, err := fmt.Fprint(w.writer, key, ":")
			if err != nil {
				return err
			}
			switch v := record[key].(type) {
			case string:
				_, err = fmt.Fprint(w.writer, Replacer.Replace(v))
			case []byte:
				_, err = fmt.Fprint(w.writer, Replacer.Replace(string(v)))
			default:
				_, err = fmt.Fprint(w.writer, v)
			}
			if err != nil {
				return err
			}
		}
		_, err := fmt.Fprint(w.writer, "\n")
		if err != nil {
			return err
		}
	}
	return nil
}
