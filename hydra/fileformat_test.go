package hydra_test

import (
	"testing"
	"time"

	"github.com/fujiwara/fluent-agent-hydra/hydra"
)

func TestConvertMap(t *testing.T) {
	convertMap := hydra.NewConvertMap("user_id:integer,paid:bool,paid_user_amount:float")
	data := map[string]interface{}{
		"user_id":          "12345",
		"paid":             "true",
		"paid_user_amount": "1.234",
		"foo":              "45678",
	}
	convertMap.ConvertTypes(data)
	if data["user_id"] != 12345 {
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
