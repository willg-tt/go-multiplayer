package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Game represents the tic-tac-toe game state
type Game struct {
	Board   [3][3]string `json:"board"`   // "", "X", or "O"
	Turn    string       `json:"turn"`    // "X" or "O"
	Winner  string       `json:"winner"`  // "", "X", "O", or "draw"
	PlayerX *Player      `json:"-"`       // - means don't include in JSON
	PlayerO *Player      `json:"-"`
}

// Player represents a connected player
type Player struct {
	ID   string
	Conn *websocket.Conn
	Mark string // "X" or "O"
}

// Global game state (we'll only have one game for simplicity)
var (
	game  = &Game{Turn: "X"}           // X always goes first
	mutex = &sync.Mutex{}              // Protects game from concurrent access
)

func main() {
	// Serve static files from the "static" directory
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

	fmt.Println("Server starting on http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error:", err)
	}
}
