package ctrsa

import (
	"math/big"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
)

func (*nat) Generate(r *rand.Rand, size int) reflect.Value {
	limbs := make([]uint, size)
	for i := 0; i < size; i++ {
		limbs[i] = uint(r.Uint64()) & 0x7FFF_FFFF_FFFF_FFFE
	}
	return reflect.ValueOf(&nat{limbs})
}

func testModAddCommutative(a *nat, b *nat) bool {
	m := &nat{make([]uint, len(a.limbs))}
	for i := 0; i < len(m.limbs); i++ {
		m.limbs[i] = 0x7FFF_FFFF_FFFF_FFFF
	}
	aPlusB := a.clone()
	aPlusB.modAdd(b, m)
	bPlusA := b.clone()
	bPlusA.modAdd(a, m)
	return aPlusB.cmpEq(bPlusA) == 1
}

func TestModAddCommutative(t *testing.T) {
	err := quick.Check(testModAddCommutative, &quick.Config{})
	if err != nil {
		t.Error(err)
	}
}

func testModSubThenAddIdentity(a *nat, b *nat) bool {
	m := &nat{make([]uint, len(a.limbs))}
	for i := 0; i < len(m.limbs); i++ {
		m.limbs[i] = 0x7FFF_FFFF_FFFF_FFFF
	}
	original := a.clone()
	a.modSub(b, m)
	a.modAdd(b, m)
	return a.cmpEq(original) == 1
}

func TestModSubThenAddIdentity(t *testing.T) {
	err := quick.Check(testModSubThenAddIdentity, &quick.Config{})
	if err != nil {
		t.Error(err)
	}
}

func testMontgomeryRoundtrip(a *nat) bool {
	one := &nat{make([]uint, len(a.limbs))}
	one.limbs[0] = 1
	m := a.clone()
	m.add(1, one)
	monty := a.clone()
	monty.montgomeryRepresentation(m)
	aAgain := monty.clone()
	aAgain.montgomeryMul(monty, one, m, minusInverseModW(m.limbs[0]))
	return a.cmpEq(aAgain) == 1
}

func TestMontgomeryRoundtrip(t *testing.T) {
	err := quick.Check(testMontgomeryRoundtrip, &quick.Config{})
	if err != nil {
		t.Error(err)
	}
}

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

func TestExpExamples(t *testing.T) {
	m := &nat{[]uint{13}}
	m0inv := minusInverseModW(13)
	x := &nat{[]uint{3}}
	out := &nat{[]uint{0}}
	out.exp(x, []byte{12}, m, m0inv)
	expected := &nat{[]uint{1}}
	if out.cmpEq(expected) != 1 {
		t.Errorf("%+v != %+v", out, expected)
	}
}

func TestToBigExamples(t *testing.T) {
	x := &nat{[]uint{0x7FFF_FFFF_FFFF_FFFF, 0x7FFF_FFFF_FFFF_FFFF, 0b111}}
	actual := x.toBig()
	expected := new(big.Int).SetBits([]big.Word{0xFFFF_FFFF_FFFF_FFFF, 0xFFFF_FFFF_FFFF_FFFF, 0b1})
	if actual.Cmp(expected) != 0 {
		t.Errorf("%+v != %+v", actual, expected)
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
