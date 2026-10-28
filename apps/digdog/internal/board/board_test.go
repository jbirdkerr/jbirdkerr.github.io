package board

import "testing"

// newTestBoard is a 2x2 board with faces [0,1,0,1]:
//
//	cell 0 → face 0,  cell 1 → face 1
//	cell 2 → face 0,  cell 3 → face 1
func newTestBoard() *Board {
	return NewBoardFromFaces(2, 2, []int{0, 1, 0, 1})
}

// TestFirstPick verifies that the first card of a turn returns ResultFirst
// and unlocks the board so the next pick is accepted.
func TestFirstPick(t *testing.T) {
	b := newTestBoard()

	if !b.Pick(0) {
		t.Fatal("Pick(0) returned false; expected true")
	}
	r := b.Reveal(0)
	if r != ResultFirst {
		t.Fatalf("Reveal after first pick: got %v, want ResultFirst", r)
	}
	// Board should be unlocked so a second pick is possible.
	if !b.Pick(2) {
		t.Fatal("Pick(2) after ResultFirst returned false; board should be unlocked")
	}
}

// TestMatchingPair verifies that revealing a matching pair returns ResultMatch
// and marks both cards as Matched.
func TestMatchingPair(t *testing.T) {
	b := newTestBoard()

	b.Pick(0)
	b.Reveal(0) // first card (face 0)
	b.Pick(2)
	r := b.Reveal(2) // second card, same face → match

	if r != ResultMatch {
		t.Fatalf("got %v, want ResultMatch", r)
	}
	if b.Card(0).State != Matched {
		t.Errorf("card 0 state = %v, want Matched", b.Card(0).State)
	}
	if b.Card(2).State != Matched {
		t.Errorf("card 2 state = %v, want Matched", b.Card(2).State)
	}
}

// TestMismatchedPair verifies ResultMiss behaviour: the board stays locked,
// and after missTicks calls to Update both cards are Buried and the board is unlocked.
func TestMismatchedPair(t *testing.T) {
	b := newTestBoard()

	b.Pick(0)
	b.Reveal(0) // face 0
	b.Pick(1)
	r := b.Reveal(1) // face 1 → mismatch

	if r != ResultMiss {
		t.Fatalf("got %v, want ResultMiss", r)
	}
	// Board must remain locked immediately after a miss.
	if b.Pick(2) {
		t.Error("Pick succeeded while board is locked after miss")
	}

	// Tick down the miss timer.
	for i := 0; i < missTicks; i++ {
		b.Update()
	}

	if b.Card(0).State != Buried {
		t.Errorf("card 0 state = %v after miss timeout, want Buried", b.Card(0).State)
	}
	if b.Card(1).State != Buried {
		t.Errorf("card 1 state = %v after miss timeout, want Buried", b.Card(1).State)
	}
	// Board must be unlocked now.
	if !b.Pick(0) {
		t.Error("board still locked after miss timeout")
	}
}

// TestPickRestrictions verifies that Pick returns false for an out-of-range
// index, an already-revealed card, and when the board is locked.
func TestPickRestrictions(t *testing.T) {
	b := newTestBoard()

	// Out-of-range index.
	if b.Pick(-1) {
		t.Error("Pick(-1) should return false")
	}
	if b.Pick(99) {
		t.Error("Pick(99) should return false (out of range)")
	}

	// Reveal card 0 (first of a pair) — board unlocks after ResultFirst.
	b.Pick(0)
	b.Reveal(0)

	// Card 0 is now Revealed; picking it again must fail.
	if b.Pick(0) {
		t.Error("Pick of already-Revealed card should return false")
	}

	// Lock the board via a second pick.
	b.Pick(2)
	// While animating / locked (after Pick but before Reveal), another pick fails.
	if b.Pick(3) {
		t.Error("Pick should return false while board is locked")
	}
}

// TestWin verifies that matching the last remaining pair returns ResultWin
// and sets Board.Won.
func TestWin(t *testing.T) {
	b := newTestBoard()

	// Match pair 0 (cells 0 & 2, face 0).
	b.Pick(0)
	b.Reveal(0)
	b.Pick(2)
	b.Reveal(2)

	// Match pair 1 (cells 1 & 3, face 1) — this is the last pair.
	b.Pick(1)
	b.Reveal(1)
	b.Pick(3)
	r := b.Reveal(3)

	if r != ResultWin {
		t.Fatalf("got %v, want ResultWin", r)
	}
	if !b.Won {
		t.Error("Board.Won should be true after the last match")
	}
}
