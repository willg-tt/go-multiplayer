package main

import (
	"math/rand"

	"github.com/gorilla/websocket"
)

// Client represents any connected user (player or spectator)
type Client struct {
	Conn *websocket.Conn
	Role string // "X", "O", or "spectator"
	Name string // Player's chosen name
}

// Action represents any event sent to the game manager
type Action struct {
	Type   ActionType
	Client *Client
	X      int    // For moves
	Y      int    // For moves
	Text   string // For chat
	Name   string // For setName
}

// Channels for communication
var (
	actions   = make(chan Action)    // All actions go here
	clients   = make(map[*Client]bool) // All connected clients
)

// startGameManager runs the single goroutine that owns all game state
func startGameManager() {
	for {
		action := <-actions // Wait for an action

		switch action.Type {
		case ActionJoin:
			handleJoin(action.Client)

		case ActionLeave:
			handleLeave(action.Client)

		case ActionMove:
			handleMoveAction(action.Client, action.X, action.Y)

		case ActionAttack:
			handleAttackAction(action.Client, action.X, action.Y)

		case ActionRoll:
			handleRollAction(action.Client)

		case ActionReset:
			handleResetAction()

		case ActionChat:
			handleChatAction(action.Client, action.Text)

		case ActionSetName:
			handleSetName(action.Client, action.Name)
		}
	}
}

const MaxClients = 10

func handleJoin(client *Client) {
	// Check connection limit
	if len(clients) >= MaxClients {
		sendJSON(client.Conn, ServerMessage{Type: "error", Error: "Server full (max 10 players)"})
		client.Conn.Close()
		return
	}

	// Assign role: first player is X, second is O, rest are spectators
	if game.PlayerX == nil {
		client.Role = "X"
		game.PlayerX = &Player{Conn: client.Conn, Mark: "X"}
	} else if game.PlayerO == nil {
		client.Role = "O"
		game.PlayerO = &Player{Conn: client.Conn, Mark: "O"}
	} else {
		client.Role = "spectator"
	}

	clients[client] = true

	// Tell this client their role
	sendJSON(client.Conn, ServerMessage{Type: "assigned", Mark: client.Role})

	// Send current game state
	sendJSON(client.Conn, ServerMessage{Type: "state", Game: game})

	// Announce to everyone
	broadcastToAll(ServerMessage{
		Type:    "chat",
		From:    "system",
		Message: client.Role + " joined",
	})
}

func handleLeave(client *Client) {
	delete(clients, client)

	// If a player left, clear their slot
	if client.Role == "X" {
		game.PlayerX = nil
	} else if client.Role == "O" {
		game.PlayerO = nil
	}

	// Announce to everyone
	broadcastToAll(ServerMessage{
		Type:    "chat",
		From:    "system",
		Message: client.Role + " left",
	})
}

func handleMoveAction(client *Client, x, y int) {
	// Only players can move
	if client.Role != "X" && client.Role != "O" {
		sendJSON(client.Conn, ServerMessage{Type: "error", Error: "Spectators cannot move"})
		return
	}

	// Validate turn
	if game.Turn != client.Role {
		sendJSON(client.Conn, ServerMessage{Type: "error", Error: "Not your turn"})
		return
	}

	// Validate game not over
	if game.Winner != "" {
		sendJSON(client.Conn, ServerMessage{Type: "error", Error: "Game is over"})
		return
	}

	// Get the player's unit
	var unit *Unit
	if client.Role == "X" {
		unit = game.UnitX
	} else {
		unit = game.UnitO
	}

	// Validate move is within 3 squares (Chebyshev distance)
	dx := abs(x - unit.X)
	dy := abs(y - unit.Y)
	distance := dx
	if dy > dx {
		distance = dy
	}
	if distance == 0 || distance > 3 {
		sendJSON(client.Conn, ServerMessage{Type: "error", Error: "Can only move up to 3 squares"})
		return
	}

	// Validate position is in bounds
	if x < 0 || x >= BoardSize || y < 0 || y >= BoardSize {
		sendJSON(client.Conn, ServerMessage{Type: "error", Error: "Out of bounds"})
		return
	}

	// Validate destination is empty
	if game.Board[y][x] != "" {
		sendJSON(client.Conn, ServerMessage{Type: "error", Error: "Square occupied"})
		return
	}

	// Move the unit
	game.Board[unit.Y][unit.X] = "" // Clear old position
	unit.X = x
	unit.Y = y
	game.Board[y][x] = client.Role // Set new position

	// Check if landed on a power-up
	checkPowerUpCollection(unit, client.Role)

	// Switch turns
	if game.Turn == "X" {
		game.Turn = "O"
	} else {
		game.Turn = "X"
	}

	// Maybe spawn a power-up for the next turn
	maybeSpawnPowerUp()

	// Broadcast to everyone
	broadcastToAll(ServerMessage{Type: "state", Game: game})
}

