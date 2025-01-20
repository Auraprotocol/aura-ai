package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"github.com/gorilla/websocket"
)

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// UserBehavior struct
type UserBehavior struct {
	ID       string `json:"id"`
	Action   string `json:"action"`
	Feedback int    `json:"feedback"` // -1 for negative feedback, 1 for positive feedback
	Device   string `json:"device"`  // e.g., "smartwatch", "smartphone"
}

// AIModel struct
type AIModel struct {
	Knowledge map[string]int // AI knowledge base
}

// NewAIModel initializes AI model
func NewAIModel() *AIModel {
	return &AIModel{
		Knowledge: make(map[string]int),
	}
}

// UpdateModel updates knowledge with weighted feedback
func (ai *AIModel) UpdateModel(behavior UserBehavior) {
	weight := 1
	if ai.Knowledge[behavior.Action] > 5 {
		weight = 2
	} else if ai.Knowledge[behavior.Action] < -5 {
		weight = 2
	}

	if behavior.Feedback == 1 {
		ai.Knowledge[behavior.Action] += weight
	} else if behavior.Feedback == -1 {
		ai.Knowledge[behavior.Action] -= weight
	}

	fmt.Printf("Updated Knowledge for action '%s' with weight %d: %d\n", behavior.Action, weight, ai.Knowledge[behavior.Action])
}

// SaveKnowledgeToFile saves the knowledge base to a JSON file
func SaveKnowledgeToFile(ai *AIModel, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Error creating file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(ai.Knowledge)
	if err != nil {
		log.Fatalf("Error saving knowledge: %v", err)
	}

	fmt.Println("Knowledge base saved to file.")
}

// LoadKnowledgeFromFile loads the knowledge base from a JSON file
func LoadKnowledgeFromFile(ai *AIModel, filename string) {
	file, err := os.Open(filename)
	if err != nil {
		log.Printf("No previous knowledge found, starting fresh.")
		return
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&ai.Knowledge)
	if err != nil {
		log.Fatalf("Error loading knowledge: %v", err)
	}

	fmt.Println("Knowledge base loaded from file.")
}

// Handle WebSocket connections
func handleWebSocket(ai *AIModel, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading connection:", err)
		return
	}
	defer conn.Close()

	for {
		var behavior UserBehavior
		err := conn.ReadJSON(&behavior)
		if err != nil {
			log.Println("Error reading JSON:", err)
			break
		}

		// Process feedback
		ai.UpdateModel(behavior)

		// Send acknowledgment
		response := fmt.Sprintf("Received feedback for action '%s'", behavior.Action)
		err = conn.WriteMessage(websocket.TextMessage, []byte(response))
		if err != nil {
			log.Println("Error writing message:", err)
			break
		}
	}
}

func main() {
	// Initialize AI model and load existing knowledge
	aiModel := NewAIModel()
	LoadKnowledgeFromFile(aiModel, "knowledge.json")

	// Simulate initial behavior updates
	fmt.Println("\nSimulating initial user behavior:")
	initialBehaviors := []UserBehavior{
		{ID: "user_1", Action: "action_1", Feedback: 1, Device: "smartphone"},
		{ID: "user_2", Action: "action_2", Feedback: -1, Device: "smartwatch"},
	}

	for _, behavior := range initialBehaviors {
		aiModel.UpdateModel(behavior)
	}

	// Log the knowledge base
	log.Println("\nCurrent AI Knowledge Base:")
	for action, score := range aiModel.Knowledge {
		fmt.Printf("Action: '%s', Score: %d\n", action, score)
	}

	// Set up WebSocket endpoint
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(aiModel, w, r)
	})

	// Start WebSocket server
	port := "8080"
	fmt.Printf("\nWebSocket server started on ws://localhost:%s/ws\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))

	// Save knowledge to file when the program ends
	SaveKnowledgeToFile(aiModel, "knowledge.json")
}
