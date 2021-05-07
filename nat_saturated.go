package ctrsa

import "math/bits"

type Word uint64

type triple struct {
	w0 Word
	w1 Word
	w2 Word
}

func (a *triple) add(b triple) {
	w0, c0 := bits.Add(uint(a.w0), uint(b.w0), 0)
	w1, c1 := bits.Add(uint(a.w1), uint(b.w1), c0)
	w2, _ := bits.Add(uint(a.w2), uint(b.w2), c1)
	a.w0 = Word(w0)
	a.w1 = Word(w1)
	a.w2 = Word(w2)
}

func tripleFromMul(a Word, b Word) triple {
	// You might be tempted to use mulWW here, but for some reason, Go cannot
	// figure out how to inline that assembly routine, but using bits.Mul directly
	// gets inlined by the compiler into effectively the same assembly.
	//
	// Beats me.
	w1, w0 := bits.Mul(uint(a), uint(b))
	return triple{w0: Word(w0), w1: Word(w1), w2: 0}
}

func montgomeryMul(out []Word, x []Word, y []Word, m []Word, m0inv Word) {
	size := len(m)

	for i := 0; i < size; i++ {
		out[i] = 0
	}
	dh := Word(0)
	for i := 0; i < size; i++ {
		f := (out[0] + x[i]*y[0]) * m0inv
		var c triple
		for j := 0; j < size; j++ {
			z := triple{w0: out[j], w1: 0, w2: 0}
			z.add(tripleFromMul(x[i], y[j]))
			z.add(tripleFromMul(f, m[j]))
			z.add(c)
			if j > 0 {
				out[j-1] = z.w0
			}
			c.w0 = z.w1
			c.w1 = z.w2
		}
		z := triple{w0: dh, w1: 0, w2: 0}
		z.add(c)
		out[size-1] = z.w0
		dh = z.w1
	}
}
