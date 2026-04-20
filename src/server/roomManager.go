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
		room := NewRoom(roomId)
		if room != nil {
			rm.rooms[roomId] = room
			logger.Infof("create room roomId=%s", roomId)
		}

		return room
	}

	return rm.rooms[roomId]
}

func (rm *RoomManager) DeleteRoom(roomId string) {
	delete(rm.rooms, roomId)
}
