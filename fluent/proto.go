//go:generate msgp

package fluent

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

//msgp:tuple FluentRecord
type FluentRecord struct {
	Tag       string
	Timestamp int64
	Data      map[string]interface{}
}

func (r FluentRecord) Pack() ([]byte, error) {
	if data, dumperr := r.MarshalMsg(nil); dumperr != nil {
		fmt.Println("Can't convert to msgpack:", r, dumperr)
		return nil, dumperr
	} else {
		return data, nil
	}
}

func (r *FluentRecord) String() string {
	_d, _ := json.Marshal(r.Data)
	return strings.Join(
		[]string{
			time.Unix(r.Timestamp, 0).Format(time.RFC3339),
			r.Tag,
			string(_d),
		},
		"\t",
	)
}

func (r *FluentRecord) GetAllData() map[string]interface{} {
	return r.Data
}

func (r *FluentRecord) GetData(key string) (interface{}, bool) {
	value, ok := r.Data[key]
	return value, ok
}

//msgp:tuple TinyFluentRecord
type TinyFluentRecord struct {
	Timestamp int64
	Data      map[string]interface{}
}

func (r *TinyFluentRecord) Pack() ([]byte, error) {
	return r.MarshalMsg(nil)
}

func (r *TinyFluentRecord) GetData(key string) (interface{}, bool) {
	value, ok := r.Data[key]
	return value, ok
}

func (r *TinyFluentRecord) GetAllData() map[string]interface{} {
	return r.Data
}

func (r *TinyFluentRecord) String() string {
	_d, _ := json.Marshal(r.Data)
	return strings.Join(
		[]string{
			time.Unix(r.Timestamp, 0).Format(time.RFC3339),
			string(_d),
		},
		"\t",
	)
}

type TinyFluentMessage struct {
	Timestamp int64
	FieldName string
	Message   []byte
}

func (r *TinyFluentMessage) Pack() ([]byte, error) {
	return toMsgpackTinyMessage(r.Timestamp, r.FieldName, r.Message), nil
}

func (r *TinyFluentMessage) GetData(key string) (interface{}, bool) {
	if key == r.FieldName {
		return r.Message, true
	} else {
		return nil, false
	}
}

func (r *TinyFluentMessage) GetAllData() map[string]interface{} {
	return map[string]interface{}{r.FieldName: r.Message}
}

func (r *TinyFluentMessage) String() string {
	_d, _ := json.Marshal(r.GetAllData())
	return strings.Join(
		[]string{
			time.Unix(r.Timestamp, 0).Format(time.RFC3339),
			string(_d),
		},
		"\t",
	)
}
