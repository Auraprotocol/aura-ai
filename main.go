package main

import (
	"fmt"
	"log"
	"net/http"
	"github.com/gorilla/websocket"
)

// Define WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// UserBehavior struct for tracking user actions and feedback
type UserBehavior struct {
	ID       string `json:"id"`
	Action   string `json:"action"`
	Feedback int    `json:"feedback"` // -1 for negative feedback, 1 for positive feedback
	Device   string `json:"device"`  // e.g., "smartwatch", "smartphone"
}

// AIModel struct for holding the knowledge base
type AIModel struct {
	Knowledge map[string]int // Mapping each action to feedback score
}

// NewAIModel initializes the AI model
func NewAIModel() *AIModel {
	return &AIModel{
		Knowledge: make(map[string]int),
	}
}

// UpdateModel updates the knowledge base based on weighted feedback
func (ai *AIModel) UpdateModel(behavior UserBehavior) {
	weight := 1
	if ai.Knowledge[behavior.Action] > 5 {
		weight = 2 // Higher weight for strongly positive actions
	} else if ai.Knowledge[behavior.Action] < -5 {
		weight = 2 // Higher weight for strongly negative actions
	}

	if behavior.Feedback == 1 {
		ai.Knowledge[behavior.Action] += weight
	} else if behavior.Feedback == -1 {
		ai.Knowledge[behavior.Action] -= weight
	}

	fmt.Printf("Updated Knowledge for action '%s' with weight %d: %d\n", behavior.Action, weight, ai.Knowledge[behavior.Action])
}

// Log current knowledge base for debugging purposes
func logKnowledge(ai *AIModel) {
	fmt.Println("\n[LOG] Current AI Knowledge Base:")
	for action, score := range ai.Knowledge {
		fmt.Printf("Action: '%s', Score: %d\n", action, score)
	}
}

// Handle WebSocket connections for real-time feedback
func handleWebSocket(ai *AIModel, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading connection:", err)
		return
	}
	defer conn.Close()

	// Continuously receive data from the client
	for {
		var behavior UserBehavior
		err := conn.ReadJSON(&behavior)
		if err != nil {
			log.Println("Error reading JSON:", err)
			break
		}

		// Process the user behavior and update the AI model
		ai.UpdateModel(behavior)

		// Send acknowledgment to the client
		response := fmt.Sprintf("Received feedback for action '%s'", behavior.Action)
		err = conn.WriteMessage(websocket.TextMessage, []byte(response))
		if err != nil {
			log.Println("Error writing message:", err)
			break
		}
	}
}

func main() {
	// Initialize the AI model
	aiModel := NewAIModel()

	// Simulate initial behavior updates
	fmt.Println("\nSimulating initial user behavior:")
	initialBehaviors := []UserBehavior{
		{ID: "user_1", Action: "action_1", Feedback: 1, Device: "smartphone"},
		{ID: "user_2", Action: "action_2", Feedback: -1, Device: "smartwatch"},
	}

	for _, behavior := range initialBehaviors {
		aiModel.UpdateModel(behavior)
	}

	// Log the initial knowledge base
	logKnowledge(aiModel)

	// Set up HTTP server and WebSocket endpoint
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(aiModel, w, r)
	})

	// Start the WebSocket server
	port := "8080"
	fmt.Printf("\nWebSocket server started on ws://localhost:%s/ws\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
