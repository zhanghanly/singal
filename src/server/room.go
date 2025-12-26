package media_center

import (
	"time"
)

type Room struct {
	roomId   string
	createTs int64
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
	if _, ok := r.users[user.userId]; ok {
		delete(r.users, user.userId)
	}
}
