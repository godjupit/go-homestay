package order

import (
	"testing"
	"time"

	"gin-looklook/internal/travel"
)

func TestPracticeStayNights(t *testing.T) {
	start := time.Date(2026, 7, 20, 14, 0, 0, 0, time.UTC).Unix()
	tests := []struct {
		name    string
		end     int64
		want    int64
		wantErr bool
	}{
		{"end before start", start - 1, 0, true},
		{"less than one night", start + int64(23*time.Hour), 0, true},
		{"exactly one night", start + int64(24*time.Hour), 1, false},
		{"two nights", start + int64(48*time.Hour), 2, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := stayNights(start, tt.end)
			if (err != nil) != tt.wantErr || got != tt.want {
				t.Fatalf("stayNights() = (%d, %v), want (%d, error=%v)", got, err, tt.want, tt.wantErr)
			}
		})
	}
}

func TestPracticeBuildOrderPricingAndSnapshot(t *testing.T) {
	h := travel.Homestay{ID: 11, Title: "Lake House", SubTitle: "quiet", Banner: "cover.jpg,second.jpg", Info: "info", PeopleNum: 4, RowType: 1, FoodInfo: "breakfast", FoodPrice: 3000, HomestayPrice: 29900, MarketHomestayPrice: 39900, HomestayBusinessID: 7, UserID: 9}
	start := time.Now().Unix()
	v := buildOrder(h, 42, h.HomestayPrice, true, start, start+int64(48*time.Hour), 2, 3, "practice")
	if v.SN == "" || v.TradeCode == "" || v.TradeState != TradeStateWaitPay {
		t.Fatalf("order identifiers/state not initialized: %+v", v)
	}
	if v.UserID != 42 || v.HomestayID != 11 || v.Title != h.Title || v.Cover != "cover.jpg" {
		t.Fatalf("order snapshot is incomplete: %+v", v)
	}
	if v.HomestayTotalPrice != 59800 || v.FoodTotalPrice != 18000 || v.OrderTotalPrice != 77800 || v.NeedFood != NeedFoodYes {
		t.Fatalf("unexpected pricing: homestay=%d food=%d total=%d", v.HomestayTotalPrice, v.FoodTotalPrice, v.OrderTotalPrice)
	}
}

func TestPracticeOrderStateMachine(t *testing.T) {
	allowed := [][2]int64{{TradeStateWaitPay, TradeStateCancel}, {TradeStateWaitPay, TradeStateWaitUse}, {TradeStateWaitUse, TradeStateUsed}, {TradeStateWaitUse, TradeStateRefund}, {TradeStateWaitUse, TradeStateExpire}}
	for _, transition := range allowed {
		if !verifyState(transition[0], transition[1]) {
			t.Errorf("expected transition %d -> %d to be allowed", transition[0], transition[1])
		}
	}
	denied := [][2]int64{{TradeStateCancel, TradeStateWaitPay}, {TradeStateWaitUse, TradeStateCancel}, {TradeStateUsed, TradeStateRefund}, {TradeStateWaitPay, TradeStateUsed}, {TradeStateWaitPay, 99}}
	for _, transition := range denied {
		if verifyState(transition[0], transition[1]) {
			t.Errorf("expected transition %d -> %d to be denied", transition[0], transition[1])
		}
	}
}
