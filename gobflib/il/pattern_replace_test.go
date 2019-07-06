package il_test

import (
	"runtime"
	"testing"
)

// BenchmarkZeroPatternReplace tests the optimization benefits from
// from doing a zero pattern replacement.
func BenchmarkZeroPatternReplace(b *testing.B) {
	var data = make([]byte, 10)
	var datap int

	b.Run("Without", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			data[datap] = 0xFF
			for data[datap] != 0 {
				data[datap]--
			}
		}
	})

	b.Run("With", func(b *testing.B) {
		setdata := func(value byte) {
			data[datap] = value
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			data[datap] = 0xFF
			runtime.KeepAlive(data[datap])
			setdata(0)
		}
	})

	b.Run("WithNoFunc", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			data[datap] = 0xFF
			runtime.KeepAlive(data[datap])
			data[datap] = 0
		}
	})
}

// BenchmarkAddPatternReplace tests the optimization benefits from
// from doing an add pattern replacement.
func BenchmarkAddPatternReplace(b *testing.B) {
	var data = make([]byte, 10)
	var datap int

	b.Run("Without", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			datap = 0
			data[datap] = 0xFF
			data[datap+1] = 0xFF

			datap++
			for data[datap] != 0 {
				data[datap]--

				datap--
				data[datap]++
				datap++
			}
			datap--
		}
	})

	b.Run("With", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			datap = 0
			data[datap] = 0xFF
			data[datap+1] = 0xFF

			data[datap] += data[datap+1]
			data[datap+1] = 0
		}
	})
}

// BenchmarkMultiplyPatternReplace tests the optimization benefits from
// from doing an multiply pattern replacement.
func BenchmarkMultiplyPatternReplace(b *testing.B) {
	const multiplier byte = 94
	var data = make([]byte, 10)
	var datap int

	b.Run("Without", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			datap = 0
			data[datap] = 0xFF
			data[datap+1] = 0xFF

			datap++
			for data[datap] != 0 {
				data[datap]--

				datap--
				data[datap] += multiplier
				datap++
			}
			datap--
		}
	})

	b.Run("With", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			datap = 0
			data[datap] = 0xFF
			data[datap+1] = 0xFF

			data[datap] += data[datap+1] * multiplier
			data[datap+1] = 0
		}
	})
}

func TestDataSetIntOverflow(t *testing.T) {
	var adder = 2386

	var value1 byte = 0x12
	var value2 byte = 0x12

	// BF way of setting overflow value
	for i := 0; i < adder; i++ {
		value1++
	}

	// new way to multiply
	value2 += byte(adder)

	if value1 != value2 {
		t.Fatalf("Values not equal: value1=%d and value2=%d", value1, value2)
	}
}

func TestMultiplicativeAssociativity(t *testing.T) {
	var multiplier byte = 45
	var adder byte = 135

	var value1 byte = 0x12
	var value2 byte = 0x12

	// BF way of multiplying
	for i := byte(0); i < multiplier; i++ {
		value1 += adder
	}

	// new way to multiply
	value2 += adder * multiplier

	if value1 != value2 {
		t.Fatalf("Values not equal: value1=%d and value2=%d", value1, value2)
	}
}

func TestMultiplicativeDestruction(t *testing.T) {
	var multiplier byte = 0xFF
	var adder byte = 135

	var value1 byte = 0x12
	var value2 byte = 0x12

	// BF way of multiplying
	for i := byte(0); i < multiplier; i++ {
		value1 += adder
	}

	// new way to multiply
	value2 += adder * multiplier

	if value1 != value2 {
		t.Fatalf("Values not equal: value1=%d and value2=%d", value1, value2)
	}
}
