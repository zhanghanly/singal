package media_center

import (
	"time"
)

type User struct {
	userId string
	joinTs int64
}

func NewUser(id string) *User {
	return &User{
		userId: id,
		joinTs: time.Now().Unix(),
	}
}
