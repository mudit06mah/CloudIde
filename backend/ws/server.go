package ws

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

var workspaces = make(map[string]string)


// initialize ws server
func StartWebSocketServer() error {
	
	wsPort := os.Getenv("WS_PORT")

	http.HandleFunc("/ws", handleWebSocket)
	log.Println("WebSocket server started on port:", wsPort)
	return http.ListenAndServe(":"+wsPort, nil)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade connection", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	// Handle messages from the client
	for {
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Error reading message:", err)
			break
		}

		messageHandler(conn,msg)

		if err := conn.WriteMessage(messageType, msg); err != nil {
			fmt.Println("Error writing message:", err)
			break
		}
	}
}