// abs returns the absolute value of n
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func handleResetAction() {
	game.Board = [BoardSize][BoardSize]string{}
	game.Turn = "X"
	game.Winner = ""
	game.PowerUps = nil // Clear power-ups
	game.initializeUnits()

	// Swap players
	if game.PlayerX != nil && game.PlayerO != nil {
		// Find the clients and swap their roles
		for client := range clients {
			if client.Role == "X" {
				client.Role = "O"
			} else if client.Role == "O" {
				client.Role = "X"
			}
		}
		// Swap the player pointers
		game.PlayerX, game.PlayerO = game.PlayerO, game.PlayerX
		game.PlayerX.Mark = "X"
		game.PlayerO.Mark = "O"
	}

	// Tell everyone their (possibly new) roles and the new state
	for client := range clients {
		sendJSON(client.Conn, ServerMessage{Type: "assigned", Mark: client.Role})
	}
	broadcastToAll(ServerMessage{Type: "state", Game: game})
}

func handleAttackAction(client *Client, x, y int) {
	// Only players can attack
	if client.Role != "X" && client.Role != "O" {
		sendJSON(client.Conn, ServerMessage{Type: "error", Error: "Spectators cannot attack"})
		return
	}

	// Validate turn
	if game.Turn != client.Role {
		sendJSON(client.Conn, ServerMessage{Type: "error", Error: "Not your turn"})
		return
	}

	// Validate game not over
	if game.Winner != "" {
		sendJSON(client.Conn, ServerMessage{Type: "error", Error: "Game is over"})
		return
	}

	// Can't start new combat if one is pending
	if pendingCombat != nil {
		sendJSON(client.Conn, ServerMessage{Type: "error", Error: "Combat already in progress"})
		return
	}

	// Get attacker and defender units and marks
	var attacker, defender *Unit
	var attackerMark, defenderMark string
	if client.Role == "X" {
		attacker = game.UnitX
		defender = game.UnitO
		attackerMark = "X"
		defenderMark = "O"
	} else {
		attacker = game.UnitO
		defender = game.UnitX
		attackerMark = "O"
		defenderMark = "X"
	}

	// Check if clicking on the enemy position
	if defender.X != x || defender.Y != y {
		sendJSON(client.Conn, ServerMessage{Type: "error", Error: "No enemy at that position"})
		return
	}

	// Check if enemy is within 1 square (adjacent)
	dx := abs(attacker.X - defender.X)
	dy := abs(attacker.Y - defender.Y)
	if dx > 1 || dy > 1 {
		sendJSON(client.Conn, ServerMessage{Type: "error", Error: "Enemy not in range"})
		return
	}

	// Check if attacker has attack boost - instant 6 damage, no dice!
	if attacker.AttackBoost {
		attacker.AttackBoost = false // Consume the boost

		// Build instant combat result
		combat := &CombatResult{
			AttackerMark: attackerMark,
			DefenderMark: defenderMark,
			AttackerRoll: 6, // Show as max roll
			DefenderRoll: 0, // No defense
			Winner:       "attacker",
			LoserMark:    defenderMark,
			Damage:       6,
		}

		// Apply damage immediately
		defender.HP -= 6
		if defender.HP < 0 {
			defender.HP = 0
		}

		// Check for winner
		game.checkWinner()

		// If defender eliminated, remove from board
		if defender.HP <= 0 {
			game.Board[defender.Y][defender.X] = ""
		}

		// Switch turns (if game not over)
		if game.Winner == "" {
			if game.Turn == "X" {
				game.Turn = "O"
			} else {
				game.Turn = "X"
			}
		}

		// Maybe spawn power-up
		maybeSpawnPowerUp()

		// Broadcast boosted attack result
		broadcastToAll(ServerMessage{Type: "combat_boosted", Game: game, Combat: combat})
		return
	}

	// Normal combat - Pre-roll dice (server determines outcome now, but don't reveal yet)
	attackRoll := rand.Intn(6) + 1 // 1-6
	defendRoll := rand.Intn(6) + 1 // 1-6

	// Build combat result (rolls hidden from clients until they click)
	combat := &CombatResult{
		AttackerMark:   attackerMark,
		DefenderMark:   defenderMark,
		AttackerRoll:   attackRoll,
		DefenderRoll:   defendRoll,
		AttackerRolled: false,
		DefenderRolled: false,
	}

	// Calculate outcome (but don't apply yet)
	if attackRoll >= defendRoll {
		combat.Winner = "attacker"
		combat.LoserMark = defenderMark
		combat.Damage = attackRoll - defendRoll
		if combat.Damage < 1 {
			combat.Damage = 1
		}
	} else {
		combat.Winner = "defender"
		combat.LoserMark = attackerMark
		combat.Damage = defendRoll - attackRoll
		if combat.Damage < 1 {
			combat.Damage = 1
		}
	}

	// Store pending combat
	pendingCombat = &PendingCombat{
		Combat:   combat,
		Attacker: attacker,
		Defender: defender,
	}

	// Broadcast combat start (without revealing rolls)
	broadcastToAll(ServerMessage{
		Type: "combat_start",
		Combat: &CombatResult{
			AttackerMark:   attackerMark,
			DefenderMark:   defenderMark,
			AttackerRolled: false,
			DefenderRolled: false,
		},
	})
}

