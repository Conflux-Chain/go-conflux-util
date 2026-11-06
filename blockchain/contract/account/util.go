package account

import "math/big"

// SafeBigFillBytes fills dest with the big-endian representation of num's absolute value.
// If the number is nil or the destination slice is empty, it does nothing.
// Short values are padded with leading zeros; oversized values retain only the
// least significant len(dest) bytes. It does not validate the sign or range.
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
