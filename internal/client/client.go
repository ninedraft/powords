package client

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"net"

	"github.com/ninedraft/powords/internal/challenger"
	"github.com/ninedraft/powords/internal/transport"
)

type Client struct {
	Addr          string
	MaxIterations int
	Dialer        *net.Dialer
}

const (
	defaultAddr          = "localhost:2939"
	defaultMaxIterations = math.MaxInt64
)

var defaultDialer = &net.Dialer{}

func (client *Client) GetQuote(ctx context.Context) (string, error) {
	dialer := client.Dialer
	if dialer == nil {
		dialer = defaultDialer
	}
	addr := cmp.Or(client.Addr, defaultAddr)

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return "", fmt.Errorf("dialing: %w", err)
	}
	defer conn.Close()

	tc := transport.NewConn(conn)

	pkt, err := tc.Receive(ctx)
	if err != nil {
		return "", fmt.Errorf("reading challenge packet: %w", err)
	}

	if pkt.Kind != transport.PacketChallenge {
		return "", fmt.Errorf("unexpected packet kind: %s", pkt.Kind)
	}

	challenge, err := pkt.DataChallenge()
	if err != nil {
		return "", fmt.Errorf("decoding challenge: %w", err)
	}

	solution, _, ok := challenger.Solve(challenge, cmp.Or(client.MaxIterations, defaultMaxIterations))
	if !ok {
		return "", fmt.Errorf("no solution found")
	}

	pkt = transport.NewPacket(transport.PacketProof, solution)
	if err := tc.Send(ctx, pkt); err != nil {
		return "", fmt.Errorf("sending proof: %w", err)
	}

	pkt, err = tc.Receive(ctx)
	if err != nil {
		return "", fmt.Errorf("reading quote packet: %w", err)
	}

	if pkt.Kind != transport.PacketData {
		return "", fmt.Errorf("unexpected packet kind: %s", pkt.Kind)
	}

	return string(pkt.Data), nil
}
