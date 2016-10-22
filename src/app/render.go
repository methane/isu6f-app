package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"

	"goji.io/pat"
)

func renderRoomImage(room *Room) []byte {
	buf := bytes.NewBuffer(make([]byte, 0, 1024))

	fmt.Fprintf(buf,
		`<?xml version="1.0" standalone="no"?><!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd"><svg xmlns="http://www.w3.org/2000/svg" version="1.1" baseProfile="full" width="%d" height="%d" style="width:%dpx;height:%dpx;background-color:white;" viewBox="0 0 %d %d">`,
		room.CanvasWidth, room.CanvasHeight,
		room.CanvasWidth, room.CanvasHeight,
		room.CanvasWidth, room.CanvasHeight)

	for _, stroke := range room.Strokes {
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
	}

	buf.WriteString(`</svg>`)
	return buf.Bytes()
}

func getRoomImageID(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	idStr := pat.Param(ctx, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		outputErrorMsg(w, http.StatusNotFound, "この部屋は存在しません。")
		return
	}

	room, err := getRoom(id)
	if err != nil {
		outputErrorMsg(w, http.StatusNotFound, "この部屋は存在しません。")
		return
	}

	svg := renderRoomImage(room)
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Write(svg)
}
