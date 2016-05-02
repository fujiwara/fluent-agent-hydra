package fluent

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import "github.com/tinylib/msgp/msgp"

// DecodeMsg implements msgp.Decodable
func (z *FluentRecord) DecodeMsg(dc *msgp.Reader) (err error) {
	var bai uint32
	bai, err = dc.ReadArrayHeader()
	if err != nil {
		return
	}
	if bai != 3 {
		err = msgp.ArrayError{Wanted: 3, Got: bai}
		return
	}
	z.Tag, err = dc.ReadString()
	if err != nil {
		return
	}
	z.Timestamp, err = dc.ReadInt64()
	if err != nil {
		return
	}
	var cmr uint32
	cmr, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	if z.Data == nil && cmr > 0 {
		z.Data = make(map[string]interface{}, cmr)
	} else if len(z.Data) > 0 {
		for key, _ := range z.Data {
			delete(z.Data, key)
		}
	}
	for cmr > 0 {
		cmr--
		var xvk string
		var bzg interface{}
		xvk, err = dc.ReadString()
		if err != nil {
			return
		}
		bzg, err = dc.ReadIntf()
		if err != nil {
			return
		}
		z.Data[xvk] = bzg
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *FluentRecord) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 3
	err = en.Append(0x93)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Tag)
	if err != nil {
		return
	}
	err = en.WriteInt64(z.Timestamp)
	if err != nil {
		return
	}
	err = en.WriteMapHeader(uint32(len(z.Data)))
	if err != nil {
		return
	}
	for xvk, bzg := range z.Data {
		err = en.WriteString(xvk)
		if err != nil {
			return
		}
		err = en.WriteIntf(bzg)
		if err != nil {
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *FluentRecord) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 3
	o = append(o, 0x93)
	o = msgp.AppendString(o, z.Tag)
	o = msgp.AppendInt64(o, z.Timestamp)
	o = msgp.AppendMapHeader(o, uint32(len(z.Data)))
	for xvk, bzg := range z.Data {
		o = msgp.AppendString(o, xvk)
		o, err = msgp.AppendIntf(o, bzg)
		if err != nil {
			return
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *FluentRecord) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var ajw uint32
	ajw, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		return
	}
	if ajw != 3 {
		err = msgp.ArrayError{Wanted: 3, Got: ajw}
		return
	}
	z.Tag, bts, err = msgp.ReadStringBytes(bts)
	if err != nil {
		return
	}
	z.Timestamp, bts, err = msgp.ReadInt64Bytes(bts)
	if err != nil {
		return
	}
	var wht uint32
	wht, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	if z.Data == nil && wht > 0 {
		z.Data = make(map[string]interface{}, wht)
	} else if len(z.Data) > 0 {
		for key, _ := range z.Data {
			delete(z.Data, key)
		}
	}
	for wht > 0 {
		var xvk string
		var bzg interface{}
		wht--
		xvk, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			return
		}
		bzg, bts, err = msgp.ReadIntfBytes(bts)
		if err != nil {
			return
		}
		z.Data[xvk] = bzg
	}
	o = bts
	return
}

