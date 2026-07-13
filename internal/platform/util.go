package platform

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"
)

func MD5(value string) string { sum := md5.Sum([]byte(value)); return hex.EncodeToString(sum[:]) }
func Random(n int) string {
	const chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	out := make([]byte, n)
	for i := range out {
		v, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			out[i] = '0'
		} else {
			out[i] = chars[v.Int64()]
		}
	}
	return string(out)
}
func GenSN(prefix string) string {
	const digits = "0123456789"
	suffix := make([]byte, 8)
	for i := range suffix {
		v, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			suffix[i] = '0'
		} else {
			suffix[i] = digits[v.Int64()]
		}
	}
	return fmt.Sprintf("%s%s%s", prefix, time.Now().Format("20060102150405"), suffix)
}
func FenToYuan(v int64) float64 { return float64(v) / 100 }
