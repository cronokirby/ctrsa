package ctrsa

import "testing"

func BenchmarkMontgomeryMul(b *testing.B) {
	b.StopTimer()

	x := make([]Word, 32)
	y := make([]Word, 32)
	m := make([]Word, 32)
	for i := 0; i < 32; i++ {
		x[i] = 0xFFFF_FFFF_FFFF_FFAA
		y[i] = 0xFFFF_FFFF_FFFF_FFAA
		m[i] = 0xFFFF_FFFF_FFFF_FFFF
	}
	out := make([]Word, 32)

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		montgomeryMul(out, x, y, m, 0xFFFF_FFFF_FFFF_FFFF)
	}
}
