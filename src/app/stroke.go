package main

import (
	"encoding/json"
	"fmt"
	"log"
	"runtime/debug"
	"time"
)

func need(err error) {
	if err != nil {
		debug.PrintStack()
		log.Println(err)
	}
}

type RoomRepo struct {
	lock  chan bool
	Rooms map[int64]*Room
}

func NewRoomRepo() *RoomRepo {
	return &RoomRepo{
		lock:  make(chan bool, 1),
		Rooms: map[int64]*Room{},
	}
}

func (r *RoomRepo) Lock() {
	r.lock <- true
}

func (r *RoomRepo) Unlock() {
	<-r.lock
}

func (r *RoomRepo) Init() {
	log.Println("room repo init start")

	r.Lock()
	defer r.Unlock()

	rooms := []Room{}
	err := dbx.Select(&rooms, "SELECT `id`, `name`, `canvas_width`, `canvas_height`, `created_at` FROM `rooms` ORDER BY `id` ASC")
	need(err)

	for i, _ := range rooms {
		strokes := []Stroke{}
		err := dbx.Select(&strokes, "SELECT `id`, `room_id`, `width`, `red`, `green`, `blue`, `alpha`, `created_at` FROM `strokes` WHERE `room_id` = ? ORDER BY `id` ASC", rooms[i].ID)
		need(err)

		var owner_id int64
		err = dbx.QueryRow("SELECT token_id FROM `room_owners` WHERE `room_id` = ?", rooms[i].ID).Scan(&owner_id)

		for j, s := range strokes {
			ps := []Point{}
			dbx.Select(&ps, "SELECT `id`, `stroke_id`, `x`, `y` FROM `points` WHERE `stroke_id` = ? ORDER BY `id` ASC", s.ID)
			strokes[j].Points = ps
			strokes[j].json, err = json.Marshal(strokes[j])
		}

		rooms[i].ownerID = owner_id
		rooms[i].Strokes = strokes
		rooms[i].StrokeCount = len(strokes)
		rooms[i].watchers = map[int64]time.Time{}
		r.Rooms[rooms[i].ID] = &rooms[i]
	}
	log.Println("room repo init end")
}

func (r *RoomRepo) Get(ID int64) (*Room, bool) {
	r.Lock()
	defer r.Unlock()

	room, ok := r.Rooms[ID]
	return room, ok
}

func (r *RoomRepo) UpdateWatcherCount(roomID int64, tokenID int64) int {
	r.Lock()
	defer r.Unlock()

	room, ok := r.Rooms[roomID]
	if !ok {
		log.Println("[warn] no such room")
	}

	if room.watchers == nil {
		room.watchers = map[int64]time.Time{}
	}
	room.watchers[tokenID] = time.Now()

	for token, t := range room.watchers {
		if time.Since(t) >= time.Second*3 {
			delete(room.watchers, token)
		}
	}

	room.WatcherCount = len(room.watchers)
	return room.WatcherCount
}

func (r *RoomRepo) GetWatcherCount(roomID int64) int {
	r.Lock()
	defer r.Unlock()

	room, ok := r.Rooms[roomID]
	if !ok {
		log.Println("[warn] no such room")
	}

	for token, t := range room.watchers {
		if time.Since(t) >= time.Second*3 {
			delete(room.watchers, token)
		}
	}
	room.WatcherCount = len(room.watchers)

	return room.WatcherCount
}

func (r *RoomRepo) GetStrokes(roomID int64, greaterThanID int64) []Stroke {
	result := []Stroke{}

	r.Lock()
	room, ok := r.Rooms[roomID]
	if !ok {
		log.Println("[warn] no such room")
	}

	// lockの外にだしたいが怖い
	for i, s := range room.Strokes {
		if s.ID > greaterThanID {
			result = room.Strokes[i:]
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
	room.watchers = map[int64]time.Time{}
	r.Rooms[room.ID] = room
}

func (r *RoomRepo) AddStroke(roomID int64, stroke Stroke, points []Point) {
	stroke.Points = points
	var err error
	stroke.json, err = json.Marshal(stroke)
	if err != nil {
		panic(err)
	}

	r.Lock()
	room, ok := r.Rooms[roomID]
	if !ok {
		log.Println("[warn] no such room")
		r.Unlock()
		return
	}

	room.Strokes = append(room.Strokes, stroke)
	room.StrokeCount = len(room.Strokes)

	room.svgMtx.Lock()
	r.Unlock()
	if room.svgInit {
		buf := room.svgBuf
		fmt.Fprintf(buf,
			`<polyline id="%d" stroke="rgba(%d,%d,%d,%v)" stroke-width="%d" stroke-linecap="round" stroke-linejoin="round" fill="none" points="`,
			stroke.ID, stroke.Red, stroke.Green, stroke.Blue, stroke.Alpha, stroke.Width)
		first := true
		for _, point := range stroke.Points {
			if !first {
				buf.WriteByte(' ')
			}
			fmt.Fprintf(buf, `%.4f,%.4f`, point.X, point.Y)
			first = false
		}
		buf.WriteString(`"></polyline>`)
		room.svgCompressed = compress(append(buf.Bytes(), "</svg>"...))
	}
	room.svgMtx.Unlock()
}
