package media_center

type RoomManager struct {
	rooms map[string]*Room
}

func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms: make(map[string]*Room),
	}
}

func (rm *RoomManager) GetOrCreateRoom(roomId string) *Room {
	if _, ok := rm.rooms[roomId]; !ok {
		rm.rooms[roomId] = NewRoom(roomId)
	}

	return rm.rooms[roomId]
}

func (rm *RoomManager) DeleteRoom(roomId string) {
	if _, ok := rm.rooms[roomId]; ok {
		delete(rm.rooms, roomId)
	}
}
