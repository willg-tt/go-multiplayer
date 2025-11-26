package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

// Upgrader converts HTTP connections to WebSocket connections
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins (fine for local dev)
	},
}

// sendJSON sends a JSON message to a player
func sendJSON(conn *websocket.Conn, msg ServerMessage) {
	conn.WriteJSON(msg)
}

// startBroadcaster runs a goroutine that listens for chat messages and broadcasts them
func startBroadcaster() {
	for {
		// Block until a message arrives on the channel
		b := <-broadcast
		fmt.Printf("Broadcasting: %+v\n", b.Message)

		// Send to all connected players
		mutex.Lock()
		if game.PlayerX != nil {
			sendJSON(game.PlayerX.Conn, b.Message)
		}
		if game.PlayerO != nil {
			sendJSON(game.PlayerO.Conn, b.Message)
		}
		mutex.Unlock()
	}
}

// handleWebSocket handles new WebSocket connections
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	fmt.Println("New player connected!")

	// Create a player for this connection
	player := &Player{
		ID:   r.RemoteAddr,
		Conn: conn,
	}

	// Assign player to X or O
	mutex.Lock()
	if game.PlayerX == nil {
		game.PlayerX = player
		player.Mark = "X"
		fmt.Println("Player assigned as X")
	} else if game.PlayerO == nil {
		game.PlayerO = player
		player.Mark = "O"
		fmt.Println("Player assigned as O")
	} else {
		fmt.Println("Game full, rejecting player")
		mutex.Unlock()
		sendJSON(conn, ServerMessage{Type: "error", Error: "Game is full"})
		return
	}
	mutex.Unlock()

	// Tell the player which mark they are
	sendJSON(conn, ServerMessage{Type: "assigned", Mark: player.Mark})

	// Send current game state
	sendJSON(conn, ServerMessage{Type: "state", Game: game})

	// Read messages from this player
	for {
		_, messageBytes, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Player disconnected:", player.Mark)
			mutex.Lock()
			if player.Mark == "X" {
				game.PlayerX = nil
			} else {
				game.PlayerO = nil
			}
			mutex.Unlock()
			break
		}

		// Parse the JSON message
		var msg ClientMessage
		err = json.Unmarshal(messageBytes, &msg)
		if err != nil {
			fmt.Println("Invalid JSON:", err)
			continue
		}

		fmt.Printf("Received from %s: %+v\n", player.Mark, msg)

		// Handle different message types
		switch msg.Type {
		case "move":
			handleMove(player, msg.X, msg.Y)
		case "reset":
			mutex.Lock()
			game.reset()
			// Tell players their new marks
			if game.PlayerX != nil {
				sendJSON(game.PlayerX.Conn, ServerMessage{Type: "assigned", Mark: "X"})
			}
			if game.PlayerO != nil {
				sendJSON(game.PlayerO.Conn, ServerMessage{Type: "assigned", Mark: "O"})
			}
			mutex.Unlock()
			broadcastState()
		case "chat":
			// Send chat through the channel - broadcaster will handle it
			broadcast <- Broadcast{
				Message: ServerMessage{
					Type:    "chat",
					From:    player.Mark,
					Message: msg.Message,
				},
			}
		}
	}
}

// handleMove processes a player's move
func handleMove(player *Player, x, y int) {
	mutex.Lock()
	defer mutex.Unlock()

	// Validate the move
	if game.Winner != "" {
		sendJSON(player.Conn, ServerMessage{Type: "error", Error: "Game is over"})
		return
	}

	if game.Turn != player.Mark {
		sendJSON(player.Conn, ServerMessage{Type: "error", Error: "Not your turn"})
		return
	}

	if x < 0 || x > 2 || y < 0 || y > 2 {
		sendJSON(player.Conn, ServerMessage{Type: "error", Error: "Invalid position"})
		return
	}

	if game.Board[y][x] != "" {
		sendJSON(player.Conn, ServerMessage{Type: "error", Error: "Cell already taken"})
		return
	}

	// Make the move
	game.Board[y][x] = player.Mark
	fmt.Printf("Player %s moved to (%d, %d)\n", player.Mark, x, y)

	// Switch turns
	if game.Turn == "X" {
		game.Turn = "O"
	} else {
		game.Turn = "X"
	}

	// Check for winner
	game.checkWinner()

	// Broadcast new state to both players
	broadcastState()
}

// broadcastState sends the current game state to all connected players
func broadcastState() {
	msg := ServerMessage{Type: "state", Game: game}

	if game.PlayerX != nil {
		sendJSON(game.PlayerX.Conn, msg)
	}
	if game.PlayerO != nil {
		sendJSON(game.PlayerO.Conn, msg)
	}
}
