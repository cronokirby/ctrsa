package ctrsa

import (
	"math/bits"
)

const (
	// The number of bits we use for our limbs
	_W = bits.UintSize - 1
	// A mask to select only those bits from a full machine word
	_MASK = (1 << _W) - 1
)

func minusInverseModW(x uint) uint {
	y := x
	// This is enough for 63 bits, and the extra iteration is not that costly for 31
	for i := 0; i < 5; i++ {
		y = (y * (2 - x*y)) & _MASK
	}
	return (1 << _W) - y
}

// nat represents an arbitrary natural number
type nat struct {
	// We represent a natural number in base 2^W with W = bits.UintSize - 1.
	// The reason for leaving the top bit of each number unset is mainly
	// for Montgomery multiplication, in the inner loop of exponentiation.
	// Using fully saturated limbs would leave us working with 129 bit numbers,
	// wasting a lot of space, and thus time.
	//
	// The reason we use uint, instead of uint64 directly, is for potential portability,
	// but mainly to be able to call `bits.Mul` and `bits.Add` directly, making our
	// code more concise.
	limbs []uint
}

// choice represents a constant-time condition
//
// The value of choice is always either 1, or 0
type choice uint

// ctEq compares two uint values for equality
//
// This function requires that both x and y fit over _W bits.
func ctEq(x, y uint) choice {
	// If x == y, then x ^ y should be all zero bits. We then underflow
	// when subtracting 1, so the top bit is set. Otherwise, the top
	// will remain unset, giving us 0.
	return choice(((x ^ y) - 1) >> _W)
}

// cmpEq compares two natural numbers for equality
//
// Both operands should have the same length.
func (x *nat) cmpEq(y *nat) choice {
	equal := choice(1)
	for i := 0; i < len(x.limbs) && i < len(y.limbs); i++ {
		equal &= ctEq(x.limbs[i], y.limbs[i])
	}
	return equal
}

// cmpGeq calculates x >= y, returning 1 if this holds, and 0 otherwise
func (x *nat) cmpGeq(y *nat) choice {
	var c uint
	for i := 0; i < len(x.limbs) && i < len(y.limbs); i++ {
		c = (x.limbs[i] - y.limbs[i] - c) >> _W
	}
	// If there was a carry, then subtracting y underflowed, so
	// x is not greater than or equal to y
	return 1 ^ choice(c)
}

// ctIfElse returns x if on == 1, and y if on == 0
//
// This leaks no information about which branch was chosen.
//
// If on is any value besides 1 or 0, the result is undefined.
func ctIfElse(on choice, x, y uint) uint {
	mask := -uint(on)
	return y ^ (mask & (y ^ x))
}

func (x *nat) clone() *nat {
	out := &nat{make([]uint, len(x.limbs))}
	copy(out.limbs, x.limbs)
	return out
}

func (x *nat) assign(on choice, y *nat) {
	for i := 0; i < len(x.limbs) && i < len(y.limbs); i++ {
		x.limbs[i] = ctIfElse(on, y.limbs[i], x.limbs[i])
	}
}

// add comptues x += y, if on == 1, and otherwise does nothing
//
// The length of both operands must be the same.
//
// The length of the operands is the only information leaked.
func (x *nat) add(on choice, y *nat) (c uint) {
	for i := 0; i < len(x.limbs) && i < len(y.limbs); i++ {
		res := x.limbs[i] + y.limbs[i] + c
		x.limbs[i] = ctIfElse(on, res&_MASK, x.limbs[i])
		c = res >> _W
	}
	return
}

// sub computes x -= y, if on == 1, and otherwise does nothing
//
// The length of both operands must be the same.
//
// The length of the operands is the only information leaked.
func (x *nat) sub(on choice, y *nat) (c uint) {
	for i := 0; i < len(x.limbs) && i < len(y.limbs); i++ {
		res := x.limbs[i] - y.limbs[i] - c
		x.limbs[i] = ctIfElse(on, res&_MASK, x.limbs[i])
		c = res >> _W
	}
	return
}

// modSub computes x = (x - y) % m
//
// The length of both operands must be the same as the modulus.
//
// Both operands must already be reduced modulo m.
func (x *nat) modSub(y *nat, m *nat) {
	underflow := x.sub(1, y)
	x.add(choice(underflow), m)
}

