package ctrsa

import "math/bits"

const (
	MASK = (1 << 63) - 1
)

func montgomeryMul2(out []Word, x []Word, y []Word, m []Word, m0inv Word) {
	size := len(m)

	for i := 0; i < size; i++ {
		out[i] = 0
	}
	dh := Word(0)
	var z_lo, z_hi, c_lo, c_hi Word
	for i := 0; i < size; i++ {
		f := ((out[0] + x[i]*y[0]) * m0inv) & MASK
		for j := 0; j < size; j++ {
			z_lo = out[j]
			hi, lo := bits.Mul(uint(x[i]), uint(y[j]))
			res, c := bits.Add(uint(z_lo), lo, 0)
			z_lo = Word(res)
			res, _ = bits.Add(hi, 0, c)
			z_hi = Word(res)
			hi, lo = bits.Mul(uint(f), uint(m[j]))
			res, c = bits.Add(uint(z_lo), lo, 0)
			z_lo = Word(res)
			res, _ = bits.Add(uint(z_hi), hi, c)
			z_hi = Word(res)
			res, c = bits.Add(uint(z_lo), uint(c_lo), 0)
			z_lo = Word(res)
			res, _ = bits.Add(uint(z_hi), uint(c_hi), c)
			z_hi = Word(res)
			if j > 0 {
				out[j-1] = z_lo
			}
			c_lo = z_hi
			c_hi = z_hi >> 62
		}
		z_lo = dh + c_lo
		z_hi = Word(z_lo >> 63)
		z_lo &= MASK
		z_hi = (z_hi + c_hi) & MASK

		out[size-1] = z_lo
		dh = z_hi
	}
}
