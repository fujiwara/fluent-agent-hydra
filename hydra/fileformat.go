package hydra

import "fmt"

type FileFormat int

const (
	None FileFormat = iota
	LTSV
)

func (f *FileFormat) UnmarshalText(text []byte) error {
	switch string(text) {
	case "LTSV", "ltsv":
		*f = LTSV
	case "", "NONE", "None", "none":
		*f = None
	default:
		return fmt.Errorf("Invalid Format %s", string(text))
	}
	return nil
}
