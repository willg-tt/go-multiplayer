package main

// ActionType represents the type of action sent to the game manager
type ActionType string

const (
	ActionJoin    ActionType = "join"
	ActionLeave   ActionType = "leave"
	ActionMove    ActionType = "move"
	ActionAttack  ActionType = "attack"
	ActionRoll    ActionType = "roll"
	ActionReset   ActionType = "reset"
	ActionChat    ActionType = "chat"
	ActionSetName ActionType = "setName"
)
