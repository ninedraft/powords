package challenger

import (
	"context"
	"errors"
	"fmt"
	"math/bits"

	"github.com/ninedraft/powords/internal/transport"
	"golang.org/x/crypto/argon2"
)

type Conn interface {
	Send(ctx context.Context, packet *transport.Packet) error
	Receive(ctx context.Context) (*transport.Packet, error)
}

type Quotes interface {
	NextQuote(ctx context.Context) (string, error)
}

type Challenger struct {
	Time, Memory uint32
	Threads      uint8
	KeyLen       uint32
	Difficulty   uint32

	Quotes Quotes
}

var ErrInvalidProof = errors.New("invalid proof")

func (challenger *Challenger) Handle(ctx context.Context, conn Conn) error {
	salt := transport.NewSalt()

	data := (&transport.DataChallenge{
		Time:       challenger.Time,
		Memory:     challenger.Memory,
		KeyLen:     challenger.KeyLen,
		Salt:       salt,
		Threads:    challenger.Threads,
		Difficulty: challenger.Difficulty,
	}).Encode()

	err := conn.Send(ctx, transport.NewPacket(transport.PacketChallenge, data))
	if err != nil {
		return fmt.Errorf("sending challenge: %w", err)
	}

	response, err := conn.Receive(ctx)
	if err != nil {
		return fmt.Errorf("receiving response: %w", err)
	}

	if response.Kind != transport.PacketProof {
		return fmt.Errorf("unexpected packet kind %s, want %s", response.Kind, transport.PacketProof)
	}

	hash := argon2.IDKey(
		response.Data,
		salt[:],
		challenger.Time,
		challenger.Memory,
		challenger.Threads,
		challenger.KeyLen,
	)

	if trailingZeros(hash) < int(challenger.Difficulty) {
		return ErrInvalidProof
	}

	quote, err := challenger.Quotes.NextQuote(ctx)
	if err != nil {
		return fmt.Errorf("getting next quote: %w", err)
	}

	err = conn.Send(ctx, transport.NewPacket(transport.PacketData, []byte(quote)))
	if err != nil {
		return fmt.Errorf("sending quote: %w", err)
	}

	return nil
}

func trailingZeros(data []byte) int {
	n := 0
	for _, b := range data {
		n += bits.TrailingZeros8(b)
		if b != 0 {
			break
		}
	}
	return n
}
