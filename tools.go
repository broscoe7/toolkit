package toolkit

import "crypto/rand"

const randomStringSource = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_+"

// Tools is the type used to instantiate this module. Any variable
// of this type will have access to all the methods with the receiver *Tools.
type Tools struct{}

// RandomString returns a string of random characters of length n,
// drawing characters from randomStringSource.
func (t *Tools) RandomString(n int) string {
	if n <= 0 {
		return ""
	}

	buf := make([]byte, n)

	if _, err := rand.Read(buf); err != nil {
		return ""
	}

	for i := range buf {
		buf[i] = randomStringSource[buf[i]&63]
	}

	return string(buf)
}
