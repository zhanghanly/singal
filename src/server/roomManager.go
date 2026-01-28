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
		router, err := gRtcServer.CreateRouterOnWorker()
		if err != nil {
			logger.Fatalln("create router failed")
			return nil
		}
		rm.rooms[roomId] = NewRoom(roomId, router)
		logger.Infof("create room roomId=%s", roomId)
	}

	return rm.rooms[roomId]
}

func (rm *RoomManager) DeleteRoom(roomId string) {
	delete(rm.rooms, roomId)
}