// modAdd computes x = (x + y) % m
//
// The length of both operands must be the same as the modulus.
//
// Both operands must already be reduced modulo m.
func (x *nat) modAdd(y *nat, m *nat) {
	overflow := x.add(1, y)
	// If x < m, then subtraction will underflow
	underflow := 1 ^ x.cmpGeq(m)
	// Three cases are possible:
	//
	// overflow = 0, underflow = 0
	//   In this case, addition fits on our limbs, but we can still subtract away m
	//   without an underflow, so we need to perform the subtraction to reduce our result.
	// overflow = 0, underflow = 1
	//   Our addition fits on our limbs, but we can't subtract m without underflowing.
	//   Our result is already reduced
	// overflow = 1, underflow = 1
	//   Our addition does not fit on our limbs, and we only underflowed because we're not able
	//   to take away this extra carry bit. We need to subtract m to reduce our result.
	//
	// The other case is not possible, because x and y are at most m - 1, so their addition
	// is at most 2m - 2, and subtracting m once is sufficient to reduce this value. To
	// see overflow = 1, and underflow = 0, we would need a value where subtracting m more than
	// once is necessary, which cannot happen.
	needSubtraction := ctEq(overflow, uint(underflow))
	x.sub(needSubtraction, m)
}

// montgomeryRepresentation calculates x = xR % m, with R := _W^n, and n = len(m)
//
// Montgomery multiplication replaces standard modular multiplication for numbers
// in this representation. This speeds up the multiplication operation in this case.
func (x *nat) montgomeryRepresentation(m *nat) {
	// This is a pretty slow way of calculating a representation.
	// The advantage of doing it with this method is that it doesn't require adding
	// any extra code. It's also not that bad compared to the cost of exponentiation
	for i := 0; i < len(m.limbs)*_W; i++ {
		x.modAdd(x, m)
	}
}

// montgomeryMul calculates out = xy / R % m, with R := _W^n, and n = len(m)
//
// This is faster than your standard modular multiplication.
//
// All inputs should be the same length, and not alias eachother.
func (out *nat) montgomeryMul(x *nat, y *nat, m *nat, m0inv uint) {
	for i := 0; i < len(out.limbs); i++ {
		out.limbs[i] = 0
	}

	overflow := uint(0)
	// The different loops are over the same size, but we use different conditions
	// to try and make the compiler elide bounds checking.
	for i := 0; i < len(x.limbs); i++ {
		f := ((out.limbs[0] + x.limbs[i]*y.limbs[0]) * m0inv) & _MASK
		// Carry fits on 64 bits
		var carry uint
		for j := 0; j < len(y.limbs) && j < len(m.limbs) && j < len(out.limbs); j++ {
			hi, lo := bits.Mul(x.limbs[i], y.limbs[j])
			z_lo, c := bits.Add(out.limbs[j], lo, 0)
			z_hi, _ := bits.Add(0, hi, c)
			hi, lo = bits.Mul(f, m.limbs[j])
			z_lo, c = bits.Add(z_lo, lo, 0)
			z_hi, _ = bits.Add(z_hi, hi, c)
			z_lo, c = bits.Add(z_lo, carry, 0)
			z_hi, _ = bits.Add(z_hi, 0, c)
			if j > 0 {
				out.limbs[j-1] = z_lo & _MASK
			}
			carry = (z_lo >> _W) | (z_hi << 1)
		}
		z, _ := bits.Add(overflow, carry, 0)
		out.limbs[len(out.limbs)-1] = z & _MASK
		overflow = z >> _W
	}
	underflow := 1 ^ out.cmpGeq(m)
	// See modAdd
	needSubtraction := ctEq(overflow, uint(underflow))
	out.sub(needSubtraction, m)
}

func (out *nat) exp(x *nat, e []byte, m *nat, m0inv uint) {
	xSquared := x.clone()
	xSquared.montgomeryRepresentation(m)
	for i := 0; i < len(out.limbs); i++ {
		out.limbs[i] = 0
	}
	out.limbs[0] = 1
	scratch := &nat{make([]uint, len(m.limbs))}
	for i := len(e) - 1; i >= 0; i-- {
		b := e[i]
		for j := 0; j < 8; j++ {
			selectMultiply := choice(b & 1)
			scratch.montgomeryMul(out, xSquared, m, m0inv)
			out.assign(selectMultiply, scratch)
			scratch.montgomeryMul(xSquared, xSquared, m, m0inv)
			tmp := scratch
			scratch = xSquared
			xSquared = tmp
			b >>= 1
		}
	}
}
