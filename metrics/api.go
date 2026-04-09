package metrics

import "context"

// MetricsAPI роль LEGO «metrics/»: агрегаты по шине, KPI-слой без записи в доменную БД.
type MetricsAPI interface {
	Counts() map[string]int64
	Start(ctx context.Context) error
	Stop() error
}
