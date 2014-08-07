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
	"errors"
	"fmt"
	"github.com/ugorji/go/codec"
	"io"
	"net"
	"reflect"
)

type FluentRecord struct {
	Tag       string
	Timestamp uint64
	Data      map[string]interface{}
}

func (r *FluentRecord) Pack() ([]byte, error) {
	msg := []interface{}{r.Tag, r.Timestamp, r.Data}
	if data, dumperr := toMsgpack(msg); dumperr != nil {
		fmt.Println("Can't convert to msgpack:", msg, dumperr)
		return nil, dumperr
	} else {
		return data, nil
	}
}

type TinyFluentRecord struct {
	Timestamp uint64
	Data      map[string]interface{}
}

func (r *TinyFluentRecord) Pack() ([]byte, error) {
	msg := []interface{}{r.Timestamp, r.Data}
	if data, dumperr := toMsgpack(msg); dumperr != nil {
		fmt.Println("Can't convert to msgpack:", msg, dumperr)
		return nil, dumperr
	} else {
		return data, nil
	}
}

type FluentRecordSet struct {
	Tag     string
	Records []TinyFluentRecord
}

func (rs *FluentRecordSet) PackAsPacketForward() ([]byte, error) {
	buffer := make([]byte, 0, len(rs.Records)*1024)
	for _, record := range rs.Records {
		data, err := record.Pack()
		if err != nil {
			return nil, err
		}
		buffer = append(buffer, data...)
	}
	if data, dumperr := toMsgpack([]interface{}{rs.Tag, buffer}); dumperr != nil {
		fmt.Println("Can't convert to msgpack")
		return nil, dumperr
	} else {
		return data, nil
	}
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
	if data, dumperr := toMsgpack([]interface{}{rs.Tag, records}); dumperr != nil {
		fmt.Println("Can't convert to msgpack")
		return nil, dumperr
	} else {
		return data, nil
	}
}

func coerceInPlace(data map[string]interface{}) {
	for k, v := range data {
		switch v_ := v.(type) {
		case []byte:
			data[k] = string(v_) // XXX: byte => rune
		case map[string]interface{}:
			coerceInPlace(v_)
		}
	}
}

func decodeRecordSet(tag []byte, entries []interface{}) (FluentRecordSet, error) {
	records := make([]TinyFluentRecord, len(entries))
	for i, _entry := range entries {
		entry, ok := _entry.([]interface{})
		if !ok {
			return FluentRecordSet{}, errors.New("Failed to decode recordSet")
		}
		timestamp, ok := entry[0].(uint64)
		if !ok {
			return FluentRecordSet{}, errors.New("Failed to decode timestamp field")
		}
		data, ok := entry[1].(map[string]interface{})
		if !ok {
			return FluentRecordSet{}, errors.New("Failed to decode data field")
		}
		coerceInPlace(data)
		records[i] = TinyFluentRecord{
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
	var mh codec.MsgpackHandle
	mh.MapType = reflect.TypeOf(map[string]interface{}(nil))
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
	case uint64:
		timestamp := timestamp_or_entries
		data, ok := v[2].(map[string]interface{})
		if !ok {
			return nil, errors.New("Failed to decode data field")
		}
		coerceInPlace(data)
		retval = []FluentRecordSet{
			{
				Tag: string(tag), // XXX: byte => rune
				Records: []TinyFluentRecord{
					{
						Timestamp: timestamp,
						Data:      data,
					},
				},
			},
		}
	case float64:
		timestamp := uint64(timestamp_or_entries)
		data, ok := v[2].(map[string]interface{})
		if !ok {
			return nil, errors.New("Failed to decode data field")
		}
		retval = []FluentRecordSet{
			{
				Tag: string(tag), // XXX: byte => rune
				Records: []TinyFluentRecord{
					{
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
