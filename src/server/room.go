package singal

import (
	"time"
)

type Room struct {
	roomId   string
	createTs int64
	node     *SfuNode
	users    map[string]*User
}

func NewRoom(id string) *Room {
	return &Room{
		roomId:   id,
		createTs: time.Now().Unix(),
		users:    make(map[string]*User),
	}
}

func (r *Room) AddUser(user *User) {
	if _, ok := r.users[user.userId]; !ok {
		r.users[user.userId] = user
	}
}

func (r *Room) DeleteUser(user *User) {
	delete(r.users, user.userId)
}
