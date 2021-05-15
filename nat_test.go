package ctrsa

import (
	"bytes"
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
	mLimbs := make([]uint, len(a.limbs))
	for i := 0; i < len(mLimbs); i++ {
		mLimbs[i] = 0x7FFF_FFFF_FFFF_FFFF
	}
	m := modulusFromNat(&nat{mLimbs})
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
	mLimbs := make([]uint, len(a.limbs))
	for i := 0; i < len(mLimbs); i++ {
		mLimbs[i] = 0x7FFF_FFFF_FFFF_FFFF
	}
	m := modulusFromNat(&nat{mLimbs})
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
	aPlusOne := a.clone()
	aPlusOne.add(1, one)
	m := modulusFromNat(aPlusOne)
	monty := a.clone()
	monty.montgomeryRepresentation(m)
	aAgain := monty.clone()
	aAgain.montgomeryMul(monty, one, m)
	return a.cmpEq(aAgain) == 1
}

func TestMontgomeryRoundtrip(t *testing.T) {
	err := quick.Check(testMontgomeryRoundtrip, &quick.Config{})
	if err != nil {
		t.Error(err)
	}
}

func testConversion(a *nat) bool {
	aBig := a.toBig()
	aFromBig := natFromBig(aBig)
	aBigAgain := aFromBig.toBig()
	return aBig.Cmp(aBigAgain) == 0
}

func TestConversion(t *testing.T) {
	err := quick.Check(testConversion, &quick.Config{})
	if err != nil {
		t.Error(err)
	}
}

func TestModSubExamples(t *testing.T) {
	m := modulusFromNat(&nat{[]uint{13}})
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
	m := modulusFromNat(&nat{[]uint{13}})
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
	m := modulusFromNat(&nat{[]uint{13}})
	x := &nat{[]uint{3}}
	out := &nat{[]uint{0}}
	out.exp(x, []byte{12}, m)
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

func TestFromBigExamples(t *testing.T) {
	theBig := new(big.Int).SetBits([]big.Word{0xFFFF_FFFF_FFFF_FFFF, 0xFFFF_FFFF_FFFF_FFFF, 0b1})
	expected := &nat{[]uint{0x7FFF_FFFF_FFFF_FFFF, 0x7FFF_FFFF_FFFF_FFFF, 0b111}}
	actual := natFromBig(theBig)
	if actual.cmpEq(expected) != 1 {
		t.Errorf("%+v != %+v", actual, expected)
	}
}

func TestFillBytes(t *testing.T) {
	x := &nat{[]uint{0x7F22_3344_5566_7788, 1}}
	xBytes := []byte{0xFF, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}
	for l := 0; l <= len(xBytes); l++ {
		actual := x.fillBytes(make([]byte, l))
		expected := xBytes[len(xBytes)-l:]
		if !bytes.Equal(actual, expected) {
			t.Errorf("%+v != %+v", actual, expected)
		}
	}
}

func TestFromBytes(t *testing.T) {
	x := &nat{[]uint{0x7F22_3344_5566_7788, 1}}
	xBytes := []byte{0xFF, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}
	actual := natFromBytes(xBytes)
	if actual.cmpEq(x) != 1 {
		t.Errorf("%+v != %+v", actual, x)
	}
}

func TestDiv(t *testing.T) {
	var hi, lo uint
	hi, lo = 0xFFFF, 0xFFFF_FFFF_FFFF_AABB
	d := uint(0xFFFF_FFFF_FFFF_FFFF)
	expectedQ, expectedR := uint(0x10000), uint(0xAABB)
	actualQ, actualR := div(hi, lo, d)
	if actualQ != expectedQ {
		t.Errorf("%+v != %+v", actualQ, expectedQ)
	}
	if actualR != expectedR {
		t.Errorf("%+v != %+v", actualR, expectedR)
	}
}

func TestShiftInExamples(t *testing.T) {
	m := modulusFromNat(&nat{[]uint{13}})
	x := &nat{[]uint{0}}
	x.shiftIn(0x7FFF_FFFF_FFFF_FFFF, m)
	expected := &nat{[]uint{7}}
	if x.cmpEq(expected) != 1 {
		t.Errorf("%+v != %+v", x, expected)
	}
	x.shiftIn(0x7FFF_FFFF_FFFF_FFFF, m)
	expected = &nat{[]uint{11}}
	if x.cmpEq(expected) != 1 {
		t.Errorf("%+v != %+v", x, expected)
	}
	m = modulusFromNat(&nat{[]uint{13, 13}})
	x = &nat{[]uint{0, 0}}
	x.shiftIn(0x7FFF_FFFF_FFFF_FFFF, m)
	expected = &nat{[]uint{0x7FFF_FFFF_FFFF_FFFF, 0}}
	if x.cmpEq(expected) != 1 {
		t.Errorf("%+v != %+v", x, expected)
	}
	x.shiftIn(0, m)
	expected = &nat{[]uint{0x8, 0x6}}
	if x.cmpEq(expected) != 1 {
		t.Errorf("%+v != %+v", x, expected)
	}
}

func makeBenchmarkModulus() *modulus {
	m := make([]uint, 32)
	for i := 0; i < 32; i++ {
		m[i] = 0x7FFF_FFFF_FFFF_FFFF
	}
	return modulusFromNat(&nat{limbs: m})
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
		out.montgomeryMul(x, y, m)
	}
}

func BenchmarkModMul(b *testing.B) {
	b.StopTimer()

	x := makeBenchmarkValue()
	y := makeBenchmarkValue()
	m := makeBenchmarkModulus()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		x.modMul(y, m)
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
		out.exp(x, e, m)
	}
}
