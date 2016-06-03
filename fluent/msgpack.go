package fluent

import (
	"bytes"
	"encoding/binary"
	"io"
	"reflect"
	"time"

	"github.com/ugorji/go/codec"
)

const (
	mpInt32         byte = 0xd2
	mpInt64              = 0xd3
	mpUint32             = 0xce
	mpUint64             = 0xcf
	mpStr                = 0xa0
	mpStr8               = 0xd9
	mpStr16              = 0xda
	mpStr32              = 0xdb
	mp2ElmArray          = 0x92
	mp1ElmMap            = 0x81
	mpBytes8             = 0xc4
	mpBytes16            = 0xc5
	mpBytes32            = 0xc6
	mpExtension8         = 0xd7
	mpEventTimeType      = 0x00
)

var (
	mh codec.MsgpackHandle

	EnableEventTime = false
)

func init() {
	mh.MapType = reflect.TypeOf(map[string]interface{}(nil))
	mh.SetBytesExt(reflect.TypeOf(time.Time{}), 0, &EventTimeExtension{})
}

func writeMsgpack(w io.Writer, val interface{}) error {
	return codec.NewEncoder(w, &mh).Encode(val)
}

type msgpackBuffer struct {
	bytes.Buffer
}

func (b *msgpackBuffer) WriteValue(v interface{}) {
	binary.Write(b, binary.BigEndian, v)
}

func (b *msgpackBuffer) WriteTime(ts time.Time) {
	if EnableEventTime {
		// EventTime
		b.WriteByte(mpExtension8)
		b.WriteByte(mpEventTimeType)
		b.WriteValue(uint32(ts.Unix()))
		b.WriteValue(uint32(ts.Nanosecond()))
	} else {
		// int64
		b.WriteByte(mpInt64)
		b.WriteValue(ts.Unix())
	}
}

func (b *msgpackBuffer) WriteMpStringHead(l int) {
	switch {
	case l < 32:
		b.WriteByte(mpStr | byte(l))
	case l < 256:
		b.WriteByte(mpStr8)
		b.WriteValue(uint8(l))
	case l < 65536:
		b.WriteByte(mpStr16)
		b.WriteValue(uint16(l))
	default:
		b.WriteByte(mpStr32)
		b.WriteValue(uint32(l))
	}
}

func (b *msgpackBuffer) WriteMpBytesHead(l int) {
	switch {
	case l < 256:
		b.WriteByte(mpBytes8)
		b.WriteValue(uint8(l))
	case l < 65536:
		b.WriteByte(mpBytes16)
		b.WriteValue(uint16(l))
	default:
		b.WriteByte(mpBytes32)
		b.WriteValue(uint32(l))
	}
}

func toMsgpackTinyMessage(ts time.Time, key string, value []byte) []byte {
	b := new(msgpackBuffer)
	// required capacity
	b.Grow(8 + len(key) + len(value) + 8)
	// 2 elments array [ts, {key: value}]
	b.WriteByte(mp2ElmArray)
	// ts
	b.WriteTime(ts)
	// 1 element map {key: value}
	b.WriteByte(mp1ElmMap)
	// key
	b.WriteMpStringHead(len(key))
	b.WriteString(key)
	// value
	b.WriteMpStringHead(len(value))
	b.Write(value)
	return b.Bytes()
}

func toMsgpackRecord(ts time.Time, data map[string]interface{}) ([]byte, error) {
	b := new(msgpackBuffer)
	// 2 elments array [ts, data]
	b.WriteByte(mp2ElmArray)
	// ts
	b.WriteTime(ts)
	// data
	if err := writeMsgpack(b, data); err != nil {
		return []byte{}, err
	}
	return b.Bytes(), nil
}

func toMsgpackRecordSet(tag string, bin []byte) []byte {
	b := new(msgpackBuffer)
	// required capacity
	b.Grow(len(tag) + len(bin) + 16)
	// 2 elments array [tag, bin]
	b.WriteByte(mp2ElmArray)
	// tag
	b.WriteMpStringHead(len(tag))
	b.WriteString(tag)
	// buf
	b.WriteMpStringHead(len(bin))
	b.Write(bin)
	return b.Bytes()
}
