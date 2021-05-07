package ctrsa

import "math/bits"

const (
	// The number of bits we use for our limbs
	_W = bits.UintSize - 1
	// A mask to select only those bits from a full machine word
	_MASK = (1 << _W) - 1
)

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

// ctIfElse returns x if on == 1, and y if on == 0
//
// This leaks no information about which branch was chosen.
//
// If on is any value besides 1 or 0, the result is undefined.
func ctIfElse(on choice, x uint, y uint) uint {
	mask := -uint(on)
	return y ^ (mask & (y ^ x))
}

// add returns x += y, if on == 1, and otherwise does nothing
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

// sub returns x -= y, if on == 1, and otherwise does nothing
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
