package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"goji.io/pat"
	"golang.org/x/net/context"
)

func renderRoomImage(w io.Writer, room *Room) {
	room.svgMtx.Lock()

	if !room.svgInit {
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

		room.svgBuf = buf
		room.svgInit = true
	}

	svg := room.svgBuf.Bytes()
	room.svgMtx.Unlock()

	w.Write(svg)
	w.Write([]byte(`</svg>`))
}

func getRoomImageID(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	idStr := pat.Param(ctx, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)

	if err != nil {
		outputErrorMsg(w, http.StatusNotFound, "この部屋は存在しません。")
		return
	}

	room, ok := roomRepo.Get(id)
	if !ok {
		log.Println("getRoomImageID", "room not found")
		outputErrorMsg(w, http.StatusNotFound, "この部屋は存在しません。")
		return
	}

	w.Header().Set("Content-Type", "image/svg+xml")
	renderRoomImage(w, room)
}