func handleRollAction(client *Client) {
	// Must have pending combat
	if pendingCombat == nil {
		sendJSON(client.Conn, ServerMessage{Type: "error", Error: "No combat in progress"})
		return
	}

	combat := pendingCombat.Combat

	// Check if this player is part of the combat
	isAttacker := client.Role == combat.AttackerMark
	isDefender := client.Role == combat.DefenderMark

	if !isAttacker && !isDefender {
		sendJSON(client.Conn, ServerMessage{Type: "error", Error: "You are not in this combat"})
		return
	}

	// Mark this player as having rolled
	if isAttacker {
		if combat.AttackerRolled {
			return // Already rolled
		}
		combat.AttackerRolled = true
	} else {
		// Defender can only roll after attacker
		if !combat.AttackerRolled {
			sendJSON(client.Conn, ServerMessage{Type: "error", Error: "Wait for attacker to roll first"})
			return
		}
		if combat.DefenderRolled {
			return // Already rolled
		}
		combat.DefenderRolled = true
	}

	// Broadcast that someone rolled
	// Include attacker's roll if they've rolled (so defender knows what to beat)
	rolledMsg := &CombatResult{
		AttackerMark:   combat.AttackerMark,
		DefenderMark:   combat.DefenderMark,
		AttackerRolled: combat.AttackerRolled,
		DefenderRolled: combat.DefenderRolled,
	}
	if combat.AttackerRolled {
		rolledMsg.AttackerRoll = combat.AttackerRoll
	}
	broadcastToAll(ServerMessage{
		Type:   "combat_rolled",
		Combat: rolledMsg,
	})

	// If both have rolled, resolve combat
	if combat.AttackerRolled && combat.DefenderRolled {
		resolveCombat()
	}
}

