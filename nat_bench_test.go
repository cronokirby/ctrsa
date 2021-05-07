package ctrsa

import "testing"

func makeBenchmarkModulus() *nat {
	m := make([]uint, 32)
	for i := 0; i < 32; i++ {
		m[i] = 0x7FFF_FFFF_FFFF_FFFF
	}
	return &nat{limbs: m}
}

func makeBenchmarkValue() *nat {
	x := make([]uint, 32)
	for i := 0; i < 32; i++ {
		x[i] = 0x7FFF_FFFF_FFFF_FFFA
	}
	return &nat{limbs: x}
}

func BenchmarkModAdd(b *testing.B) {
	b.StopTimer()

	x := makeBenchmarkValue()
	y := makeBenchmarkValue()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		x.add(1, y)
	}
}

func BenchmarkModSub(b *testing.B) {
	b.StopTimer()

	x := makeBenchmarkValue()
	y := makeBenchmarkValue()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		x.sub(1, y)
	}
}
