package main

import "github.com/gorilla/websocket"

// Client represents any connected user (player or spectator)
type Client struct {
	Conn *websocket.Conn
	Role string // "X", "O", or "spectator"
}

// Action represents any event sent to the game manager
type Action struct {
	Type   string // "join", "leave", "move", "reset", "chat"
	Client *Client
	X      int    // For moves
	Y      int    // For moves
	Text   string // For chat
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
		case "join":
			handleJoin(action.Client)

		case "leave":
			handleLeave(action.Client)

		case "move":
			handleMoveAction(action.Client, action.X, action.Y)

		case "reset":
			handleResetAction()

		case "chat":
			handleChatAction(action.Client, action.Text)
		}
	}
}

func handleJoin(client *Client) {
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

	// Validate position
	if x < 0 || x > 2 || y < 0 || y > 2 {
		sendJSON(client.Conn, ServerMessage{Type: "error", Error: "Invalid position"})
		return
	}

	if game.Board[y][x] != "" {
		sendJSON(client.Conn, ServerMessage{Type: "error", Error: "Cell already taken"})
		return
	}

	// Make the move
	game.Board[y][x] = client.Role

	// Switch turns
	if game.Turn == "X" {
		game.Turn = "O"
	} else {
		game.Turn = "X"
	}

	// Check for winner
	game.checkWinner()

	// Broadcast to everyone
	broadcastToAll(ServerMessage{Type: "state", Game: game})
}

func handleResetAction() {
	game.Board = [3][3]string{}
	game.Turn = "X"
	game.Winner = ""

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

func handleChatAction(client *Client, text string) {
	// Everyone can chat
	broadcastToAll(ServerMessage{
		Type:    "chat",
		From:    client.Role,
		Message: text,
	})
}

// broadcastToAll sends a message to every connected client
func broadcastToAll(msg ServerMessage) {
	for client := range clients {
		sendJSON(client.Conn, msg)
	}
}
