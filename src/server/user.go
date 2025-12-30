package singal

import (
	"time"
)

type User struct {
	userId   string
	createTs int64
}

func NewUser(id string) *User {
	return &User{
		userId:   id,
		createTs: time.Now().Unix(),
	}
}
