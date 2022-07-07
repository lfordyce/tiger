package domain

import (
	"math"
	"sort"
	"time"
)

type statsError struct {
	err string
}

func (s statsError) Error() string {
	return s.err
}

func (s statsError) String() string {
	return s.err
}

var (
	// EmptyInputErr Input must not be empty
	EmptyInputErr = statsError{"Input must not be empty."}
)

// A Sample is a single measurement.
type Sample struct {
	WorkerID   int
	Elapsed    float64
	Overhead   time.Duration
	HostnameID string
	StartEnd   time.Time
	EndTime    time.Time
}

type SampleByHostname struct {
	HostnameID string
	Elapsed    []float64
	Overhead   []float64
}

// Float64Data is a named type for []float64 with helper methods
type Float64Data []float64

// Get item in slice
func (f Float64Data) Get(i int) float64 { return f[i] }

// Len returns length of slice
func (f Float64Data) Len() int { return len(f) }

// Less returns if one number is less than another
func (f Float64Data) Less(i, j int) bool { return f[i] < f[j] }

// Swap switches out two numbers in slice
func (f Float64Data) Swap(i, j int) { f[i], f[j] = f[j], f[i] }

// Min returns the minimum number in the data
func (f Float64Data) Min() (float64, error) { return Min(f) }

// Max returns the maximum number in the data
func (f Float64Data) Max() (float64, error) { return Max(f) }

// Sum returns the total of all the numbers in the data
func (f Float64Data) Sum() (float64, error) { return Sum(f) }

// Sum adds all the numbers of a slice together
func Sum(input Float64Data) (sum float64, err error) {

	if input.Len() == 0 {
		return math.NaN(), EmptyInputErr
	}

	// Add em up
	for _, n := range input {
		sum += n
	}

	return sum, nil
}

// Mean gets the average of a slice of numbers
func Mean(input Float64Data) (float64, error) {

	if input.Len() == 0 {
		return math.NaN(), EmptyInputErr
	}

	sum, _ := input.Sum()

	return sum / float64(input.Len()), nil
}

// Median gets the median number in a slice of numbers
func Median(input Float64Data) (median float64, err error) {

	// Start by sorting a copy of the slice
	c := sortedCopy(input)

	// No math is needed if there are no numbers
	// For even numbers we add the two middle numbers
	// and divide by two using the mean function above
	// For odd numbers we just use the middle number
	l := len(c)
	if l == 0 {
		return math.NaN(), EmptyInputErr
	} else if l%2 == 0 {
		median, _ = Mean(c[l/2-1 : l/2+1])
	} else {
		median = c[l/2]
	}

	return median, nil
}

// Min finds the lowest number in a set of data
func Min(input Float64Data) (min float64, err error) {

	// Get the count of numbers in the slice
	l := input.Len()

	// Return an error if there are no numbers
	if l == 0 {
		return math.NaN(), EmptyInputErr
	}

	// Get the first value as the starting point
	min = input.Get(0)

	// Iterate until done checking for a lower value
	for i := 1; i < l; i++ {
		if input.Get(i) < min {
			min = input.Get(i)
		}
	}
	return min, nil
}

// Max finds the highest number in a slice
func Max(input Float64Data) (max float64, err error) {

	// Return an error if there are no numbers
	if input.Len() == 0 {
		return math.NaN(), EmptyInputErr
	}

	// Get the first value as the starting point
	max = input.Get(0)

	// Loop and replace higher values
	for i := 1; i < input.Len(); i++ {
		if input.Get(i) > max {
			max = input.Get(i)
		}
	}

	return max, nil
}

// copyslice copies a slice of float64s
func copyslice(input Float64Data) Float64Data {
	s := make(Float64Data, input.Len())
	copy(s, input)
	return s
}

// sortedCopy returns a sorted copy of float64s
func sortedCopy(input Float64Data) (copy Float64Data) {
	copy = copyslice(input)
	sort.Float64s(copy)
	return
}
