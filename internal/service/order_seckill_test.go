package service

import (
	"testing"
	"time"

	"gin-looklook/internal/model"
)

func TestBuildSeckillOrderUsesCampaignPrice(t *testing.T) {
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.Local)
	homestay := model.Homestay{ID: 11, HomestayPrice: 29900, MarketHomestayPrice: 39900, FoodPrice: 3000}
	order := buildOrder(homestay, 7, 9900, false, start.Unix(), start.Add(48*time.Hour).Unix(), 2, "seckill")
	if order.HomestayPrice != 9900 || order.HomestayTotalPrice != 19800 || order.OrderTotalPrice != 19800 {
		t.Fatalf("unexpected seckill amount: %+v", order)
	}
	if order.NeedFood != model.HomestayOrderNeedFoodNo || order.FoodTotalPrice != 0 {
		t.Fatalf("seckill order should not contain food: %+v", order)
	}
}

func TestSeckillIdentifiersAreUniqueAndDeterministic(t *testing.T) {
	first := makeSeckillReservationSN(1783930000, 100)
	second := makeSeckillReservationSN(1783930000, 101)
	if len(first) != 25 || first == second {
		t.Fatalf("invalid reservation ids: %q %q", first, second)
	}
	orderSN := seckillOrderSN(first, "fallback")
	if len(orderSN) != 25 || orderSN[:3] != "HSO" || orderSN[3:] != first[3:] {
		t.Fatalf("invalid deterministic order sn: %q", orderSN)
	}
}
