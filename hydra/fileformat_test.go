package hydra_test

import (
	"testing"

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
	hydra.ConvertTypes(data, convertMap)
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