func resolveCombat() {
	if pendingCombat == nil {
		return
	}

	combat := pendingCombat.Combat
	attacker := pendingCombat.Attacker
	defender := pendingCombat.Defender

	// Apply damage
	if combat.Winner == "attacker" {
		defender.HP -= combat.Damage
		if defender.HP < 0 {
			defender.HP = 0
		}
	} else {
		attacker.HP -= combat.Damage
		if attacker.HP < 0 {
			attacker.HP = 0
		}
	}

	// Check for winner
	game.checkWinner()

	// If defender eliminated, remove from board
	if defender.HP <= 0 {
		game.Board[defender.Y][defender.X] = ""
	}
	// If attacker eliminated (from counter), remove from board
	if attacker.HP <= 0 {
		game.Board[attacker.Y][attacker.X] = ""
	}

	// Switch turns (if game not over)
	if game.Winner == "" {
		if game.Turn == "X" {
			game.Turn = "O"
		} else {
			game.Turn = "X"
		}
	}

	// Clear pending combat
	pendingCombat = nil

	// Maybe spawn power-up
	maybeSpawnPowerUp()

	// Broadcast final combat result and new state
	broadcastToAll(ServerMessage{Type: "combat", Game: game, Combat: combat})
}

func handleChatAction(client *Client, text string) {
	// Limit message length
	if len(text) > 200 {
		text = text[:200]
	}
	if len(text) == 0 {
		return
	}

	broadcastToAll(ServerMessage{
		Type:    "chat",
		From:    client.Role,
		Name:    client.Name,
		Message: text,
	})
}

func handleSetName(client *Client, name string) {
	// Limit name length
	if len(name) > 20 {
		name = name[:20]
	}
	if len(name) == 0 {
		return
	}

	client.Name = name

	// Announce name change
	broadcastToAll(ServerMessage{
		Type:    "chat",
		From:    "system",
		Name:    "",
		Message: client.Role + " is now known as " + name,
	})
}

// broadcastToAll sends a message to every connected client
func broadcastToAll(msg ServerMessage) {
	for client := range clients {
		sendJSON(client.Conn, msg)
	}
}

// maybeSpawnPowerUp has a chance to spawn a power-up on an empty square
func maybeSpawnPowerUp() {
	// 30% chance to spawn a power-up each turn
	if rand.Intn(100) >= 30 {
		return
	}

	// Limit to 3 power-ups on board at once
	if len(game.PowerUps) >= 3 {
		return
	}

	// Find empty squares (not occupied by units or other power-ups)
	var emptySquares [][2]int
	for y := 0; y < BoardSize; y++ {
		for x := 0; x < BoardSize; x++ {
			if game.Board[y][x] != "" {
				continue // Unit here
			}
			// Check if power-up already here
			hasPowerUp := false
			for _, p := range game.PowerUps {
				if p.X == x && p.Y == y {
					hasPowerUp = true
					break
				}
			}
			if !hasPowerUp {
				emptySquares = append(emptySquares, [2]int{x, y})
			}
		}
	}

	if len(emptySquares) == 0 {
		return
	}

	// Pick random empty square
	pos := emptySquares[rand.Intn(len(emptySquares))]

	// Pick random type (50/50)
	powerUpType := "hp"
	if rand.Intn(2) == 1 {
		powerUpType = "attack"
	}

	game.PowerUps = append(game.PowerUps, PowerUp{
		Type: powerUpType,
		X:    pos[0],
		Y:    pos[1],
	})
}

// checkPowerUpCollection checks if a unit landed on a power-up and applies it
func checkPowerUpCollection(unit *Unit, mark string) {
	for i := len(game.PowerUps) - 1; i >= 0; i-- {
		p := game.PowerUps[i]
		if p.X == unit.X && p.Y == unit.Y {
			// Collect it!
			if p.Type == "hp" {
				unit.HP += 3
				if unit.HP > unit.MaxHP {
					unit.HP = unit.MaxHP
				}
				broadcastToAll(ServerMessage{
					Type:    "chat",
					From:    "system",
					Message: mark + " collected HP boost! (+3 HP)",
				})
			} else if p.Type == "attack" {
				unit.AttackBoost = true
				broadcastToAll(ServerMessage{
					Type:    "chat",
					From:    "system",
					Message: mark + " collected Attack boost! (Next attack deals 6 damage)",
				})
			}
			// Remove from board
			game.PowerUps = append(game.PowerUps[:i], game.PowerUps[i+1:]...)
		}
	}
}
