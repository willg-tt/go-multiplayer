package main

import "github.com/gorilla/websocket"

const BoardSize = 9
const MaxHP = 10

// Unit represents a player's unit on the board
type Unit struct {
	X           int  `json:"x"`
	Y           int  `json:"y"`
	HP          int  `json:"hp"`
	MaxHP       int  `json:"maxHp"`
	AttackBoost bool `json:"attackBoost"` // Next attack bypasses dice, deals 6 damage
}

// PowerUp represents a collectible on the board
type PowerUp struct {
	Type string `json:"type"` // "hp" or "attack"
	X    int    `json:"x"`
	Y    int    `json:"y"`
}

// CombatResult holds the details of a combat exchange for animation
type CombatResult struct {
	AttackerMark   string `json:"attackerMark"`             // "X" or "O"
	DefenderMark   string `json:"defenderMark"`             // "X" or "O"
	AttackerRoll   int    `json:"attackerRoll"`             // 1-6
	DefenderRoll   int    `json:"defenderRoll"`             // 1-6
	Winner         string `json:"winner"`                   // "attacker" or "defender"
	Damage         int    `json:"damage"`                   // Damage dealt to loser
	LoserMark      string `json:"loserMark"`                // Who took damage ("X" or "O")
	AttackerRolled bool   `json:"attackerRolled,omitempty"` // Has attacker clicked their dice?
	DefenderRolled bool   `json:"defenderRolled,omitempty"` // Has defender clicked their dice?
}

// PendingCombat tracks an in-progress combat waiting for both players to roll
type PendingCombat struct {
	Combat   *CombatResult
	Attacker *Unit
	Defender *Unit
}

// Game represents the Grid Wars game state
type Game struct {
	Board    [BoardSize][BoardSize]string `json:"board"`    // "", "X", or "O"
	Turn     string                       `json:"turn"`     // "X" or "O"
	Winner   string                       `json:"winner"`   // "", "X", or "O"
	PlayerX  *Player                      `json:"-"`        // - means don't include in JSON
	PlayerO  *Player                      `json:"-"`
	UnitX    *Unit                        `json:"unitX"`
	UnitO    *Unit                        `json:"unitO"`
	PowerUps []PowerUp                    `json:"powerUps"` // Active power-ups on board
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
	Type    string        `json:"type"`              // "state", "error", "assigned", "chat", "combat"
	Game    *Game         `json:"game,omitempty"`    // Current game state
	Mark    string        `json:"mark,omitempty"`    // "X", "O", or "spectator"
	Error   string        `json:"error,omitempty"`
	From    string        `json:"from,omitempty"`    // Role: "X", "O", "spectator", "system"
	Name    string        `json:"name,omitempty"`    // Display name (optional)
	Message string        `json:"message,omitempty"` // Chat message text
	Combat  *CombatResult `json:"combat,omitempty"`  // Combat result for animation
}

// Global game state - only touched by game manager goroutine, no mutex needed!
var game = newGame()

// Pending combat - set when combat starts, cleared when both roll
var pendingCombat *PendingCombat

// newGame creates a fresh game with units initialized
func newGame() *Game {
	g := &Game{Turn: "X"}
	g.initializeUnits()
	return g
}

// initializeUnits spawns units in opposite corners
func (g *Game) initializeUnits() {
	// X spawns bottom-left (0, 8), O spawns top-right (8, 0)
	g.UnitX = &Unit{X: 0, Y: BoardSize - 1, HP: MaxHP, MaxHP: MaxHP}
	g.UnitO = &Unit{X: BoardSize - 1, Y: 0, HP: MaxHP, MaxHP: MaxHP}
	g.Board[g.UnitX.Y][g.UnitX.X] = "X"
	g.Board[g.UnitO.Y][g.UnitO.X] = "O"
}

// checkWinner checks if a unit has been eliminated
func (g *Game) checkWinner() {
	if g.UnitX != nil && g.UnitX.HP <= 0 {
		g.Winner = "O"
		return
	}
	if g.UnitO != nil && g.UnitO.HP <= 0 {
		g.Winner = "X"
		return
	}
}
