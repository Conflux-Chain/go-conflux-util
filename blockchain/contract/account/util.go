package account

import "math/big"

// SafeBigFillBytes fills the destination byte slice with the big-endian representation of the given big.Int.
// If the number is nil or the destination slice is empty, it does nothing.
// This function ensures that the destination slice is always filled with the correct number of bytes, truncating or padding as necessary.
func SafeBigFillBytes(num *big.Int, dest []byte) {
	if num == nil || len(dest) == 0 {
		return
	}

	src := num.Bytes()

	if len(src) <= len(dest) {
		num.FillBytes(dest)
	} else {
		copy(dest, src[len(src)-len(dest):])
	}
}
