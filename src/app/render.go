package main

import (
	"bytes"
	"fmt"
	"github.com/klauspost/compress/gzip"
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
		gbuf := append(buf.Bytes(), "</svg>"...)
		room.svgCompressed = compress(gbuf)
		room.svgInit = true
	}

	gsvg := room.svgCompressed
	room.svgMtx.Unlock()

	w.Write(gsvg)
}

func compress(src []byte) []byte {
	buf := &bytes.Buffer{}
	w, err := gzip.NewWriterLevel(buf, 7)
	if err != nil {
		panic(err)
	}
	w.Write(src)
	w.Close()
	return buf.Bytes()
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
	w.Header().Set("Content-Encoding", "gzip")
	renderRoomImage(w, room)
}
