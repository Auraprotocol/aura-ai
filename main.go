package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Define WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Message struct for WebSocket communication
type Message struct {
	MessageType string       `json:"messageType"` // "feedback" or "retrieve"
	Behavior    UserBehavior `json:"behavior,omitempty"`
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
	mu        sync.Mutex     // Mutex for thread-safe access
}

// NewAIModel initializes the AI model
func NewAIModel() *AIModel {
	return &AIModel{
		Knowledge: make(map[string]int),
	}
}

// UpdateModel updates the knowledge base based on feedback
func (ai *AIModel) UpdateModel(behavior UserBehavior) {
	ai.mu.Lock()
	defer ai.mu.Unlock()

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

	log.Printf("Updated Knowledge for action '%s' from device '%s' with weight %d: %d\n",
		behavior.Action, behavior.Device, weight, ai.Knowledge[behavior.Action])
}

// RetrieveKnowledge returns the current knowledge base as JSON
func (ai *AIModel) RetrieveKnowledge() string {
	ai.mu.Lock()
	defer ai.mu.Unlock()

	data, err := json.MarshalIndent(ai.Knowledge, "", "  ")
	if err != nil {
		log.Printf("Error marshaling knowledge base: %v", err)
		return "{}"
	}
	return string(data)
}

// SaveKnowledgeToFile saves the AI knowledge base to a JSON file
func SaveKnowledgeToFile(ai *AIModel, filename string) {
	ai.mu.Lock()
	defer ai.mu.Unlock()

	file, err := os.Create(filename)
	if err != nil {
		log.Printf("Error creating file: %v", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(ai.Knowledge); err != nil {
		log.Printf("Error saving knowledge: %v", err)
	} else {
		log.Println("Knowledge base saved to file.")
	}
}

// LoadKnowledgeFromFile loads the AI knowledge base from a JSON file
func LoadKnowledgeFromFile(ai *AIModel, filename string) {
	file, err := os.Open(filename)
	if err != nil {
		log.Println("No previous knowledge found. Starting fresh.")
		return
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&ai.Knowledge); err != nil {
		log.Printf("Error loading knowledge: %v", err)
	} else {
		log.Println("Knowledge base loaded from file.")
	}
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
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println("Error reading JSON:", err)
			break
		}

		switch msg.MessageType {
		case "feedback":
			ai.UpdateModel(msg.Behavior)
			conn.WriteMessage(websocket.TextMessage, []byte("Feedback processed successfully"))
		case "retrieve":
			knowledge := ai.RetrieveKnowledge()
			conn.WriteMessage(websocket.TextMessage, []byte(knowledge))
		default:
			conn.WriteMessage(websocket.TextMessage, []byte("Unknown message type"))
		}
	}
}

func main() {
	aiModel := NewAIModel()
	LoadKnowledgeFromFile(aiModel, "knowledge.json")

	// Periodic save of knowledge base
	go func() {
		for {
			time.Sleep(30 * time.Second)
			SaveKnowledgeToFile(aiModel, "knowledge.json")
		}
	}()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(aiModel, w, r)
	})

	port := "8080"
	log.Printf("WebSocket server started on ws://localhost:%s/ws\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
