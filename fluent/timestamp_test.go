package fluent_test

import (
	"reflect"
	"testing"
)

var testNumbers = []interface{}{
	int(10),
	uint(10),
	int32(10),
	uint32(10),
	int64(10),
	uint64(10),
	float32(10.123),
	float32(10.234),
}

func TestToInt64(t *testing.T) {
	for _, n := range testNumbers {
		if toInt64(n) != int64(10) {
			t.Errorf("toInt64 %s can't convert to int64(10)", n)
		}
		if toInt64ReflectConvert(n) != int64(10) {
			t.Errorf("toInt64ReflectConvert %s can't convert to int64(10)", n)
		}
		if toInt64If(n) != int64(10) {
			t.Errorf("toInt64If %s can't convert to int64(10)", n)
		}
		if toInt64Switch(n) != int64(10) {
			t.Errorf("toInt64Switch %s can't convert to int64(10)", n)
		}
	}
}

func toInt64(v interface{}) int64 {
	switch v.(type) {
	case int, int32, int64:
		return reflect.ValueOf(v).Int()
	case uint, uint32, uint64:
		return int64(reflect.ValueOf(v).Uint())
	case float32, float64:
		return int64(reflect.ValueOf(v).Float())
	default:
		return int64(0)
	}
}

func toInt64ReflectConvert(v interface{}) int64 {
	var i int64
	switch v.(type) {
	case int, uint, int64, uint64, int32, uint32, float32, float64:
		value := reflect.ValueOf(v)
		v2 := value.Convert(reflect.TypeOf(i))
		i = v2.Int()
	}
	return i
}

func toInt64Switch(v interface{}) int64 {
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

func toInt64If(v interface{}) int64 {
	var i int64
	switch v.(type) {
	case int, uint, int64, uint64, int32, uint32, float32, float64:
		if value, ok := v.(int64); ok {
			i = value
		} else if value, ok := v.(int); ok {
			i = int64(value)
		} else if value, ok := v.(uint); ok {
			i = int64(value)
		} else if value, ok := v.(uint64); ok {
			i = int64(value)
		} else if value, ok := v.(int32); ok {
			i = int64(value)
		} else if value, ok := v.(uint32); ok {
			i = int64(value)
		} else if value, ok := v.(float32); ok {
			i = int64(value)
		} else if value, ok := v.(float64); ok {
			i = int64(value)
		}
	}
	return i
}

func BenchmarkToInt64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, n := range testNumbers {
			_ = toInt64(n)
		}
	}
}

func BenchmarkToInt64ReflectConvert(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, n := range testNumbers {
			_ = toInt64ReflectConvert(n)
		}
	}
}

func BenchmarkToInt64If(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, n := range testNumbers {
			_ = toInt64If(n)
		}
	}
}

func BenchmarkToInt64Switch(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, n := range testNumbers {
			_ = toInt64Switch(n)
		}
	}
}
