// Package board implements the pure game-rule logic for DigDog.
// It has no dependency on Ebiten, image, or color packages so it can
// be tested without a graphics environment.
package board

import "math/rand"

const missTicks = 45 // how long a mismatched pair stays face up (60 ticks = 1 s)

// CardState describes how a card is currently presented on the board.
type CardState int

const (
	Buried   CardState = iota
	Revealed           // face up, waiting for its match
	Matched            // permanently face up / won
)

// Result tells the game what just happened after a card was dug up.
type Result int

const (
	ResultFirst Result = iota // first card of a pair
	ResultMatch
	ResultMiss
	ResultWin
)

// Card is a single cell on the board.
type Card struct {
	Face  int
	State CardState
}

// Board holds the complete mutable state of one game.
type Board struct {
	cols, rows int
	cards      []Card

	first        int // index of the first card dug up this turn, or -1
	missA, missB int // the pair waiting to be buried again after a miss
	missTimer    int
	locked       bool // true while the dog is busy or a miss is showing

	Moves int
	Won   bool
}

// NewBoardFromFaces constructs a Board from an already-decided face sequence.
// Panics if len(faces) != cols*rows.
func NewBoardFromFaces(cols, rows int, faces []int) *Board {
	if len(faces) != cols*rows {
		panic("len(faces) must equal cols*rows")
	}
	cards := make([]Card, len(faces))
	for i, f := range faces {
		cards[i] = Card{Face: f}
	}
	return &Board{cols: cols, rows: rows, cards: cards, first: -1}
}

// NewBoard shuffles a fresh deck and returns a ready-to-play board.
func NewBoard(cols, rows int) *Board {
	n := cols * rows
	if n%2 != 0 {
		panic("board needs an even number of cells")
	}
	faces := make([]int, 0, n)
	for i := 0; i < n/2; i++ {
		faces = append(faces, i, i)
	}
	rand.Shuffle(len(faces), func(i, j int) { faces[i], faces[j] = faces[j], faces[i] })
	return NewBoardFromFaces(cols, rows, faces)
}

// Cols returns the number of columns.
func (b *Board) Cols() int { return b.cols }

// Rows returns the number of rows.
func (b *Board) Rows() int { return b.rows }

// Len returns the total number of cells.
func (b *Board) Len() int { return len(b.cards) }

// Card returns the card at index i.
func (b *Board) Card(i int) Card { return b.cards[i] }

// Pick reserves a buried card for digging. Returns false if the pick is not allowed.
func (b *Board) Pick(i int) bool {
	if b.locked || i < 0 || i >= len(b.cards) || b.cards[i].State != Buried {
		return false
	}
	b.locked = true
	return true
}

// Reveal flips a picked card face up and returns what that means for the game.
func (b *Board) Reveal(i int) Result {
	b.cards[i].State = Revealed

	if b.first < 0 {
		b.first = i
		b.locked = false
		return ResultFirst
	}

	a := b.first
	b.first = -1
	b.Moves++

	if b.cards[a].Face == b.cards[i].Face {
		b.cards[a].State, b.cards[i].State = Matched, Matched
		if b.allMatched() {
			b.Won = true // stays locked
			return ResultWin
		}
		b.locked = false
		return ResultMatch
	}

	b.missA, b.missB, b.missTimer = a, i, missTicks // stays locked until Update re-buries them
	return ResultMiss
}

// Update counts down the miss timer and buries a mismatched pair again.
func (b *Board) Update() {
	if b.missTimer == 0 {
		return
	}
	b.missTimer--
	if b.missTimer == 0 {
		b.cards[b.missA].State = Buried
		b.cards[b.missB].State = Buried
		b.locked = false
	}
}

func (b *Board) allMatched() bool {
	for _, c := range b.cards {
		if c.State != Matched {
			return false
		}
	}
	return true
}
