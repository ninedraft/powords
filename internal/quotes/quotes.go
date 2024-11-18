package quotes

import (
	"context"
	_ "embed"
	"math/rand/v2"
	"strings"
	"sync"
)

//go:embed quotes.txt
var quotesData string

var quotes = sync.OnceValue(func() []string {
	return strings.Split(quotesData, "\n")
})

type Quotes struct{}

func (Quotes) NextQuote(ctx context.Context) (string, error) {
	q := quotes()

	return q[rand.N(len(q))], nil
}
