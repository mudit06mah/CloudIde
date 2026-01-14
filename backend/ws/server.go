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
		podName := query.Get("pod")
		if podName == "" {
			http.Error(w, "Query missing pod name", http.StatusBadRequest)
			return
		}

		workspaceId := query.Get("workspaceId")
		if workspaceId == "" {
			http.Error(w, "Query missing workspaceId", http.StatusBadRequest)
			return
		}

		k8sClient, err := k8s.NewK8sClient(workspaceId)
		if err != nil {
			http.Error(w, "Failed to create K8s client: "+err.Error(), http.StatusInternalServerError)
			return
		}

		HandleTerminal(w, r, k8sClient.Clientset, k8sClient.Config, podName)
		return

	default:
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
	defer conn.Close()

	session := NewSession(conn)
	workspaceId := r.URL.Query().Get("workspaceId")
	defer session.cleanup(workspaceId)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			// Log disconnection if needed
			break
		}
		// Route message to the specific session instance
		session.HandleMessage(msg)
	}
}