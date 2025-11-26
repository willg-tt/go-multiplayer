package main

import "testing"

func TestCheckWinner_RowWin(t *testing.T) {
	g := &Game{
		Board: [3][3]string{
			{"X", "X", "X"},
			{"O", "O", ""},
			{"", "", ""},
		},
	}

	g.checkWinner()

	if g.Winner != "X" {
		t.Errorf("expected winner X, got %s", g.Winner)
	}
}

func TestCheckWinner_ColumnWin(t *testing.T) {
	g := &Game{
		Board: [3][3]string{
			{"O", "X", ""},
			{"O", "X", ""},
			{"", "X", ""},
		},
	}

	g.checkWinner()

	if g.Winner != "X" {
		t.Errorf("expected winner X, got %s", g.Winner)
	}
}

func TestCheckWinner_DiagonalWin(t *testing.T) {
	g := &Game{
		Board: [3][3]string{
			{"O", "", "X"},
			{"", "X", ""},
			{"X", "", "O"},
		},
	}

	g.checkWinner()

	if g.Winner != "X" {
		t.Errorf("expected winner X, got %s", g.Winner)
	}
}

func TestCheckWinner_Draw(t *testing.T) {
	g := &Game{
		Board: [3][3]string{
			{"X", "O", "X"},
			{"X", "O", "O"},
			{"O", "X", "X"},
		},
	}

	g.checkWinner()

	if g.Winner != "draw" {
		t.Errorf("expected draw, got %s", g.Winner)
	}
}

func TestCheckWinner_NoWinnerYet(t *testing.T) {
	g := &Game{
		Board: [3][3]string{
			{"X", "O", ""},
			{"", "X", ""},
			{"", "", ""},
		},
	}

	g.checkWinner()

	if g.Winner != "" {
		t.Errorf("expected no winner yet, got %s", g.Winner)
	}
}

func TestReset(t *testing.T) {
	g := &Game{
		Board: [3][3]string{
			{"X", "O", "X"},
			{"X", "O", "O"},
			{"O", "X", "X"},
		},
		Turn:   "O",
		Winner: "draw",
	}

	g.reset()

	if g.Turn != "X" {
		t.Errorf("expected turn X, got %s", g.Turn)
	}
	if g.Winner != "" {
		t.Errorf("expected no winner, got %s", g.Winner)
	}
	if g.Board[0][0] != "" {
		t.Errorf("expected empty board, got %v", g.Board)
	}
}
