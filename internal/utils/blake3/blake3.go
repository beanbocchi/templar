package blake3

import (
	"encoding/hex"
	"io"

	"github.com/zeebo/blake3"
)

func Compute(data io.Reader) (string, error) {
	hash := blake3.New()
	if _, err := io.Copy(hash, data); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
