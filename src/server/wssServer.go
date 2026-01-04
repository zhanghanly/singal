package singal

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type Message struct {
	Type    string      `json:"type"`
	Content interface{} `json:"content"`
	Sender  string      `json:"sender"`
	Time    int64       `json:"time"`
}

type WsServer struct {
	Users      map[*User]bool
	Register   chan *User
	Unregister chan *User
	Broadcast  chan Message
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NewServer() *WsServer {
	return &WsServer{
		Users:      make(map[*User]bool),
		Register:   make(chan *User),
		Unregister: make(chan *User),
		Broadcast:  make(chan Message),
	}
}

func (w *WsServer) Run() {
	for {
		select {
		case user := <-w.Register:
			w.Users[user] = true
			logger.Infof("user id=%s connected", user.userId)

			joinMsg := Message{
				Type:    "system",
				Content: fmt.Sprintf("user id=%s join room", user.userId),
				Sender:  "server",
				Time:    time.Now().Unix(),
			}
			w.Broadcast <- joinMsg

		case user := <-w.Unregister:
			if _, ok := w.Users[user]; ok {
				delete(w.Users, user)
				close(user.sendMsg)
				logger.Infof("user id=%s disconnected", user.userId)

				leaveMsg := Message{
					Type:    "system",
					Content: fmt.Sprintf("user id=%s disconnected", user.userId),
					Sender:  "server",
					Time:    time.Now().Unix(),
				}
				w.Broadcast <- leaveMsg
			}

		case message := <-w.Broadcast:
			for user := range w.Users {
				select {
				case user.sendMsg <- message:
				default:
					close(user.sendMsg)
					delete(w.Users, user)
				}
			}
		}
	}
}

// handle WebSocket connection
func (w *WsServer) HandleWebSocket(rw http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(rw, r, nil)
	if err != nil {
		logger.Fatal("WebSocket upgrade failed: ", err)
		return
	}

	user := NewUser(conn, w)
	user.userId = fmt.Sprintf("user_%d", time.Now().UnixNano())

	w.Register <- user

	go user.WriteMessage()
	go user.ReadMessage()
}
