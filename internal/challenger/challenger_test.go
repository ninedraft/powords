package challenger_test

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/ninedraft/powords/internal/challenger"
	"github.com/ninedraft/powords/internal/transport"
	"github.com/stretchr/testify/require"
)

func TestChallenge(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	quote := testQuote("test quote")
	ch := challenger.Challenger{
		Time:       1,
		Memory:     1,
		KeyLen:     4,
		Difficulty: 2,
		Threads:    1,
		Quotes:     quote,
	}

	ctx := context.Background()

	conn := newTestConn(t)
	err := ch.Handle(ctx, conn)
	require.NoError(t, err)

	require.EqualValuesf(t, quote, conn.got, "unexpected quote")
}

type testQuote string

func (q testQuote) NextQuote(ctx context.Context) (string, error) {
	return string(q), nil
}

type testConn struct {
	t       *testing.T
	packets chan *transport.Packet
	got     string
}

func newTestConn(t *testing.T) *testConn {
	return &testConn{
		t:       t,
		packets: make(chan *transport.Packet, 1),
	}
}

func (conn *testConn) Send(ctx context.Context, packet *transport.Packet) error {
	if packet.Kind == transport.PacketData {
		conn.got = string(packet.Data)
		return nil
	}

	if packet.Kind != transport.PacketChallenge {
		panic("unexpected packet kind: " + packet.Kind.String())
	}

	challenge, err := packet.DataChallenge()
	if err != nil {
		return err
	}

	proof, n, ok := challenger.Solve(challenge, math.MaxInt64)
	require.True(conn.t, ok, "proof not found")

	conn.t.Logf("solved in %d iterations", n)

	conn.packets <- transport.NewPacket(transport.PacketProof, proof)

	return nil
}

func (conn *testConn) Receive(ctx context.Context) (*transport.Packet, error) {
	return <-conn.packets, nil
}

var errTest = errors.New("test error")

type testConnErr struct {
	err error
	pkt *transport.Packet
}

func (conn *testConnErr) Send(ctx context.Context, packet *transport.Packet) error {
	return conn.err
}

func (conn *testConnErr) Receive(ctx context.Context) (*transport.Packet, error) {
	return conn.pkt, conn.err
}

func TestChallengeError(t *testing.T) {

	ctx := context.Background()
	quote := testQuote("test quote")
	ch := challenger.Challenger{
		Time:       1,
		Memory:     1,
		KeyLen:     4,
		Difficulty: 100,
		Threads:    1,
		Quotes:     quote,
	}

	t.Run("send error", func(t *testing.T) {
		t.Parallel()

		conn := &testConnErr{
			err: errTest,
		}

		err := ch.Handle(ctx, conn)
		require.ErrorIs(t, err, errTest)
	})

	t.Run("bad solution", func(t *testing.T) {
		t.Parallel()

		conn := &testConnErr{
			pkt: transport.NewPacket(transport.PacketProof, []byte("bad solution")),
		}

		err := ch.Handle(ctx, conn)

		require.ErrorIs(t, err, challenger.ErrInvalidProof)
	})
}
