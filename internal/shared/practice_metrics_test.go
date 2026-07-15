package shared

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func metricValue(t *testing.T, name string) float64 {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() != name || len(family.Metric) == 0 {
			continue
		}
		metric := family.Metric[0]
		if metric.Counter != nil {
			return metric.Counter.GetValue()
		}
		if metric.Gauge != nil {
			return metric.Gauge.GetValue()
		}
	}
	t.Fatalf("metric %q not registered", name)
	return 0
}

func TestPracticeReliabilityMetrics(t *testing.T) {
	ObserveOrderTransition(0, -1, "success")
	SetPaymentOutboxState(7, 90*time.Second)
	if got := metricValue(t, "gin_looklook_order_transitions_total"); got < 1 {
		t.Fatalf("transition counter = %v, want >= 1", got)
	}
	if got := metricValue(t, "gin_looklook_payment_outbox_pending"); got != 7 {
		t.Fatalf("outbox pending = %v, want 7", got)
	}
	if got := metricValue(t, "gin_looklook_payment_outbox_oldest_age_seconds"); got != 90 {
		t.Fatalf("outbox age = %v, want 90", got)
	}
}
