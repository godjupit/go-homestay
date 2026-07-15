package payment

import (
	"testing"

	"gin-looklook/internal/order"
)

func TestPracticePayStatusMapping(t *testing.T) {
	tests := map[string]int64{"SUCCESS": StatusSuccess, "USERPAYING": StatusWait, "REFUND": StatusRefund, "CLOSED": StatusFail, "PAYERROR": StatusFail}
	for input, want := range tests {
		if got := payStatus(input); got != want {
			t.Errorf("payStatus(%q) = %d, want %d", input, got, want)
		}
	}
}

func TestPracticeValidatePayableOrder(t *testing.T) {
	if err := validatePayableOrder(&order.HomestayOrder{TradeState: order.TradeStateWaitPay, OrderTotalPrice: 9900}); err != nil {
		t.Fatalf("valid pending order rejected: %v", err)
	}
	invalid := []*order.HomestayOrder{nil, {TradeState: order.TradeStateWaitPay, OrderTotalPrice: 0}, {TradeState: order.TradeStateCancel, OrderTotalPrice: 9900}, {TradeState: order.TradeStateWaitUse, OrderTotalPrice: 9900}}
	for i, item := range invalid {
		if err := validatePayableOrder(item); err == nil {
			t.Errorf("invalid order %d was accepted", i)
		}
	}
}
