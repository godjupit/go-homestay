package seckill

import (
	"strings"
	"testing"
)

func TestPracticeReservationSN(t *testing.T) {
	a := makeReservationSN(1784140000, 1)
	b := makeReservationSN(1784140000, 2)
	if len(a) != 25 || !strings.HasPrefix(a, "SKR") {
		t.Fatalf("reservation SN %q must have length 25 and SKR prefix", a)
	}
	if a == b {
		t.Fatalf("different sequences produced same reservation SN %q", a)
	}
}
