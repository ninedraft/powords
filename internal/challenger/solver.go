package challenger

import (
	"crypto/rand"

	"github.com/ninedraft/powords/internal/transport"
	"golang.org/x/crypto/argon2"
)

func Solve(challenge *transport.DataChallenge, maxIterations int) ([]byte, int, bool) {
	passwd := make([]byte, 16)
	for i := 1; i < maxIterations; i++ {
		rand.Read(passwd)

		got := argon2.IDKey(
			passwd,
			challenge.Salt[:],
			challenge.Time,
			challenge.Memory,
			challenge.Threads,
			challenge.KeyLen,
		)

		if trailingZeros(got) >= int(challenge.Difficulty) {
			return passwd, i, true
		}
	}

	return nil, maxIterations, false
}
