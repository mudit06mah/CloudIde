package ws

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
	"github.com/mudit06mah/CloudIde/k8s"
)

// StartWebSocketServer initializes the router
func StartWebSocketServer() error {
	wsPort := os.Getenv("WS_PORT")
	if wsPort == "" {
		wsPort = "8080"
	}

	http.HandleFunc("/ws", wsHandler)
	log.Println("WebSocket server started on port:", wsPort)
	return http.ListenAndServe(":"+wsPort, nil)
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	connType := query.Get("type")

	switch connType {
	case "terminal":
		// 1. Get Pod Name
		podName := query.Get("pod")
		if podName == "" {
			http.Error(w, "Query missing pod name", http.StatusBadRequest)
			return
		}

		// 2. Get Workspace ID (The Fix)
		workspaceId := query.Get("workspaceId")
		if workspaceId == "" {
			http.Error(w, "Query missing workspaceId", http.StatusBadRequest)
			return
		}

		// 3. Create K8s Client using the REAL Workspace ID
		k8sClient, err := k8s.NewK8sClient(workspaceId)
		if err != nil {
			http.Error(w, "Failed to create K8s client: "+err.Error(), http.StatusInternalServerError)
			return
		}

		HandleTerminal(w, r, k8sClient.Clientset, k8sClient.Config, podName)
		return

	default:
		// IDE Logic (File Tree, Editor, etc.)
		handleWebSocket(w, r)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to upgrade connection:", err)
		return
	}
	// Note: We don't defer conn.Close() here because Session manages it, 
	// or we rely on the loop breaking. 
	// Ideally, Session should own the connection lifecycle.
	
	session := NewSession(conn)
	defer conn.Close() // Close when loop breaks

	// Listen for messages
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			// Remove session logic could go here
			break
		}
		// Route message to the specific session instance
		session.HandleMessage(msg)
	}
}