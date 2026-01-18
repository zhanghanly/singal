package singal

type RoomManager struct {
	rooms map[string]*Room
}

var gRoomManager *RoomManager

func NewRoomManager() {
	gRoomManager = &RoomManager{
		rooms: make(map[string]*Room),
	}
}

func (rm *RoomManager) GetOrCreateRoom(roomId string) *Room {
	if _, ok := rm.rooms[roomId]; !ok {
		rm.rooms[roomId] = NewRoom(roomId)
		logger.Infof("create room roomId=%s", roomId)
	}

	return rm.rooms[roomId]
}

func (rm *RoomManager) DeleteRoom(roomId string) {
	delete(rm.rooms, roomId)
}
