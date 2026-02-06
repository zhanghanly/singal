package singal

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type WsServer struct {
	Users      map[*User]bool
	Register   chan *User
	Unregister chan *User
	//Broadcast  chan Message
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
		//Broadcast:  make(chan Message),
	}
}

func (w *WsServer) Run() {
	for {
		select {
		case user := <-w.Register:
			w.Users[user] = true
			room := gRoomManager.GetOrCreateRoom(user.roomId)
			room.AddUser(user)
			logger.Infof("userId=%s peerId=%s roomId=%s connected", user.userId, user.PeerId, user.roomId)
			//w.Broadcast <- joinMsg

		case user := <-w.Unregister:
			if _, ok := w.Users[user]; ok {
				delete(w.Users, user)
				close(user.sendMsg)
				room := gRoomManager.GetOrCreateRoom(user.roomId)
				room.DeleteUser(user)
				logger.Infof("userId=%s peerId=%s roomId=%s disconnected", user.userId, user.PeerId, user.roomId)
				//w.Broadcast <- leaveMsg
			}

			//case message := <-w.Broadcast:
			//	for user := range w.Users {
			//		select {
			//		case user.sendMsg <- message:
			//		default:
			//			close(user.sendMsg)
			//			delete(w.Users, user)
			//		}
			//	}
		}
	}
}

// handle WebSocket connection
func (w *WsServer) HandleWebSocket(rw http.ResponseWriter, r *http.Request) {
	logger.Infof("recieve wbsocket connection, url=%s", r.URL.String())

	queryParams := r.URL.Query()
	roomId := queryParams.Get("roomId")
	peerId := queryParams.Get("peerId")
	clientProtocols := websocket.Subprotocols(r)
	if len(clientProtocols) > 0 {
		upgrader.Subprotocols = []string{clientProtocols[0]}
	}

	conn, err := upgrader.Upgrade(rw, r, nil)
	if err != nil {
		logger.Fatal("WebSocket upgrade failed: ", err)
		return
	}
	logger.Infof("accept wbsocket connection")

	user := NewUser(conn, w, peerId, roomId)
	user.userId = fmt.Sprintf("user_%d", time.Now().UnixNano())
	w.Register <- user

	go user.WriteMessage()
	go user.ReadMessage()
}

func StartWssServer() {
	server := NewServer()
	go server.Run()

	logger.Infoln("wss Server running on :8080...")
	http.HandleFunc("/", server.HandleWebSocket)
	http.ListenAndServe(":8080", nil)
	//err := http.ListenAndServeTLS(":4443", gConfig.Http.Cert, gConfig.Http.Key, nil)
	//if err != nil {
	//	logger.Warnf("start wss server failed, %v", err)
	//}
}
