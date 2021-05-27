/*
Original code from https://github.com/moriyoshi/ik/
--
Copyright (c) 2014 Ik authors.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package fluent

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/ugorji/go/codec"
)

type FluentRecord struct {
	Tag       string
	Timestamp time.Time
	Data      map[string]interface{}
}

func (r FluentRecord) Pack() ([]byte, error) {
	msg := []interface{}{r.Tag, r.Timestamp, r.Data}
	var b bytes.Buffer
	if err := writeMsgpack(&b, msg); err != nil {
		fmt.Println("Can't convert to msgpack:", msg, err)
		return nil, err
	} else {
		return b.Bytes(), nil
	}
}

func (r *FluentRecord) String() string {
	_d, _ := json.Marshal(r.Data)
	return strings.Join(
		[]string{
			r.Timestamp.Format(time.RFC3339),
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

type FluentRecordType interface {
	Pack() ([]byte, error)
	GetData(string) (interface{}, bool)
	GetAllData() map[string]interface{}
	String() string
}

type TinyFluentRecord struct {
	Timestamp time.Time
	Data      map[string]interface{}
}

func (r *TinyFluentRecord) Pack() ([]byte, error) {
	return toMsgpackRecord(r.Timestamp, r.Data)
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
			r.Timestamp.Format(time.RFC3339),
			string(_d),
		},
		"\t",
	)
}

type TinyFluentMessage struct {
	Timestamp time.Time
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
			r.Timestamp.Format(time.RFC3339),
			string(_d),
		},
		"\t",
	)
}

type FluentRecordSet struct {
	Tag     string
	Records []FluentRecordType
}

func (rs *FluentRecordSet) PackAsPackedForward() ([]byte, error) {
	buffer := make([]byte, 0)
	for _, record := range rs.Records {
		data, err := record.Pack()
		if err != nil {
			return nil, err
		}
		buffer = append(buffer, data...)
	}
	return toMsgpackRecordSet(rs.Tag, buffer), nil
}

func (rs *FluentRecordSet) PackAsForward() ([]byte, error) {
	records := make([]interface{}, len(rs.Records))
	var err error
	for i, record := range rs.Records {
		records[i], err = record.Pack()
		if err != nil {
			return nil, err
		}
	}
	var b bytes.Buffer
	if err := writeMsgpack(&b, []interface{}{rs.Tag, records}); err != nil {
		return nil, err
	} else {
		return b.Bytes(), nil
	}
}

type EventTimeExtension struct{}

func (e *EventTimeExtension) WriteExt(v interface{}) []byte {
	panic("WriteExt() must not be called")
}

func (e *EventTimeExtension) ReadExt(dst interface{}, src []byte) {
	if t, ok := dst.(*time.Time); ok {
		b := bytes.NewReader(src)
		var sec, nanosec uint32
		binary.Read(b, binary.BigEndian, &sec)
		binary.Read(b, binary.BigEndian, &nanosec)
		*t = time.Unix(int64(sec), int64(nanosec))
	}
}

func coerceInPlaceInArray(data []interface{}) []interface{} {
	var newArr = make([]interface{}, len(data))
	for i, av := range data {
		switch v2_ := av.(type) {
		case []byte:
			newArr[i] = string(v2_)
		case []interface{}:
			na := coerceInPlaceInArray(v2_)
			newArr[i] = na
		case map[string]interface{}:
			coerceInPlace(v2_)
			newArr[i] = v2_
		default:
			newArr[i] = av
		}
	}
	return newArr
}

func coerceInPlace(data map[string]interface{}) {
	for k, v := range data {
		switch v_ := v.(type) {
		case []byte:
			data[k] = string(v_) // XXX: byte => rune
		case []interface{}:
			na := coerceInPlaceInArray(v_)
			data[k] = na
		case map[string]interface{}:
			coerceInPlace(v_)
		}
	}
}

func decodeRecordSet(tag []byte, entries []interface{}) (FluentRecordSet, error) {
	records := make([]FluentRecordType, len(entries))
	for i, _entry := range entries {
		entry, ok := _entry.([]interface{})
		if !ok || len(entry) != 2 {
			return FluentRecordSet{}, errors.New("Failed to decode recordSet")
		}
		// timestamp
		var timestamp time.Time
		switch entry[0].(type) {
		case int, uint, int64, uint64, int32, uint32, float32, float64:
			timestamp = time.Unix(toInt64(entry[0]), 0)
		case time.Time:
			timestamp = entry[0].(time.Time)
		default:
			return FluentRecordSet{}, errors.New("Failed to decode timestamp field")
		}
		// data
		data, ok := entry[1].(map[string]interface{})
		if !ok {
			return FluentRecordSet{}, errors.New("Failed to decode data field")
		}
		coerceInPlace(data)
		records[i] = &TinyFluentRecord{
			Timestamp: timestamp,
			Data:      data,
		}
	}
	return FluentRecordSet{
		Tag:     string(tag), // XXX: byte => rune
		Records: records,
	}, nil
}

func DecodeEntries(conn net.Conn) ([]FluentRecordSet, error) {
	dec := codec.NewDecoder(conn, &mh)
	v := []interface{}{nil, nil, nil}
	err := dec.Decode(&v)
	if err != nil {
		return nil, err
	}
	tag, ok := v[0].([]byte)
	if !ok {
		return nil, errors.New("Failed to decode tag field")
	}

	var retval []FluentRecordSet
	switch timestamp_or_entries := v[1].(type) {
	case int, uint, int64, uint64, int32, uint32, float32, float64:
		timestamp := toInt64(timestamp_or_entries)
		data, ok := v[2].(map[string]interface{})
		if !ok {
			return nil, errors.New("Failed to decode data field")
		}
		coerceInPlace(data)
		retval = []FluentRecordSet{
			{
				Tag: string(tag), // XXX: byte => rune
				Records: []FluentRecordType{
					&TinyFluentRecord{
						Timestamp: time.Unix(timestamp, 0),
						Data:      data,
					},
				},
			},
		}
	case time.Time:
		timestamp := timestamp_or_entries
		data, ok := v[2].(map[string]interface{})
		if !ok {
			return nil, errors.New("Failed to decode data field")
		}
		coerceInPlace(data)
		retval = []FluentRecordSet{
			{
				Tag: string(tag), // XXX: byte => rune
				Records: []FluentRecordType{
					&TinyFluentRecord{
						Timestamp: timestamp,
						Data:      data,
					},
				},
			},
		}
	case []interface{}: // Forward
		if !ok {
			return nil, errors.New("Unexpected payload format")
		}
		recordSet, err := decodeRecordSet(tag, timestamp_or_entries)
		if err != nil {
			return nil, err
		}
		retval = []FluentRecordSet{recordSet}
	case []byte: // PackedForward
		reader := bytes.NewReader(timestamp_or_entries)
		entries := make([]interface{}, 0)
		for {
			entry := make([]interface{}, 0)
			err := codec.NewDecoder(reader, &mh).Decode(&entry)
			if err == io.EOF {
				break
			} else if err != nil {
				return nil, errors.New("Unexpected payload format")
			}
			entries = append(entries, entry)
		}
		recordSet, err := decodeRecordSet(tag, entries)
		if err != nil {
			return nil, err
		}
		retval = []FluentRecordSet{recordSet}
	default:
		return nil, errors.New(fmt.Sprintf("Unknown type: %t", timestamp_or_entries))
	}
	return retval, nil
}

func toInt64(v interface{}) int64 {
	switch v := v.(type) {
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return int64(v)
	case uint:
		return int64(v)
	case uint32:
		return int64(v)
	case uint64:
		return int64(v)
	case float32:
		return int64(v)
	case float64:
		return int64(v)
	}
	return 0
}
