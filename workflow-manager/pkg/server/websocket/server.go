package websocket

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

func writeErrorWebSocket(c *websocket.Conn, msg string) {
	formatedMessage := msg + " ...Connection Closed"
	err := c.WriteMessage(websocket.TextMessage, []byte(formatedMessage))
	if err != nil {
		log.Println("failed writing error message to websocket", err)
	}
}

// ServeConnection handles the incoming connection and serves access logs
func NewWebSocketServer() *mux.Router {
	router := mux.NewRouter()
	return router
}
