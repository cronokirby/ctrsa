package ctrsa

import (
	"math/big"
	"math/bits"
)

const (
	// The number of bits we use for our limbs
	_W = bits.UintSize - 1
	// A mask to select only those bits from a full machine word
	_MASK = (1 << _W) - 1
)

// choice represents a constant-time boolean
//
// The value of choice is always either 1 or 0.
//
// We use a separate type instead of bool, in order to be able to make decisions without leaking
// which decision was made.
type choice uint

// ctIfElse returns x if on == 1, and y if on == 0
//
// This leaks no information about which branch was chosen.
//
// If on is any value besides 1 or 0, the result is undefined.
func ctIfElse(on choice, x, y uint) uint {
	mask := -uint(on)
	return y ^ (mask & (y ^ x))
}

// ctEq compares two uint values for equality
//
// This works with any two uint values, not just those that fit over _W bits
func ctEq(x, y uint) choice {
	// If x == y, then x ^ y should be all zero bits.
	q := x ^ y
	// For any q != 0, either the MSB of q, or the MSB of -q is 1.
	// We can thus or those together, and check the top bit. When q is zero,
	// that means that x and y are equal, so we negate that top bit.
	return 1 ^ choice((q|-q)>>(bits.UintSize-1))
}

// ctGeq calculates x >= y
//
// This works with any two uint values, not just those that fit over _W bits
func ctGeq(x, y uint) choice {
	// If subtracting y from x overflows, then x >= y cannot be true
	_, carry := bits.Sub(x, y, 0)
	return 1 ^ choice(carry)
}

