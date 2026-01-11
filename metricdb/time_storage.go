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

	return ts.Values[l: r]
}

// Get quant lower bound index
func (ts* TimeStorage[T]) LowerBound(first int, q Quant) int {
	i := 0
	count := len(ts.Values) - first
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

// Remove elements older than quant
func (ts *TimeStorage[T]) RemoveBefore(quant Quant) {
	if len(ts.Values) == 0 {
		return
	}

	if ts.Values[0].Quant >= quant {
		return
	}

	l := ts.LowerBound(0, quant)
	if l >= len(ts.Values) {
		ts.Values = make([]TimeStorageElement[T], 0, 64)
		return
	}

	vals := make([]TimeStorageElement[T], 0, 64)
	vals = append(vals, ts.Values[l:]...)
	ts.Values = vals
}

// Merge 2 time storages
func (ts *TimeStorage[T]) Merge(ts2 *TimeStorage[T]) {
	res := make([]TimeStorageElement[T], 0, len(ts.Values) + len(ts2.Values))

	i1 := 0
	i2 := 0
	for i1 < len(ts.Values) && i2 < len(ts2.Values) {
		if ts.Values[i1].Quant < ts2.Values[i2].Quant {
			res = append(res, ts.Values[i1])
			i1++
		} else {
			res = append(res, ts2.Values[i2])
			i2++
		}
	}

	for i1 < len(ts.Values) {
		res = append(res, ts.Values[i1])
		i1++
	}

	for i2 < len(ts2.Values) {
		res = append(res, ts2.Values[i2])
		i2++
	}

	ts.Values = res
}
