package service

import (
	"testing"

	"gin-looklook/internal/model"
)

func TestVerifyState(t *testing.T) {
	tests := []struct {
		old, next int64
		want      bool
	}{
		{model.OrderTradeStateWaitPay, model.OrderTradeStateCancel, true},
		{model.OrderTradeStateWaitPay, model.OrderTradeStateWaitUse, true},
		{model.OrderTradeStateWaitUse, model.OrderTradeStateUsed, true},
		{model.OrderTradeStateWaitUse, model.OrderTradeStateRefund, true},
		{model.OrderTradeStateWaitPay, model.OrderTradeStateUsed, false},
		{model.OrderTradeStateCancel, model.OrderTradeStateWaitUse, false},
	}
	for _, tt := range tests {
		if got := verifyState(tt.old, tt.next); got != tt.want {
			t.Fatalf("verifyState(%d,%d)=%v, want %v", tt.old, tt.next, got, tt.want)
		}
	}
}

func TestPayStatus(t *testing.T) {
	if payStatus("SUCCESS") != model.PaymentStatusSuccess || payStatus("REFUND") != model.PaymentStatusWait || payStatus("CLOSED") != model.PaymentStatusFail {
		t.Fatal("unexpected payment status mapping")
	}
}

func TestParseStar(t *testing.T) {
	if got := ParseStar([]byte("4.8")); got != 4.8 {
		t.Fatalf("got %v", got)
	}
	if got := ParseStar([]byte("bad")); got != 0 {
		t.Fatalf("got %v", got)
	}
}
