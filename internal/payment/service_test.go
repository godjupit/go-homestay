package payment

import (
	"testing"

	"gin-looklook/internal/order"
)

func TestValidatePayableOrder(t *testing.T) {
	valid := &order.HomestayOrder{TradeState: order.TradeStateWaitPay, OrderTotalPrice: 9900}
	if err := validatePayableOrder(valid); err != nil {
		t.Fatalf("valid order rejected: %v", err)
	}
	for name, ord := range map[string]*order.HomestayOrder{
		"nil":        nil,
		"zero":       {TradeState: order.TradeStateWaitPay},
		"canceled":   {TradeState: order.TradeStateCancel, OrderTotalPrice: 9900},
		"alreadyPay": {TradeState: order.TradeStateWaitUse, OrderTotalPrice: 9900},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validatePayableOrder(ord); err == nil {
				t.Fatal("invalid order was accepted")
			}
		})
	}
}

func TestPayStatus(t *testing.T) {
	tests := map[string]int64{
		"SUCCESS":    StatusSuccess,
		"USERPAYING": StatusWait,
		"REFUND":     StatusRefund,
		"CLOSED":     StatusFail,
		"UNKNOWN":    StatusFail,
	}
	for input, want := range tests {
		if got := payStatus(input); got != want {
			t.Errorf("payStatus(%q) = %d, want %d", input, got, want)
		}
	}
}
