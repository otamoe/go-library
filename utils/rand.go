package libutils

import (
	"crypto/rand"
	"math/big"
)

var (
	RandNumber = []byte("0123456789")

	RandAlpha       = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	RandAlphaNumber = []byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	RandAlphaLower       = []byte("abcdefghijklmnopqrstuvwxyz")
	RandAlphaLowerNumber = []byte("0123456789abcdefghijklmnopqrstuvwxyz")
)

func RandInt(n int) int {
	bn, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		panic(err)
	}
	return int(bn.Int64())
}

func RandInt64(n int64) int64 {
	bn, err := rand.Int(rand.Reader, big.NewInt(n))
	if err != nil {
		panic(err)
	}
	return bn.Int64()
}

func RandRune(n int, runes []rune) []rune {
	b := make([]rune, n)
	for i := range b {
		bn, err := rand.Int(rand.Reader, big.NewInt(int64(len(runes))))
		if err != nil {
			panic(err)
		}
		b[i] = runes[bn.Int64()]
	}
	return b
}

func RandByte(n int, bytes []byte) []byte {
	b := make([]byte, n)
	for i := range b {
		bn, err := rand.Int(rand.Reader, big.NewInt(int64(len(bytes))))
		if err != nil {
			panic(err)
		}
		b[i] = bytes[bn.Int64()]
	}
	return b
}
