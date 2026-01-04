package singal

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
	delete(rm.rooms, roomId)
}
