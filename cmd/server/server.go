package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"unsafe"

	"github.com/ninedraft/powords/internal/challenger"
	"github.com/ninedraft/powords/internal/quotes"
	"github.com/ninedraft/powords/internal/server"
	"github.com/ninedraft/powords/internal/transport"
	"golang.org/x/exp/constraints"
)

func main() {
	cfg := config{
		Addr:       cmp.Or(os.Getenv("ADDR"), "localhost:2939"),
		MaxConns:   cmp.Or(int(envUint[uint32]("MAX_CONNS", 1)), runtime.NumCPU()),
		Time:       envUint[uint32]("POW_TIME", 1),
		Memory:     envUint[uint32]("POW_MEM", 1),
		KeyLen:     envUint[uint32]("POW_KEY_LEN", 1),
		Difficulty: envUint[uint32]("POW_DIFFICULTY", 1),
		Threads:    envUint[uint8]("POW_THREADS", 1),
	}

	slog.Info("config",
		"values", cfg)

	ch := challenger.Challenger{
		Time:       cfg.Time,
		Memory:     cfg.Memory,
		KeyLen:     cfg.KeyLen,
		Difficulty: cfg.Difficulty,
		Threads:    cfg.Threads,
		Quotes:     quotes.Quotes{},
	}

	srv := server.New(cfg.Addr, runtime.NumCPU(),
		func(ctx context.Context, conn *transport.Conn) error {
			return ch.Handle(ctx, conn)
		})

	ctx := context.Background()

	if err := srv.Run(ctx); err != nil {
		panic("serving: " + err.Error())
	}
}

type config struct {
	Addr     string
	MaxConns int

	Time       uint32
	Memory     uint32
	KeyLen     uint32
	Difficulty uint32
	Threads    uint8
}

func (cfg config) validate() error {
	var err error

	if cfg.Time <= 0 {
		err = errors.Join(err,
			fmt.Errorf("invalid time: %d, must be >=0", cfg.Time))
	}

	if cfg.Memory <= 0 {
		err = errors.Join(err,
			fmt.Errorf("invalid memory: %d, must be >=0", cfg.Memory))
	}

	if cfg.KeyLen <= 0 {
		err = errors.Join(err,
			fmt.Errorf("invalid key length: %d, must be >=0", cfg.KeyLen))
	}

	if cfg.Difficulty <= 0 {
		err = errors.Join(err,
			fmt.Errorf("invalid difficulty: %d, must be >=0", cfg.Difficulty))
	}

	if cfg.Threads <= 0 {
		err = errors.Join(err,
			fmt.Errorf("invalid threads: %d, must be >=0", cfg.Threads))
	}

	return err
}

func envUint[I constraints.Integer](key string, def I) I {
	val, ok := os.LookupEnv(key)
	if !ok {
		return def

	}

	i, err := strconv.ParseInt(val, 10, int(unsafe.Sizeof(def)/8))
	if err != nil {
		panic("parsing " + key + ": " + err.Error())
	}

	return I(i)
}
