package crypto

import "crypto/rand"

func getRandomBytes(l int64) ([]byte, error) {
	buf := make([]byte, l)
	_, err := rand.Reader.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf, nil
}
