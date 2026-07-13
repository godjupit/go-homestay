package platform

import "github.com/prometheus/client_golang/prometheus"

var (
	seckillReservations = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "gin_looklook_seckill_reservations_total", Help: "Seckill reservation outcomes."}, []string{"result"})
	seckillOrders       = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "gin_looklook_seckill_orders_total", Help: "Seckill asynchronous order outcomes."}, []string{"result"})
)

func init() {
	prometheus.MustRegister(seckillReservations, seckillOrders)
}

func ObserveSeckillReservation(result string) { seckillReservations.WithLabelValues(result).Inc() }
func ObserveSeckillOrder(result string)       { seckillOrders.WithLabelValues(result).Inc() }
