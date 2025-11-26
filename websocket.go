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
		return true
	},
}

// sendJSON sends a JSON message to a client
func sendJSON(conn *websocket.Conn, msg ServerMessage) {
	conn.WriteJSON(msg)
}

// handleWebSocket handles new WebSocket connections
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	// Create client and tell game manager they joined
	client := &Client{Conn: conn}
	actions <- Action{Type: ActionJoin, Client: client}

	// Read messages and forward to game manager
	for {
		_, messageBytes, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Client disconnected:", client.Role)
			actions <- Action{Type: ActionLeave, Client: client}
			break
		}

		var msg ClientMessage
		err = json.Unmarshal(messageBytes, &msg)
		if err != nil {
			fmt.Println("Invalid JSON:", err)
			continue
		}

		fmt.Printf("Received from %s: %+v\n", client.Role, msg)

		// Forward to game manager via channel
		switch ActionType(msg.Type) {
		case ActionMove:
			actions <- Action{Type: ActionMove, Client: client, X: msg.X, Y: msg.Y}
		case ActionReset:
			actions <- Action{Type: ActionReset, Client: client}
		case ActionChat:
			actions <- Action{Type: ActionChat, Client: client, Text: msg.Message}
		case ActionSetName:
			actions <- Action{Type: ActionSetName, Client: client, Name: msg.Name}
		}
	}
}
