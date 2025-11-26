package main

import (
	"sync"

	"github.com/gorilla/websocket"
)

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
	ID   string
	Conn *websocket.Conn
	Mark string // "X" or "O"
}

// ClientMessage is what the browser sends to us
type ClientMessage struct {
	Type string `json:"type"` // "move"
	X    int    `json:"x"`    // 0, 1, or 2
	Y    int    `json:"y"`    // 0, 1, or 2
}

// ServerMessage is what we send to the browser
type ServerMessage struct {
	Type   string `json:"type"`           // "state", "error", "assigned"
	Game   *Game  `json:"game,omitempty"` // Current game state
	Mark   string `json:"mark,omitempty"` // "X" or "O" (when assigned)
	Error  string `json:"error,omitempty"`
}

// Global game state (we'll only have one game for simplicity)
var (
	game  = &Game{Turn: "X"}  // X always goes first
	mutex = &sync.Mutex{}     // Protects game from concurrent access
)
