package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"

	"github.com/ninedraft/powords/internal/transport"
	"golang.org/x/net/netutil"
	_ "golang.org/x/net/netutil"
	"golang.org/x/sync/errgroup"
)

type Handle func(ctx context.Context, conn *transport.Conn) error

type Server struct {
	addr     string
	limit    int
	listener net.Listener
	handler  Handle
}

func New(addr string, limit int, handle Handle) *Server {
	return &Server{
		addr:    addr,
		limit:   limit,
		handler: handle,
	}
}

func (server *Server) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	listener, err := net.Listen("tcp", server.addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	context.AfterFunc(ctx, func() {
		_ = listener.Close()
	})

	if server.limit > 0 {
		listener = netutil.LimitListener(listener, server.limit)
	}

	wg := &errgroup.Group{}
	wg.SetLimit(server.limit)

	var errHandle error
	for {
		conn, err := listener.Accept()
		if err != nil {
			errHandle = fmt.Errorf("accepting connection: %w", err)
			break
		}

		wg.Go(func() error {
			defer conn.Close()

			err := server.handler(ctx, transport.NewConn(conn))
			if err != nil {
				slog.ErrorContext(ctx, "handling connection",
					"error", err)
			}

			return nil
		})
	}

	if err := wg.Wait(); err != nil {
		errHandle = errors.Join(errHandle, fmt.Errorf("handling connections: %w", err))
	}

	return errHandle
}
