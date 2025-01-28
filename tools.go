package toolkit

import "crypto/rand"

const (
	// randomStringSource is used to generate random strings
	// it is inlcuded in the GenerateRandomString method
	randomStringSource string = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"
)

// Tools is the type used to instantiate this module.
// Any variable of this type will have access to all methods with receiver *Tools
type Tools struct{}

// GenerateRandomString generates a random string of length n.
// The string is composed of characters from the predefined
// randomStringSource, which includes uppercase and lowercase
// letters, digits, and an underscore.
func (t *Tools) GenerateRandomString(n int) string {
	s, r := make([]rune, n), []rune(randomStringSource)
	for i := range s {
		p, _ := rand.Prime(rand.Reader, len(r))
		x, y := p.Uint64(), uint64(len(r))
		s[i] = r[x%y]
	}

	return string(s)
}
