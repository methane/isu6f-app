package main

import (
	"log"
	"runtime/debug"
	"sync"
	"time"
)

func need(err error) {
	if err != nil {
		debug.PrintStack()
		log.Println(err)
	}
}

type RoomRepo struct {
	sync.Mutex
	Rooms map[int64]*Room
}

func NewRoomRepo() *RoomRepo {
	return &RoomRepo{
		Rooms: map[int64]*Room{},
	}
}

func (r *RoomRepo) Init() {
	log.Println("room repo init start")
	r.Lock()
	defer r.Unlock()

	rooms := []Room{}
	err := dbx.Select(&rooms, "SELECT `id`, `name`, `canvas_width`, `canvas_height`, `created_at` FROM `rooms` ORDER BY `id` ASC")
	need(err)

	for i, room := range rooms {
		strokes := []Stroke{}
		err := dbx.Select(&strokes, "SELECT `id`, `room_id`, `width`, `red`, `green`, `blue`, `alpha`, `created_at` FROM `strokes` WHERE `room_id` = ? ORDER BY `id` ASC", room.ID)
		need(err)
		rooms[i].Strokes = strokes
		rooms[i].watchers = map[int64]time.Time{}

		var owner_id int64
		err = dbx.QueryRow("SELECT token_id FROM `room_owners` WHERE `room_id` = ?", room.ID).Scan(&owner_id)
		rooms[i].ownerID = owner_id

		for j, s := range strokes {
			ps := []Point{}
			dbx.Select(&ps, "SELECT `id`, `stroke_id`, `x`, `y` FROM `points` WHERE `stroke_id` = ? ORDER BY `id` ASC", s.ID)
			strokes[j].Points = ps
		}

		r.Rooms[room.ID] = &room
	}
	log.Println("room repo init end")
}

func (r *RoomRepo) Get(ID int64) (Room, bool) {
	r.Lock()
	defer r.Unlock()

	room, ok := r.Rooms[ID]
	return *room, ok
}

func (r *RoomRepo) UpdateWatcherCount(roomID int64, tokenID int64) {
	r.Lock()
	defer r.Unlock()

	room, ok := r.Rooms[roomID]
	if !ok {
		log.Println("[warn] no such room")
	}

	room.watchers[tokenID] = time.Now()

	for token, t := range room.watchers {
		if time.Since(t) >= time.Second*3 {
			delete(room.watchers, token)
		}
	}

	room.WatcherCount = len(room.watchers)
}

func (r *RoomRepo) GetWatcherCount(roomID int64) int {
	r.Lock()
	defer r.Unlock()

	room, ok := r.Rooms[roomID]
	if !ok {
		log.Println("[warn] no such room")
	}
	return room.WatcherCount
}

func (r *RoomRepo) GetStrokes(roomID int64, greaterThanID int64) []Stroke {
	var result []Stroke

	r.Lock()
	room, ok := r.Rooms[roomID]
	if !ok {
		log.Println("[warn] no such room")
	}

	// lockの外にだしたいが怖い
	for i, s := range room.Strokes {
		if s.ID > greaterThanID {
			result = append(result, room.Strokes[i:]...)
			break
		}
	}
	r.Unlock()
	return result
}

func (r *RoomRepo) GetStrokeCount(roomID int64) int {
	r.Lock()
	defer r.Unlock()
	room, ok := r.Rooms[roomID]
	if !ok {
		log.Println("[warn] no such room")
		return 0
	}
	return len(room.Strokes)
}

func (r *RoomRepo) AddRoom(room *Room, ownerID int64) {
	r.Lock()
	defer r.Unlock()

	room.ownerID = ownerID
	r.Rooms[room.ID] = room
}

func (r *RoomRepo) AddStroke(roomID int64, stroke Stroke, points []Point) {
	r.Lock()
	defer r.Unlock()

	room, ok := r.Rooms[roomID]
	if !ok {
		log.Println("[warn] no such room")
		return
	}

	stroke.Points = points
	room.Strokes = append(room.Strokes, stroke)
}