// div calculates the (hi:lo / d, hi:lo % d)
//
// All of the inputs are over the full size of uint.
func div(hi, lo, d uint) (uint, uint) {
	var quo uint
	hi = ctIfElse(ctEq(hi, d), 0, hi)
	for i := bits.UintSize - 1; i > 0; i-- {
		j := bits.UintSize - i
		w := (hi << j) | (lo >> i)
		sel := ctGeq(w, d) | choice(hi>>i)
		hi2 := (w - d) >> j
		lo2 := lo - (d << i)
		hi = ctIfElse(sel, hi2, hi)
		lo = ctIfElse(sel, lo2, lo)
		quo |= uint(sel)
		quo <<= 1
	}
	sel := ctGeq(lo, d) | choice(hi)
	rem := ctIfElse(sel, lo-d, lo)
	quo |= uint(sel)
	return quo, rem
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

func (x *nat) expand(size int) *nat {
	if cap(x.limbs) < size {
		newLimbs := make([]uint, size)
		copy(newLimbs, x.limbs)
		x.limbs = newLimbs
	} else {
		x.limbs = x.limbs[:size]
	}
	return x
}

func (x *nat) clone() *nat {
	out := &nat{make([]uint, len(x.limbs))}
	copy(out.limbs, x.limbs)
	return out
}

func (x *nat) toBig() *big.Int {
	bitSize := len(x.limbs) * _W
	requiredLimbs := (bitSize + bits.UintSize - 1) / bits.UintSize

	out := make([]big.Word, requiredLimbs)
	shift := big.Word(0)
	outI := 0
	for i := 0; i < len(x.limbs); i++ {
		xi := big.Word(x.limbs[i])
		out[outI] |= xi << shift
		topShift := bits.UintSize - shift
		if topShift <= _W {
			outI++
			out[outI] = xi >> topShift
		}
		shift = (shift + _W) % bits.UintSize
	}

	return new(big.Int).SetBits(out)
}

func natFromBig(x *big.Int) *nat {
	xLimbs := x.Bits()
	bitSize := len(xLimbs) * bits.UintSize
	requiredLimbs := (bitSize + _W - 1) / _W

	out := &nat{make([]uint, requiredLimbs)}
	// shift is < _W
	shift := uint(0)
	outI := 0
	for i := 0; i < len(xLimbs); i++ {
		xi := uint(xLimbs[i])
		out.limbs[outI] |= (xi << shift) & _MASK
		outI++
		out.limbs[outI] = (xi >> (_W - shift))
		shift++
		if shift >= _W {
			shift -= _W
			outI++
		}
	}
	return out
}

// fillBytes writes out this number as big endian bytes to a buffer
//
// If the bytes are not large enough to contain the number, the output is truncated,
// keeping the least significant bytes that do fit.
func (x *nat) fillBytes(bytes []byte) []byte {
	outI := len(bytes) - 1
	fittingLimbs := len(bytes) * 8 / _W
	var shift uint
	for _, limb := range x.limbs[:fittingLimbs] {
		// The number of bits to consume from this limb
		remainingBits := uint(_W)
		if shift > 0 {
			bytes[outI] |= byte(limb) << shift
			outI--
			consumed := 8 - shift
			limb >>= consumed
			remainingBits -= consumed
		}
		// The number of bytes we'll fill completely
		fullBytes := int(remainingBits >> 3)
		// The shift for the next round becomes what's left
		shift = remainingBits & 0b111
		for i := 0; i < fullBytes; i++ {
			bytes[outI] = byte(limb)
			outI--
			limb >>= 8
		}
		bytes[outI] = byte(limb)
	}
	// If all of the limbs fit in the bytes, we have nothing left to do
	if fittingLimbs >= len(x.limbs) {
		return bytes
	}
	// Becuase of how we calculated fittingLimbs, only the last remaining limb
	// has any potential bits to contribute
	lastLimb := x.limbs[fittingLimbs]
	if shift > 0 {
		bytes[outI] |= byte(lastLimb) << shift
		outI--
		lastLimb >>= 8 - shift
	}
	for outI >= 0 {
		bytes[outI] = byte(lastLimb)
		outI--
		lastLimb >>= 8
	}
	return bytes
}

// natFromBytes converts a slice of big endian bytes into a nat
func natFromBytes(bytes []byte) *nat {
	bits := len(bytes) * 8
	requiredLimbs := (bits + _W - 1) / _W
	out := &nat{make([]uint, requiredLimbs)}
	outI := 0
	shift := 0
	for i := len(bytes) - 1; i >= 0; i-- {
		bi := bytes[i]
		out.limbs[outI] |= uint(bi) << shift
		shift += 8
		if shift >= _W {
			shift -= _W
			out.limbs[outI] &= _MASK
			outI++
			out.limbs[outI] = uint(bi) >> (8 - shift)
		}
	}
	return out
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

func (x *nat) mulSub(q uint, m *nat) (cc uint) {
	for i := 0; i < len(x.limbs) && i < len(m.limbs); i++ {
		hi, lo := bits.Mul(q, m.limbs[i])
		lo, cc = bits.Add(lo, cc, 0)
		hi += cc
		cc = (hi << 1) | (lo >> _W)
		res := x.limbs[i] - (lo & _MASK)
		cc += res >> _W
		x.limbs[i] = res & _MASK
	}
	return
}

// modulus is used for modular arithmetic, precomputed useful constants
type modulus struct {
	// The underlying natural number for this modulus.
	//
	// This will be stored without any padding.
	//
	// The contract here is that this shouldn't alias with any other natural number being used.
	nat *nat
	// The number of leading zeros in the modulus
	leading uint
	// -nat.limbs[0]^-1 mod _W
	m0inv uint
}

func minusInverseModW(x uint) uint {
	y := x
	// This is enough for 63 bits, and the extra iteration is not that costly for 31
	for i := 0; i < 5; i++ {
		y = (y * (2 - x*y)) & _MASK
	}
	return (1 << _W) - y
}

// modulusFromNat creates a new modulus from a nat
//
// The nat should not be zero, and the number of significant bits in the number should be
// leakable.
func modulusFromNat(nat *nat) *modulus {
	var m modulus
	m.nat = nat
	var size uint
	for size = uint(len(m.nat.limbs)); size > 0 && m.nat.limbs[size-1] == 0; size-- {
	}
	m.nat.limbs = m.nat.limbs[:size]
	m.leading = uint(bits.LeadingZeros(m.nat.limbs[size-1]) - 1)
	m.m0inv = minusInverseModW(m.nat.limbs[0])
	return &m
}

// shiftIn calculates x = x << _W + y mod m
//
// This assumes that x is already reduced mod m.
func (x *nat) shiftIn(y uint, m *modulus) {
	size := len(m.nat.limbs)
	if size == 0 {
		return
	}
	if size == 1 {
		// In this case, x:y % m is exactly what we need to calculate
		// div expects fully saturated limbs, so we have a bit of manipulation to do here
		_, r := div(x.limbs[0]>>1, (x.limbs[0]<<_W)|y, m.nat.limbs[0])
		x.limbs[0] = r
		return
	}

	// The idea is as follows:
	//
	// We want to shift y into x, and then divide by m. Instead of dividing by
	// m, we can get a good estimate, using the top two 2 * _W bits of x, and the
	// top _W bits of m. These are stored in a1:a0, and b0 respectively.

	// We need to keep around the top limb of x, pre-shifts
	hi := x.limbs[size-1]
	a1 := ((hi << m.leading) | (x.limbs[size-2] >> (_W - m.leading))) & _MASK
	// The actual shift can be performed by moving the limbs of x up, then inserting y
	for i := size - 1; i > 0; i-- {
		x.limbs[i] = x.limbs[i-1]
	}
	x.limbs[0] = y
	a0 := ((x.limbs[size-1] << m.leading) | (x.limbs[size-2] >> (_W - m.leading))) & _MASK
	b0 := ((m.nat.limbs[size-1] << m.leading) | (m.nat.limbs[size-2] >> (_W - m.leading))) & _MASK

	// We want to use a1:a0 / b0 - 1 as our estimate. If rawQ is 0, we should
	// use 0 as our estimate. Another edge case when an overflow happens in the quotient.
	// It can be shown that this happens when a1 == b0. In this case, we want
	// to use the maximum value for q
	rawQ, _ := div(a1>>1, (a1<<_W)|a0, b0)
	q := ctIfElse(ctEq(a1, b0), _MASK, ctIfElse(ctEq(rawQ, 0), 0, rawQ-1))
	// This estimate is off by +- 1, so we subtract q * m, and then either add
	// or subtract m, based on the result.
	cc := x.mulSub(q, m.nat)
	// If the carry from subtraction is greater than the limb of x we've shifted out,
	// then we've underflowed, and need to add in m
	under := 1 ^ ctGeq(hi, cc)
	// For us to be too large, we first need to not be too low, as per the previous flag.
	// Then, if the lower limbs of x are still larger, or the top limb of x is equal to the carry,
	// we can conclude that we're too large, and need to subtract m
	stillBigger := x.cmpGeq(m.nat)
	over := (1 ^ under) & (stillBigger | (1 ^ ctEq(cc, hi)))
	x.add(under, m.nat)
	x.sub(over, m.nat)
}

// mod calculates out = x mod m
//
// This works regardless how large the value of x is
func (out *nat) mod(x *nat, m *modulus) {
	out.expand(len(m.nat.limbs))
	for i := 0; i < len(out.limbs); i++ {
		out.limbs[i] = 0
	}
	i := len(x.limbs) - 1
	// We can inject at least N - 1 limbs while staying under m
	// Thus, we start injecting from index N - 2
	start := len(m.nat.limbs) - 2
	// That is, if there are at least that many limbs to choose from
	if i < start {
		start = i
	}
	for j := start; j >= 0; j-- {
		out.limbs[j] = x.limbs[i]
		i--
	}
	// We shift in the remaining limbs, making sure to reduce modulo M each time
	for ; i >= 0; i-- {
		out.shiftIn(x.limbs[i], m)
	}
}

// expandFor makes sure that out has the right size to work with operations modulo m
//
// This assumes that out is already reduced modulo m, but may not be properly sized. Since
// modular operations assume that operands are exactly the right size, this allows us
// to expand a natural number to meet this expectation.
func (out *nat) expandFor(m *modulus) {
	out.expand(len(m.nat.limbs))
}

// modSub computes x = (x - y) % m
//
// The length of both operands must be the same as the modulus.
//
// Both operands must already be reduced modulo m.
func (x *nat) modSub(y *nat, m *modulus) {
	underflow := x.sub(1, y)
	x.add(choice(underflow), m.nat)
}

// modAdd computes x = (x + y) % m
//
// The length of both operands must be the same as the modulus.
//
// Both operands must already be reduced modulo m.
func (x *nat) modAdd(y *nat, m *modulus) {
	overflow := x.add(1, y)
	// If x < m, then subtraction will underflow
	underflow := 1 ^ x.cmpGeq(m.nat)
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
	x.sub(needSubtraction, m.nat)
}

// montgomeryRepresentation calculates x = xR % m, with R := _W^n, and n = len(m)
//
// Montgomery multiplication replaces standard modular multiplication for numbers
// in this representation. This speeds up the multiplication operation in this case.
func (x *nat) montgomeryRepresentation(m *modulus) {
	for i := 0; i < len(m.nat.limbs); i++ {
		x.shiftIn(0, m)
	}
}

// montgomeryMul calculates out = xy / R % m, with R := _W^n, and n = len(m)
//
// This is faster than your standard modular multiplication.
//
// All inputs should be the same length, and not alias eachother.
func (out *nat) montgomeryMul(x *nat, y *nat, m *modulus) {
	for i := 0; i < len(out.limbs); i++ {
		out.limbs[i] = 0
	}

	overflow := uint(0)
	// The different loops are over the same size, but we use different conditions
	// to try and make the compiler elide bounds checking.
	for i := 0; i < len(x.limbs); i++ {
		f := ((out.limbs[0] + x.limbs[i]*y.limbs[0]) * m.m0inv) & _MASK
		// Carry fits on 64 bits
		var carry uint
		for j := 0; j < len(y.limbs) && j < len(m.nat.limbs) && j < len(out.limbs); j++ {
			hi, lo := bits.Mul(x.limbs[i], y.limbs[j])
			z_lo, c := bits.Add(out.limbs[j], lo, 0)
			z_hi, _ := bits.Add(0, hi, c)
			hi, lo = bits.Mul(f, m.nat.limbs[j])
			z_lo, c = bits.Add(z_lo, lo, 0)
			z_hi, _ = bits.Add(z_hi, hi, c)
			z_lo, c = bits.Add(z_lo, carry, 0)
			z_hi, _ = bits.Add(z_hi, 0, c)
			if j > 0 {
				out.limbs[j-1] = z_lo & _MASK
			}
			carry = (z_lo >> _W) | (z_hi << 1)
		}
		z := overflow + carry
		out.limbs[len(out.limbs)-1] = z & _MASK
		overflow = z >> _W
	}
	underflow := 1 ^ out.cmpGeq(m.nat)
	// See modAdd
	needSubtraction := ctEq(overflow, uint(underflow))
	out.sub(needSubtraction, m.nat)
}

// modMul calculates x *= y mod m
func (x *nat) modMul(y *nat, m *modulus) {
	xMonty := x.clone()
	xMonty.montgomeryRepresentation(m)
	x.montgomeryMul(xMonty, y, m)
}

func (out *nat) exp(x *nat, e []byte, m *modulus) {
	size := len(m.nat.limbs)
	out.expand(size)

	xs := make([]*nat, 15)
	xs[0] = x.clone()
	xs[0].montgomeryRepresentation(m)
	for i := 1; i < len(xs); i++ {
		xs[i] = &nat{make([]uint, size)}
		xs[i].montgomeryMul(xs[i-1], xs[0], m)
	}

	selectedX := &nat{make([]uint, size)}
	for i := 0; i < len(out.limbs); i++ {
		out.limbs[i] = 0
	}
	out.limbs[0] = 1
	out.montgomeryRepresentation(m)
	scratch := &nat{make([]uint, size)}
	for _, b := range e {
		for j := 4; j >= 0; j -= 4 {
			scratch.montgomeryMul(out, out, m)
			out.montgomeryMul(scratch, scratch, m)
			scratch.montgomeryMul(out, out, m)
			out.montgomeryMul(scratch, scratch, m)

			window := uint((b >> j) & 0b1111)
			for i := 0; i < len(xs); i++ {
				selectedX.assign(ctEq(window, uint(i+1)), xs[i])
			}
			scratch.montgomeryMul(out, selectedX, m)
			out.assign(1^ctEq(window, 0), scratch)
		}
	}
	for i := 0; i < len(scratch.limbs); i++ {
		scratch.limbs[i] = 0
	}
	scratch.limbs[0] = 1
	outC := out.clone()
	out.montgomeryMul(outC, scratch, m)
}
