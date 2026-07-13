package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCacheSerialization(t *testing.T) {
	userData, err := json.Marshal(User{Mobile: "18888888888", Password: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(userData), "secret") || strings.Contains(string(userData), "Password") {
		t.Fatalf("password leaked into cache payload: %s", userData)
	}
	homestayData, err := json.Marshal(Homestay{FoodPrice: 100, HomestayPrice: 200, MarketHomestayPrice: 300})
	if err != nil {
		t.Fatal(err)
	}
	var decoded Homestay
	if err = json.Unmarshal(homestayData, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.FoodPrice != 100 || decoded.HomestayPrice != 200 || decoded.MarketHomestayPrice != 300 {
		t.Fatalf("prices lost in cache roundtrip: %+v", decoded)
	}
}
