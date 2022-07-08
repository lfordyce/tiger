package statistics

import (
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
	"time"
)

// A Sample is a single measurement.
type Sample struct {
	WorkerID   int
	Elapsed    float64
	Overhead   time.Duration
	HostnameID string
	StartTime  time.Time
	EndTime    time.Time
}

// GroupedSample represents all the measurements grouped by hostname.
type GroupedSample struct {
	HostnameID string
	Elapsed    []float64
	Overhead   []time.Duration
}

// Number represents and numeric type
type Number interface {
	constraints.Float | constraints.Integer
}

// Sum adds all the numbers of a slice together
func Sum[T Number](data []T) T {
	var sum T = 0
	for _, n := range data {
		sum += n
	}
	return sum
}

// Mean gets the average of a slice of numbers
func Mean[T Number](data []T) float64 {
	if len(data) == 0 {
		return 0
	}
	var sum float64
	for _, d := range data {
		sum += float64(d)
	}
	return sum / float64(len(data))
}

// Median gets the median number in a slice of numbers
func Median[T Number](data []T) float64 {
	dataCopy := make([]T, len(data))
	copy(dataCopy, data)

	slices.Sort(dataCopy)

	var median float64
	l := len(dataCopy)
	if l == 0 {
		return 0
	} else if l%2 == 0 {
		median = Mean(dataCopy[l/2-1 : l/2+1])
	} else {
		median = float64(dataCopy[l/2])
	}

	return median
}

// Max finds the highest number in a slice
func Max[T constraints.Ordered](s []T) T {
	if len(s) == 0 {
		return *new(T)
	}
	max := s[0]
	for _, v := range s[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

// Min finds the lowest number in a set of data
func Min[T constraints.Ordered](s []T) T {
	if len(s) == 0 {
		return *new(T)
	}
	min := s[0]
	for _, v := range s[1:] {
		if v < min {
			min = v
		}
	}
	return min
}
