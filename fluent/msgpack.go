package fluent

import (
	"encoding/binary"
	"bytes"
)

const (
	mpInt64     byte = 0xd3
	mpStr            = 0xa0
	mpStr8           = 0xd9
	mpStr16          = 0xda
	mpStr32          = 0xdb
	mp2ElmArray      = 0x92
	mp1ElmMap        = 0x81
	mpBytes8         = 0xc4
	mpBytes16        = 0xc5
	mpBytes32        = 0xc6
)

func writeMpStringHead(buf *bytes.Buffer, l int) {
	switch {
	case l < 32:
		buf.WriteByte(mpStr | byte(l))
	case l < 256:
		buf.WriteByte(mpStr8)
		binary.Write(buf, binary.BigEndian, uint8(l))
	case l < 65536:
		buf.WriteByte(mpStr16)
		binary.Write(buf, binary.BigEndian, uint16(l))
	default:
		buf.WriteByte(mpStr32)
		binary.Write(buf, binary.BigEndian, uint32(l))
	}
}

func writeMpBytesHead(buf *bytes.Buffer, l int) {
	switch {
	case l < 256:
		buf.WriteByte(mpBytes8)
		binary.Write(buf, binary.BigEndian, uint8(l))
	case l < 65536:
		buf.WriteByte(mpBytes16)
		binary.Write(buf, binary.BigEndian, uint16(l))
	default:
		buf.WriteByte(mpBytes32)
		binary.Write(buf, binary.BigEndian, uint32(l))
	}
}

func toMsgpackRecord(ts int64, key string, value []byte) []byte {
	buf := new(bytes.Buffer)
	// 2 elments array [ts, {key: value}]
	buf.WriteByte(mp2ElmArray)
	// ts
	buf.WriteByte(mpInt64)
	binary.Write(buf, binary.BigEndian, ts)
	// 1 element map {key: value}
	buf.WriteByte(mp1ElmMap)
	// key
	writeMpStringHead(buf, len(key))
	buf.WriteString(key)
	// value
	writeMpStringHead(buf, len(value))
	buf.Write(value)
	return buf.Bytes()
}

func toMsgpackRecordSet(tag string, bin *[]byte) []byte {
	buf := new(bytes.Buffer)
	// 2 elments array [ts, bin]
	buf.WriteByte(mp2ElmArray)
	// tag
	writeMpStringHead(buf, len(tag))
	buf.WriteString(tag)
	// buf
	writeMpBytesHead(buf, len(*bin))
	buf.Write(*bin)
	return buf.Bytes()
}
