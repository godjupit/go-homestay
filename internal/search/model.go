package search

type OutboxEvent struct {
	ID          int64  `gorm:"column:id;primaryKey;autoIncrement"`
	EventKey    string `gorm:"column:event_key"`
	AggregateID int64  `gorm:"column:aggregate_id"`
	EventType   string `gorm:"column:event_type"`
	RetryCount  int64  `gorm:"column:retry_count"`
}

func (OutboxEvent) TableName() string { return "search_event_outbox" }
