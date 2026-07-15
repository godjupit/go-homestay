package search

import (
	"reflect"
	"testing"
)

func TestPracticeSplitTags(t *testing.T) {
	want := []string{"西湖", "亲子", "早餐"}
	if got := splitTags(" 西湖,亲子，，早餐, "); !reflect.DeepEqual(got, want) {
		t.Fatalf("splitTags() = %#v, want %#v", got, want)
	}
}

func TestPracticeSearchSort(t *testing.T) {
	if got := searchSort(nil, false, 0, 0); len(got) != 2 {
		t.Fatalf("default sort length = %d, want 2", len(got))
	}
	if got := searchSort([]string{"distance_asc"}, false, 0, 0); len(got) != 2 {
		t.Fatalf("invalid geo sort should fall back to default, got %#v", got)
	}
	got := searchSort([]string{"price_asc"}, false, 0, 0)
	if len(got) != 2 || !reflect.DeepEqual(got[0], map[string]any{"homestayPrice": "asc"}) || !reflect.DeepEqual(got[1], map[string]any{"id": "desc"}) {
		t.Fatalf("price sort = %#v", got)
	}
}
