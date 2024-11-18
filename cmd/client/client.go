package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"time"

	"github.com/ninedraft/powords/internal/client"
)

func main() {
	addr := "localhost:2939"
	flag.StringVar(&addr, "addr", addr, "server address")

	timeout := time.Minute
	flag.DurationVar(&timeout, "timeout", timeout, "timeout for waiting server responses")

	maxIterations := math.MaxInt64
	flag.IntVar(&maxIterations, "max-iterations", maxIterations, "max iterations for solving challenge")

	flag.Parse()

	cl := client.Client{
		Addr:          addr,
		MaxIterations: maxIterations,
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	quote, err := cl.GetQuote(ctx)
	if err != nil {
		panic("getting quote: " + err.Error())
	}

	fmt.Println(string(quote))
}
