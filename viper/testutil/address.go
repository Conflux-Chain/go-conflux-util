package testutil

import (
	"encoding/hex"
	"fmt"
	"strings"
)

type HexAddress [20]byte

func ParseHexAddress(hexStr string) (HexAddress, error) {
	hexStr = strings.TrimPrefix(hexStr, "0x")

	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return HexAddress{}, err
	}

	if len(bytes) != 20 {
		return HexAddress{}, fmt.Errorf("invalid address length %v", len(bytes))
	}

	var address HexAddress

	copy(address[:], bytes)

	return address, nil
}

func MustParseHexAddress(hexStr string) HexAddress {
	address, err := ParseHexAddress(hexStr)
	if err != nil {
		panic(err)
	}

	return address
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (a *HexAddress) UnmarshalText(text []byte) error {
	address, err := ParseHexAddress(string(text))
	if err != nil {
		return err
	}

	*a = address

	return nil
}
