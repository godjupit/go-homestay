package shared

import "time"

func ObserveOrderTransition(from, to int64, result string) {
	// TODO(practice-09): 用低基数 CounterVec 记录订单状态迁移结果。
}

func SetPaymentOutboxState(pending int, oldestAge time.Duration) {
	// TODO(practice-09): 用 Gauge 记录 Outbox 积压数和最老事件延迟（秒）。
}
