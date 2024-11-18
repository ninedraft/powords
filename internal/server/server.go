package server

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/ninedraft/powords/internal/transport"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/net/netutil"
	_ "golang.org/x/net/netutil"
	"golang.org/x/sync/errgroup"
)

var (
	activeConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "active_connections",
		Help: "The current number of active connections",
	})
	totalConnections = promauto.NewCounter(prometheus.CounterOpts{
		Name: "total_connections",
		Help: "The total number of handled connections",
	})
	connectionHandlingDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "connection_handling_duration_seconds",
		Help:    "The duration of handling connections",
		Buckets: prometheus.DefBuckets,
	})
	successfulConnections = promauto.NewCounter(prometheus.CounterOpts{
		Name: "successful_connections",
		Help: "The total number of successfully handled connections",
	})
	failedConnections = promauto.NewCounter(prometheus.CounterOpts{
		Name: "failed_connections",
		Help: "The total number of failed connection handlings",
	})
)

type Handle func(ctx context.Context, conn *transport.Conn) error

type Server struct {
	Addr string

	// Max concurrent connections
	Limit int

	// Max connection handling time
	Timeout time.Duration

	Handler Handle
}

const (
	defaultTimeout = 10 * time.Second
	defaultAddr    = "localhost:2939"
)

func (server *Server) Run(ctx context.Context) error {
	handleTimeout := cmp.Or(server.Timeout, defaultTimeout)

	listener, err := net.Listen("tcp", cmp.Or(server.Addr, defaultAddr))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	context.AfterFunc(ctx, func() {
		_ = listener.Close()
	})

	if server.Limit > 0 {
		listener = netutil.LimitListener(listener, server.Limit)
	}

	wg := &errgroup.Group{}
	wg.SetLimit(server.Limit)

	var errHandle error
	for {
		conn, err := listener.Accept()
		if err != nil {
			errHandle = fmt.Errorf("accepting connection: %w", err)
			break
		}

		activeConnections.Inc()
		totalConnections.Inc()

		slog.InfoContext(ctx, "new connection",
			"from", conn.RemoteAddr())

		wg.Go(func() error {
			defer conn.Close()
			defer activeConnections.Dec()
			defer slog.InfoContext(ctx, "connection closed",
				"from", conn.RemoteAddr())

			start := time.Now()
			defer func() {
				duration := time.Since(start).Seconds()
				connectionHandlingDuration.Observe(duration)
			}()

			ctx, cancel := context.WithTimeout(ctx, handleTimeout)
			defer cancel()

			conn.SetDeadline(time.Now().Add(handleTimeout))

			err := server.Handler(ctx, transport.NewConn(conn))
			if err != nil {
				slog.ErrorContext(ctx, "handling connection",
					"error", err)
				failedConnections.Inc()
			} else {
				successfulConnections.Inc()
			}

			return nil
		})
	}

	if err := wg.Wait(); err != nil {
		errHandle = errors.Join(errHandle, fmt.Errorf("handling connections: %w", err))
	}

	return errHandle
}
