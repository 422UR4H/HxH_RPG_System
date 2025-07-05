package game

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// TODO: evaluate to action
type Message struct {
	Nick string `json:"nick"`
	Msg  string `json:"msg"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: IN PRODUCTION, IMPLEMENT ORIGIN CHECKING
		return true
	},
}
var broadcast = make(chan Message)
var clients = make(map[*websocket.Conn]bool)

func main() {
	http.HandleFunc("/ws", handleConnections)

	// Start the message handler in a separate goroutine
	go handleMessages()

	// Start the server
	log.Println("Server started on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// TODO: move to handlers
func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// TODO: verify this before game testing with other players
		log.Fatal(err)
	}
	defer ws.Close()

	clients[ws] = true
	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Println(err)
			delete(clients, ws)
			break
		}
		broadcast <- msg
	}
}

func handleMessages() {
	for {
		msg := <-broadcast
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Println(err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}
