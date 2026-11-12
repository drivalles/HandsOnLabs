package lddata_test

import (
	"ld-backend/internal/impl/core/data"
	"testing"
)

func TestPhones(t *testing.T) {
	number := lddata.KeepDigitsOnly("1(665)  234d ffgd567d fgh89 0")
	if number != "1665234567890" {
		t.Fatal("unexpected number " + number)
	}
}
