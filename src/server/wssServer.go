package media_center

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var clients = make(map[*websocket.Conn]bool) // 连接的客户端
var broadcast = make(chan Message)           // 广播通道

type Message struct {
	Username string `json:"username"`
	Text     string `json:"text"`
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	// 升级HTTP连接为WebSocket连接
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer ws.Close()

	// 注册新的客户端
	clients[ws] = true

	for {
		var msg Message
		// 读取客户端发送的消息
		err := ws.ReadJSON(&msg)
		if err != nil {
			delete(clients, ws)
			break
		}
		// 将消息发送到广播通道
		broadcast <- msg
	}
}

func handleMessages() {
	for {
		// 从广播通道获取消息
		msg := <-broadcast
		// 将消息发送给所有连接的客户端
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				client.Close()
				delete(clients, client)
			}
		}
	}
}
