package hydra_test

import (
	"testing"
	"time"

	"github.com/fujiwara/fluent-agent-hydra/hydra"
)

func TestConvertMap(t *testing.T) {
	convertMap := hydra.NewConvertMap("user_id:integer,paid:bool,paid_user_amount:float,bar:integer,baz:integer")
	data := map[string]interface{}{
		"user_id":          "12345",
		"paid":             "true",
		"paid_user_amount": "1.234",
		"foo":              "45678",
		"bar":              float64(67890),
		"baz":              98765,
	}
	convertMap.ConvertTypes(data)
	if data["user_id"] != int64(12345) {
		t.Errorf("convert integer failed")
	}
	if data["paid"] != true {
		t.Errorf("convert bool failed")
	}
	if data["paid_user_amount"] != float64(1.234) {
		t.Errorf("convert float failed %#v", data["paid_user_amount"])
	}
	if data["foo"] != "45678" {
		t.Errorf("foo must be not converted %#v", data["foo"])
	}
	if data["bar"] != int64(67890) {
		t.Errorf("bar must be not converted %#v", data["bar"])
	}
	if data["baz"] != int64(98765) {
		t.Errorf("baz must be not converted %#v", data["baz"])
	}
}

func TestTimeConverter(t *testing.T) {
	tc := hydra.TimeConverter(time.RFC3339)
	ts, err := tc.Convert("2015-05-26T11:22:33+09:00")
	if err != nil {
		t.Error(err)
	}
	if ts.Unix() != 1432606953 {
		t.Error("2015-05-26T11:22:33+09:00 != 1432606953")
	}
}

func TestTimeConverterUnix(t *testing.T) {
	tc := hydra.TimeConverter("unix")
	ts, err := tc.Convert("1469429601.376390000")
	if err != nil {
		t.Error(err)
	}
	if ts.Unix() != 1469429601 || ts.Nanosecond() != 376390000 {
		t.Errorf("1469429601.376390000 sec != %s", ts)
	}

	ts, err = tc.Convert("1469429601")
	if err != nil {
		t.Error(err)
	}
	if ts.Unix() != 1469429601 || ts.Nanosecond() != 0 {
		t.Errorf("1469429601 sec != %s", ts)
	}

	ts, err = tc.Convert("1469429601.123")
	if err != nil {
		t.Error(err)
	}
	if ts.Unix() != 1469429601 || ts.Nanosecond() != 123000000 {
		t.Errorf("1469429601.123 sec != %s", ts)
	}

	ts, err = tc.Convert("1469429601.123456789123")
	if err != nil {
		t.Error(err)
	}
	if ts.Unix() != 1469429601 || ts.Nanosecond() != 123456789 {
		t.Errorf("1469429601.123456789123 sec != %s", ts)
	}
}

func BenchmarkConvertMap(b *testing.B) {
	convertMap := hydra.NewConvertMap("user_id:integer,paid:bool,paid_user_amount:float")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data := map[string]interface{}{
			"user_id":          "12345",
			"paid":             "true",
			"paid_user_amount": "1.234",
			"foo":              "45678",
		}
		convertMap.ConvertTypes(data)
	}
}
