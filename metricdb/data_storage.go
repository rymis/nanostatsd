package metricdb

type DataStorage[T any] interface {
	BeginTransaction() error
	CommitTransaction() error
	RollbackTransaction() error
	WriteValue(name string, quant Quant, value *T) error
	Query(name string, begin, end Quant) ([]DataStorageRow[T], error)
	Reduce(width, end Quant, reduce func (name string, quant Quant, bucket []T) error) error
	RemoveBefore(quant Quant) error
	ListMetrics() ([]string, error)
	Close() error
}

type DataStorageRow[T any] struct {
	Name string
	Quant Quant
	Value *T
}
