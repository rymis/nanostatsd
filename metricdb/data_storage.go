package metricdb

type DataStorage[T any] interface {
	BeginTransaction() error
	CommitTransaction() error
	RollbackTransaction() error
	WriteValue(name string, tags []string, quant Quant, value *T) error
	Query(name string, tags []string, begin, end Quant) ([]DataStorageRow[T], error)
	Reduce(width, end Quant, reduce DataStorageReduce[T]) error
	RemoveBefore(quant Quant) error
	ListMetrics() ([]string, error)
	Close() error
}

type DataStorageRow[T any] struct {
	Name string
	Tags string
	Quant Quant
	Value *T
}

type DataStorageReduce[T any] func (name string, tags []string, quant Quant, bucket []T) error
