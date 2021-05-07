package ctrsa

import "math/bits"

const (
	_W    = bits.UintSize - 1
	_MASK = (1 << _W) - 1
)

func ctIfElse(on uint, x uint, y uint) uint {
	mask := -on
	return y ^ (mask & (y ^ x))
}

type nat struct {
	limbs []uint
}

func (x *nat) add(on uint, y *nat) {
	var c uint
	for i := 0; i < len(x.limbs); i++ {
		res := x.limbs[i] + y.limbs[i] + c
		x.limbs[i] = ctIfElse(on, res&_MASK, x.limbs[i])
		c = res >> _W
	}
}
