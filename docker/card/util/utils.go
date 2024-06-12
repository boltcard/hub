package util

import (
	"crypto/rand"
	"encoding/hex"
)

func Check(e error) {
	if e != nil {
		panic(e)
	}
}

func ConvertPaymentHash(paymentHash string) []int {
	rHashByteSlice, err := hex.DecodeString(paymentHash)
	Check(err)

	rHashIntSlice := []int{}
	for _, rHashByte := range rHashByteSlice {
		rHashIntSlice = append(rHashIntSlice, int(rHashByte))
	}

	return rHashIntSlice
}

func Random_hex() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	Check(err)

	return hex.EncodeToString(b)
}

func Max(a, b int) int {
	var max int
	if a > b {
		max = a
	} else {
		max = b
	}
	return max
}
