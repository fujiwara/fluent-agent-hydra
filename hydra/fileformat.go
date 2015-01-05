package hydra

import (
	"fmt"
	"strings"
)

type FileFormat int

const (
	None FileFormat = iota
	LTSV
	JSON
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
