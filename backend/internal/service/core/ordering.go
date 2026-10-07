package core

import (
	"context"

	"github.com/google/uuid"
)

// TempOrderBase is far above any real order number. While items are being
// renumbered they wait on these numbers, because two items of the same
// parent cannot hold one number even for a moment.
const TempOrderBase = 1_000_000

// MoveTo returns ids with id placed at the 1-based position. A position past
// the end puts it last. id may already be in the list or be new.
func MoveTo(ids []uuid.UUID, id uuid.UUID, position int) []uuid.UUID {
	rest := make([]uuid.UUID, 0, len(ids)+1)

	for _, other := range ids {
		if other != id {
			rest = append(rest, other)
		}
	}

	index := position - 1
	if index < 0 {
		index = 0
	}

	if index > len(rest) {
		index = len(rest)
	}

	result := make([]uuid.UUID, 0, len(rest)+1)
	result = append(result, rest[:index]...)
	result = append(result, id)

	return append(result, rest[index:]...)
}

// Renumber gives the items the numbers 1..n in the order of ids. It runs in two
// steps (temporary numbers first), so it never breaks the rule that one number
// belongs to one item. Call it inside a transaction.
func Renumber(ctx context.Context, ids []uuid.UUID, set func(ctx context.Context, id uuid.UUID, number int) error) error {
	for i, id := range ids {
		if err := set(ctx, id, TempOrderBase+i); err != nil {
			return err
		}
	}

	for i, id := range ids {
		if err := set(ctx, id, i+1); err != nil {
			return err
		}
	}

	return nil
}
