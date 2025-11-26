package main

import "github.com/gorilla/websocket"

// Game represents the tic-tac-toe game state
type Game struct {
	Board   [3][3]string `json:"board"`  // "", "X", or "O"
	Turn    string       `json:"turn"`   // "X" or "O"
	Winner  string       `json:"winner"` // "", "X", "O", or "draw"
	PlayerX *Player      `json:"-"`      // - means don't include in JSON
	PlayerO *Player      `json:"-"`
}

// Player represents a connected player
type Player struct {
	Conn *websocket.Conn
	Mark string // "X" or "O"
}

// ClientMessage is what the browser sends to us
type ClientMessage struct {
	Type    string `json:"type"`    // "move", "chat", "reset", "setName"
	X       int    `json:"x"`       // 0, 1, or 2
	Y       int    `json:"y"`       // 0, 1, or 2
	Message string `json:"message"` // Chat message text
	Name    string `json:"name"`    // Display name
}

// ServerMessage is what we send to the browser
type ServerMessage struct {
	Type    string `json:"type"`              // "state", "error", "assigned", "chat"
	Game    *Game  `json:"game,omitempty"`    // Current game state
	Mark    string `json:"mark,omitempty"`    // "X", "O", or "spectator"
	Error   string `json:"error,omitempty"`
	From    string `json:"from,omitempty"`    // Role: "X", "O", "spectator", "system"
	Name    string `json:"name,omitempty"`    // Display name (optional)
	Message string `json:"message,omitempty"` // Chat message text
}

// Global game state - only touched by game manager goroutine, no mutex needed!
var game = &Game{Turn: "X"}

// checkWinner checks if someone won or if it's a draw
func (g *Game) checkWinner() {
	// Check rows
	for y := 0; y < 3; y++ {
		if g.Board[y][0] != "" && g.Board[y][0] == g.Board[y][1] && g.Board[y][1] == g.Board[y][2] {
			g.Winner = g.Board[y][0]
			return
		}
	}

	// Check columns
	for x := 0; x < 3; x++ {
		if g.Board[0][x] != "" && g.Board[0][x] == g.Board[1][x] && g.Board[1][x] == g.Board[2][x] {
			g.Winner = g.Board[0][x]
			return
		}
	}

	// Check diagonals
	if g.Board[0][0] != "" && g.Board[0][0] == g.Board[1][1] && g.Board[1][1] == g.Board[2][2] {
		g.Winner = g.Board[0][0]
		return
	}
	if g.Board[0][2] != "" && g.Board[0][2] == g.Board[1][1] && g.Board[1][1] == g.Board[2][0] {
		g.Winner = g.Board[0][2]
		return
	}

	// Check for draw (all cells filled, no winner)
	for y := 0; y < 3; y++ {
		for x := 0; x < 3; x++ {
			if g.Board[y][x] == "" {
				return // Empty cell found, game not over
			}
		}
	}
	g.Winner = "draw"
}
