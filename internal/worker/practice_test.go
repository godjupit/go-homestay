package worker

import (
	"testing"

	"gin-looklook/internal/order"
	"gin-looklook/internal/payment"
)

func TestPracticeOrderStateForPayment(t *testing.T) {
	tests := []struct {
		payStatus int64
		wantState int64
		wantOK    bool
	}{{payment.StatusSuccess, order.TradeStateWaitUse, true}, {payment.StatusRefund, order.TradeStateRefund, true}, {payment.StatusWait, 0, false}, {payment.StatusFail, 0, false}}
	for _, tt := range tests {
		state, ok := orderStateForPayment(tt.payStatus)
		if state != tt.wantState || ok != tt.wantOK {
			t.Errorf("orderStateForPayment(%d) = (%d,%v), want (%d,%v)", tt.payStatus, state, ok, tt.wantState, tt.wantOK)
		}
	}
}
