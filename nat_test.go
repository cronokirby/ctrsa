package ctrsa

import (
	"math/big"
	"testing"
)

func TestModSubExamples(t *testing.T) {
	m := &nat{[]uint{13}}
	x := &nat{[]uint{6}}
	y := &nat{[]uint{7}}
	x.modSub(y, m)
	expected := &nat{[]uint{12}}
	if x.cmpEq(expected) != 1 {
		t.Errorf("%+v != %+v", x, expected)
	}
	x.modSub(y, m)
	expected = &nat{[]uint{5}}
	if x.cmpEq(expected) != 1 {
		t.Errorf("%+v != %+v", x, expected)
	}
}

func TestModAddExamples(t *testing.T) {
	m := &nat{[]uint{13}}
	x := &nat{[]uint{6}}
	y := &nat{[]uint{7}}
	x.modAdd(y, m)
	expected := &nat{[]uint{0}}
	if x.cmpEq(expected) != 1 {
		t.Errorf("%+v != %+v", x, expected)
	}
	x.modAdd(y, m)
	expected = &nat{[]uint{7}}
	if x.cmpEq(expected) != 1 {
		t.Errorf("%+v != %+v", x, expected)
	}
}

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

func makeBenchmarkExponent() []byte {
	e := make([]byte, 256)
	for i := 0; i < 32; i++ {
		e[i] = 0xFF
	}
	return e
}

func BenchmarkModAdd(b *testing.B) {
	b.StopTimer()

	x := makeBenchmarkValue()
	y := makeBenchmarkValue()
	m := makeBenchmarkModulus()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		x.modAdd(y, m)
	}
}

func BenchmarkModSub(b *testing.B) {
	b.StopTimer()

	x := makeBenchmarkValue()
	y := makeBenchmarkValue()
	m := makeBenchmarkModulus()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		x.modSub(y, m)
	}
}

func BenchmarkMontgomeryRepr(b *testing.B) {
	b.StopTimer()

	x := makeBenchmarkValue()
	m := makeBenchmarkModulus()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		x.montgomeryRepresentation(m)
	}
}

func BenchmarkMontgomeryMul(b *testing.B) {
	b.StopTimer()

	x := makeBenchmarkValue()
	y := makeBenchmarkValue()
	out := makeBenchmarkValue()
	m := makeBenchmarkModulus()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		out.montgomeryMul(x, y, m, 0xABCD)
	}
}

func BenchmarkExpBig(b *testing.B) {
	b.StopTimer()

	out := new(big.Int)
	exponentBytes := makeBenchmarkExponent()
	x := new(big.Int).SetBytes(exponentBytes)
	e := new(big.Int).SetBytes(exponentBytes)
	n := new(big.Int).SetBytes(exponentBytes)
	one := new(big.Int).SetUint64(1)
	n.Add(n, one)

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		out.Exp(x, e, n)
	}
}

func BenchmarkExp(b *testing.B) {
	b.StopTimer()

	x := makeBenchmarkValue()
	e := makeBenchmarkExponent()
	out := makeBenchmarkValue()
	m := makeBenchmarkModulus()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		out.exp(x, e, m, m.limbs[0])
	}
}
