package platform

import (
	"strings"
	"testing"
)

func TestGenSNCompatibleLength(t *testing.T) {
	sn := GenSN("HSO")
	if len(sn) != 25 || !strings.HasPrefix(sn, "HSO") {
		t.Fatalf("unexpected sn %q", sn)
	}
	for _, ch := range sn[17:] {
		if ch < '0' || ch > '9' {
			t.Fatalf("suffix must be numeric: %q", sn)
		}
	}
}
