package statistics

import (
	"math/rand"
	"reflect"
	"testing"
	"time"
)

func makeFloatSlice(c int) []float64 {
	lf := make([]float64, 0, c)
	for i := 0; i < c; i++ {
		f := float64(i * 100)
		lf = append(lf, f)
	}
	return lf
}

func makeRandFloatSlice(c int) []float64 {
	lf := make([]float64, 0, c)
	rand.Seed(time.Now().UTC().UnixNano())
	for i := 0; i < c; i++ {
		f := float64(i * 100)
		lf = append(lf, f)
	}
	return lf
}

func TestMedian(t *testing.T) {
	cases := [...]struct {
		in  []float64
		out float64
	}{
		{[]float64{5, 3, 4, 2, 1}, 3.0},
		{[]float64{6, 3, 2, 4, 5, 1}, 3.5},
		{[]float64{1}, 1.0},
	}
	for _, tst := range cases {
		if got := Median(tst.in); got != tst.out {
			t.Errorf("Median(%.1f) => %.1f != %.1f", tst.in, got, tst.out)
		}
	}
}

func BenchmarkMedianSmallFloatSlice(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Median(makeFloatSlice(5))
	}
}

func BenchmarkMedianLargeFloatSlice(b *testing.B) {
	lf := makeFloatSlice(100000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Median(lf)
	}
}

func TestMedianSortSideEffects(t *testing.T) {
	s := []float64{0.1, 0.3, 0.2, 0.4, 0.5}
	a := []float64{0.1, 0.3, 0.2, 0.4, 0.5}
	_ = Median(s)
	if !reflect.DeepEqual(s, a) {
		t.Errorf("%.1f != %.1f", s, a)
	}
}

func TestMean(t *testing.T) {
	cases := [...]struct {
		in  []float64
		out float64
	}{
		{[]float64{1, 2, 3, 4, 5}, 3.0},
		{[]float64{1, 2, 3, 4, 5, 6}, 3.5},
		{[]float64{1}, 1.0},
	}
	for _, tst := range cases {
		if got := Mean(tst.in); got != tst.out {
			t.Errorf("Mean(%.1f) => %.1f != %.1f", tst.in, got, tst.out)
		}
	}
}

func BenchmarkMeanSmallFloatSlice(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Mean(makeFloatSlice(5))
	}
}

func BenchmarkMeanLargeFloatSlice(b *testing.B) {
	lf := makeFloatSlice(100000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Mean(lf)
	}
}

func TestMin(t *testing.T) {
	cases := [...]struct {
		in  []float64
		out float64
	}{
		{[]float64{1.1, 2, 3, 4, 5}, 1.1},
		{[]float64{10.534, 3, 5, 7, 9}, 3.0},
		{[]float64{-5, 1, 5}, -5.0},
		{[]float64{5}, 5},
	}
	for _, tst := range cases {
		if got := Min(tst.in); got != tst.out {
			t.Errorf("Min(%.1f) => %.1f != %.1f", tst.in, got, tst.out)
		}
	}
}

func BenchmarkMinSmallFloatSlice(b *testing.B) {
	testData := makeFloatSlice(5)
	for i := 0; i < b.N; i++ {
		_ = Min(testData)
	}
}

func BenchmarkMinSmallRandFloatSlice(b *testing.B) {
	testData := makeRandFloatSlice(5)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Min(testData)
	}
}

func BenchmarkMinLargeFloatSlice(b *testing.B) {
	testData := makeFloatSlice(100000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Min(testData)
	}
}

func BenchmarkMinLargeRandFloatSlice(b *testing.B) {
	testData := makeRandFloatSlice(100000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Min(testData)
	}
}

func TestMax(t *testing.T) {
	cases := [...]struct {
		in  []float64
		out float64
	}{
		{[]float64{1, 2, 3, 4, 5}, 5.0},
		{[]float64{10.5, 3, 5, 7, 9}, 10.5},
		{[]float64{-20, -1, -5.5}, -1.0},
		{[]float64{-1.0}, -1.0},
	}
	for _, tst := range cases {
		if got := Max(tst.in); got != tst.out {
			t.Errorf("Max(%.1f) => %.1f != %.1f", tst.in, got, tst.out)
		}
	}
}

func BenchmarkMaxSmallFloatSlice(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Max(makeFloatSlice(5))
	}
}

func BenchmarkMaxLargeFloatSlice(b *testing.B) {
	lf := makeFloatSlice(100000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Max(lf)
	}
}