func (z *FluentRecord) Msgsize() (s int) {
	s = 1 + msgp.StringPrefixSize + len(z.Tag) + msgp.Int64Size + msgp.MapHeaderSize
	if z.Data != nil {
		for xvk, bzg := range z.Data {
			_ = bzg
			s += msgp.StringPrefixSize + len(xvk) + msgp.GuessSize(bzg)
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *TinyFluentMessage) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var hct uint32
	hct, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for hct > 0 {
		hct--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Timestamp":
			z.Timestamp, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "FieldName":
			z.FieldName, err = dc.ReadString()
			if err != nil {
				return
			}
		case "Message":
			z.Message, err = dc.ReadBytes(z.Message)
			if err != nil {
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *TinyFluentMessage) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 3
	// write "Timestamp"
	err = en.Append(0x83, 0xa9, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Timestamp)
	if err != nil {
		return
	}
	// write "FieldName"
	err = en.Append(0xa9, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x4e, 0x61, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteString(z.FieldName)
	if err != nil {
		return
	}
	// write "Message"
	err = en.Append(0xa7, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteBytes(z.Message)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *TinyFluentMessage) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 3
	// string "Timestamp"
	o = append(o, 0x83, 0xa9, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70)
	o = msgp.AppendInt64(o, z.Timestamp)
	// string "FieldName"
	o = append(o, 0xa9, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x4e, 0x61, 0x6d, 0x65)
	o = msgp.AppendString(o, z.FieldName)
	// string "Message"
	o = append(o, 0xa7, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65)
	o = msgp.AppendBytes(o, z.Message)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *TinyFluentMessage) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var cua uint32
	cua, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for cua > 0 {
		cua--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Timestamp":
			z.Timestamp, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "FieldName":
			z.FieldName, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "Message":
			z.Message, bts, err = msgp.ReadBytesBytes(bts, z.Message)
			if err != nil {
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

func (z *TinyFluentMessage) Msgsize() (s int) {
	s = 1 + 10 + msgp.Int64Size + 10 + msgp.StringPrefixSize + len(z.FieldName) + 8 + msgp.BytesPrefixSize + len(z.Message)
	return
}

// DecodeMsg implements msgp.Decodable
func (z *TinyFluentRecord) DecodeMsg(dc *msgp.Reader) (err error) {
	var daf uint32
	daf, err = dc.ReadArrayHeader()
	if err != nil {
		return
	}
	if daf != 2 {
		err = msgp.ArrayError{Wanted: 2, Got: daf}
		return
	}
	z.Timestamp, err = dc.ReadInt64()
	if err != nil {
		return
	}
	var pks uint32
	pks, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	if z.Data == nil && pks > 0 {
		z.Data = make(map[string]interface{}, pks)
	} else if len(z.Data) > 0 {
		for key, _ := range z.Data {
			delete(z.Data, key)
		}
	}
	for pks > 0 {
		pks--
		var xhx string
		var lqf interface{}
		xhx, err = dc.ReadString()
		if err != nil {
			return
		}
		lqf, err = dc.ReadIntf()
		if err != nil {
			return
		}
		z.Data[xhx] = lqf
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *TinyFluentRecord) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 2
	err = en.Append(0x92)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Timestamp)
	if err != nil {
		return
	}
	err = en.WriteMapHeader(uint32(len(z.Data)))
	if err != nil {
		return
	}
	for xhx, lqf := range z.Data {
		err = en.WriteString(xhx)
		if err != nil {
			return
		}
		err = en.WriteIntf(lqf)
		if err != nil {
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *TinyFluentRecord) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 2
	o = append(o, 0x92)
	o = msgp.AppendInt64(o, z.Timestamp)
	o = msgp.AppendMapHeader(o, uint32(len(z.Data)))
	for xhx, lqf := range z.Data {
		o = msgp.AppendString(o, xhx)
		o, err = msgp.AppendIntf(o, lqf)
		if err != nil {
			return
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *TinyFluentRecord) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var jfb uint32
	jfb, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		return
	}
	if jfb != 2 {
		err = msgp.ArrayError{Wanted: 2, Got: jfb}
		return
	}
	z.Timestamp, bts, err = msgp.ReadInt64Bytes(bts)
	if err != nil {
		return
	}
	var cxo uint32
	cxo, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	if z.Data == nil && cxo > 0 {
		z.Data = make(map[string]interface{}, cxo)
	} else if len(z.Data) > 0 {
		for key, _ := range z.Data {
			delete(z.Data, key)
		}
	}
	for cxo > 0 {
		var xhx string
		var lqf interface{}
		cxo--
		xhx, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			return
		}
		lqf, bts, err = msgp.ReadIntfBytes(bts)
		if err != nil {
			return
		}
		z.Data[xhx] = lqf
	}
	o = bts
	return
}

func (z *TinyFluentRecord) Msgsize() (s int) {
	s = 1 + msgp.Int64Size + msgp.MapHeaderSize
	if z.Data != nil {
		for xhx, lqf := range z.Data {
			_ = lqf
			s += msgp.StringPrefixSize + len(xhx) + msgp.GuessSize(lqf)
		}
	}
	return
}
