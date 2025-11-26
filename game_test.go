package main

import "testing"

func TestCheckWinner_XEliminated(t *testing.T) {
	g := newGame()
	g.UnitX.HP = 0

	g.checkWinner()

	if g.Winner != "O" {
		t.Errorf("expected winner O, got %s", g.Winner)
	}
}

func TestCheckWinner_OEliminated(t *testing.T) {
	g := newGame()
	g.UnitO.HP = 0

	g.checkWinner()

	if g.Winner != "X" {
		t.Errorf("expected winner X, got %s", g.Winner)
	}
}

func TestCheckWinner_NoWinnerYet(t *testing.T) {
	g := newGame()

	g.checkWinner()

	if g.Winner != "" {
		t.Errorf("expected no winner, got %s", g.Winner)
	}
}

func TestInitializeUnits(t *testing.T) {
	g := newGame()

	// X should spawn at bottom-left (0, 8)
	if g.UnitX.X != 0 || g.UnitX.Y != BoardSize-1 {
		t.Errorf("X unit at wrong position: got (%d, %d), expected (0, %d)", g.UnitX.X, g.UnitX.Y, BoardSize-1)
	}

	// O should spawn at top-right (8, 0)
	if g.UnitO.X != BoardSize-1 || g.UnitO.Y != 0 {
		t.Errorf("O unit at wrong position: got (%d, %d), expected (%d, 0)", g.UnitO.X, g.UnitO.Y, BoardSize-1)
	}

	// Both should start with MaxHP
	if g.UnitX.HP != MaxHP {
		t.Errorf("X unit should have %d HP, got %d", MaxHP, g.UnitX.HP)
	}
	if g.UnitO.HP != MaxHP {
		t.Errorf("O unit should have %d HP, got %d", MaxHP, g.UnitO.HP)
	}
	if g.UnitX.MaxHP != MaxHP {
		t.Errorf("X unit MaxHP should be %d, got %d", MaxHP, g.UnitX.MaxHP)
	}

	// Board should have units placed
	if g.Board[g.UnitX.Y][g.UnitX.X] != "X" {
		t.Errorf("X not on board at its position")
	}
	if g.Board[g.UnitO.Y][g.UnitO.X] != "O" {
		t.Errorf("O not on board at its position")
	}
}

func TestNewGame(t *testing.T) {
	g := newGame()

	if g.Turn != "X" {
		t.Errorf("expected X to start, got %s", g.Turn)
	}
	if g.Winner != "" {
		t.Errorf("expected no winner at start, got %s", g.Winner)
	}
	if g.UnitX == nil || g.UnitO == nil {
		t.Error("units should be initialized")
	}
}

func TestAbs(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{5, 5},
		{-5, 5},
		{0, 0},
		{-1, 1},
	}

	for _, test := range tests {
		result := abs(test.input)
		if result != test.expected {
			t.Errorf("abs(%d) = %d, expected %d", test.input, result, test.expected)
		}
	}
}
