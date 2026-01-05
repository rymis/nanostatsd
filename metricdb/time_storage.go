package metricdb

type TimeStorageElement[T any] struct {
	Quant Quant `json:"quant"`
	Value T `json:"value"`
}

// Storage for time series data that allows to search for values
type TimeStorage[T any] struct {
	Values []TimeStorageElement[T]
}

// Append value to storage
func (ts* TimeStorage[T]) Append(quant Quant, value T) {
	ts.Values = append(ts.Values, TimeStorageElement[T]{
		Quant: quant,
		Value: value,
	})

	// Now we move it to the right place:
	i := len(ts.Values) - 1
	for i > 0 {
		if ts.Values[i - 1].Quant <= ts.Values[i].Quant {
			break
		}
	}
}

// Query interval
func (ts* TimeStorage[T]) Query(begin, end Quant) []TimeStorageElement[T] {
	l := ts.LowerBound(0, begin)
	if l == len(ts.Values) {
		return nil
	}
	r := ts.LowerBound(l, end)
	if r < len(ts.Values) {
		r += 1
	}

	return ts.Values[l: r]
}

// Get quant lower bound index
func (ts* TimeStorage[T]) LowerBound(first int, q Quant) int {
	i := 0
	count := len(ts.Values)
	for count > 0 {
		step := count / 2;
		i = first + step
		if ts.Values[i].Quant < q {
			first = i + 1
			count -= step + 1
		} else {
			count = step
		}
	}
	return first
}

// Create new time storage
func NewTimeStorage[T any]() *TimeStorage[T] {
	return &TimeStorage[T]{
		Values: make([]TimeStorageElement[T], 0, 64),
	}
}

// Remove elements from the time range
func (ts *TimeStorage[T]) Pop(begin, end Quant) []TimeStorageElement[T] {
	// Pop values from the storage and return them as a slice
	l := ts.LowerBound(0, begin)
	if l == len(ts.Values) {
		return nil
	}
	r := ts.LowerBound(l, end)
	if r < len(ts.Values) {
		r += 1
	}

	res := ts.Values[l:r]

	ts.Values = append(ts.Values[:l], ts.Values[r:]...)

	return res
}
