// Package ledger keeps a running balance for each account and writes a
// statement of what moved.
package ledger

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	maxEntries = 1 << 12
	ratePerDay = 0.0125
	separator  = '\t'
)

// ErrOverdrawn is returned when a debit would take a balance below zero.
var ErrOverdrawn = errors.New("ledger: account overdrawn")

type Kind uint8

const (
	Credit Kind = iota
	Debit
)

// An Entry is one movement on an account.
type Entry struct {
	At      time.Time `json:"at"`
	Account string    `json:"account"`
	Kind    Kind      `json:"kind"`
	Pence   int64     `json:"pence"`
	Note    string    `json:"note,omitempty"`
}

type Ledger struct {
	mu       sync.RWMutex
	balances map[string]int64
	entries  []Entry
	now      func() time.Time
}

func New() *Ledger {
	return &Ledger{balances: map[string]int64{}, now: time.Now}
}

/*
Post records e and moves the balance of its account. A debit that would

	overdraw the account is refused, and nothing is recorded.
*/
func (l *Ledger) Post(ctx context.Context, e Entry) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("post %q: %w", e.Account, err)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.entries) >= maxEntries {
		return fmt.Errorf("ledger is full at %d entries", maxEntries)
	}
	delta := e.Pence
	if e.Kind == Debit {
		delta = -delta
	}
	if l.balances[e.Account]+delta < 0 {
		return ErrOverdrawn
	}
	if e.At.IsZero() {
		e.At = l.now()
	}
	l.balances[e.Account] += delta
	l.entries = append(l.entries, e)
	return nil
}

// Balance is what an account holds, and whether it has ever been posted to.
func (l *Ledger) Balance(account string) (pence int64, ok bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	pence, ok = l.balances[account]
	return pence, ok
}

func sum[T int64 | float64](values ...T) (total T) {
	for _, v := range values {
		total += v
	}
	return total
}

// Statement writes every entry for account to w, oldest first, as lines of
// tab-separated fields.
func (l *Ledger) Statement(w io.Writer, account string) (int, error) {
	l.mu.RLock()
	mine := slices.DeleteFunc(slices.Clone(l.entries), func(e Entry) bool { return e.Account != account })
	l.mu.RUnlock()
	slices.SortStableFunc(mine, func(a, b Entry) int { return cmp.Compare(a.At.UnixNano(), b.At.UnixNano()) })

	var b strings.Builder
	for i, e := range mine {
		sign := "+"
		switch e.Kind {
		case Debit:
			sign = "-"
		case Credit:
		default:
			panic(fmt.Sprintf("unknown kind %d", e.Kind))
		}
		fmt.Fprintf(&b, "%03d%c%s%c%s%d.%02d", i+1, separator, e.At.Format(time.DateOnly), separator, sign, e.Pence/100, e.Pence%100)
		if e.Note != "" {
			b.WriteString(`	"` + e.Note + `"`)
		}
		b.WriteByte('\n')
	}
	n, err := io.WriteString(w, b.String())
	return n, err
}

// Accrue adds interest to every account in credit, a goroutine each.
func (l *Ledger) Accrue(ctx context.Context, days int) error {
	l.mu.RLock()
	accounts := make([]string, 0, len(l.balances))
	for name, pence := range l.balances {
		if pence > 0 {
			accounts = append(accounts, name)
		}
	}
	l.mu.RUnlock()

	errs := make(chan error, len(accounts))
	var wg sync.WaitGroup
	for _, name := range accounts {
		wg.Go(func() {
			pence, _ := l.Balance(name)
			interest := int64(float64(pence) * ratePerDay * float64(days))
			select {
			case <-ctx.Done():
				errs <- ctx.Err()
			default:
				errs <- l.Post(ctx, Entry{Account: name, Kind: Credit, Pence: interest, Note: "interest"})
			}
		})
	}
	wg.Wait()
	close(errs)
	var all []error
	for err := range errs {
		all = append(all, err)
	}
	return errors.Join(all...)
}
