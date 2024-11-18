package transport_test

import (
	"bytes"
	"testing"

	"github.com/ninedraft/powords/internal/transport"
	"github.com/stretchr/testify/require"
)

func TestPacket(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	packet := transport.NewPacket(transport.PacketData, []byte("hello, world!"))

	n, err := packet.WriteTo(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if n != int64(buf.Len()) {
		t.Fatalf("unexpected number of bytes written: %d", n)
	}

	decoded, err := transport.ReadPacket(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	require.EqualValuesf(t, packet, decoded, "unexpected packet")
}

func TestDataChallenge(t *testing.T) {
	t.Parallel()

	data := &transport.DataChallenge{
		Time:       1,
		Memory:     2,
		KeyLen:     3,
		Difficulty: 4,
		Threads:    5,
		Salt:       transport.NewSalt(),
	}

	encoded := data.Encode()
	require.EqualValuesf(t, transport.DataChallengeSize, len(encoded), "unexpected encoded data challenge size")

	decoded := transport.DecodeDataChallenge(encoded)
	require.EqualValuesf(t, data, decoded, "unexpected data challenge")
}
