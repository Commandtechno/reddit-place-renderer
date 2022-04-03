package main

import (
	"encoding/csv"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

type Action struct {
	X         int
	Y         int
	Color     int
	Timestamp time.Time
}

func hex(hexInt int) color.Color {
	return color.RGBA{uint8(hexInt >> 16), uint8(hexInt >> 8), uint8(hexInt), 0xFF}
}

func main() {
	if _, err := os.Stat("data.csv"); err != nil {
		fmt.Println("Downloading data...")
		res, err := http.Get("https://storage.googleapis.com/justin_bassett/place_tiles")
		if err != nil {
			panic(err)
		}

		if res.StatusCode != 200 {
			panic(fmt.Sprintf("Status code: %d", res.StatusCode))
		}

		file, err := os.Create("data.csv")
		if err != nil {
			panic(err)
		}

		io.Copy(file, res.Body)
	}

	os.RemoveAll("frames")
	os.Mkdir("frames", os.ModePerm)

	palette := []color.Color{
		hex(0xFFFFFF), // 0
		hex(0xE4E4E4), // 1
		hex(0x888888), // 2
		hex(0x222222), // 3
		hex(0xFFA7D1), // 4
		hex(0xE50000), // 5
		hex(0xE59500), // 6
		hex(0xA06A42), // 7
		hex(0xE5D900), // 8
		hex(0x94E044), // 9
		hex(0x02BE01), // 10
		hex(0x00E5F0), // 11
		hex(0x0083C7), // 12
		hex(0x0000EA), // 13
		hex(0xE04AFF), // 14
		hex(0x820080), // 15
	}

	file, err := os.Open("data.csv")
	if err != nil {
		panic(err)
	}

	data := csv.NewReader(file)
	if _, err := data.Read(); err != nil {
		panic(err)
	}

	var actions []*Action

	fmt.Println("Collecting data...")
	for {
		record, err := data.Read()
		if err != nil {
			break
		}

		x, err := strconv.Atoi(record[2])
		if err != nil {
			// panic(err)
			continue
		}

		y, err := strconv.Atoi(record[3])
		if err != nil {
			// panic(err)
			continue
		}

		color, err := strconv.Atoi(record[4])
		if err != nil {
			// panic(err)
			continue
		}

		timestamp, err := time.Parse("2006-01-02 15:04:05.999 MST", record[0])
		if err != nil {
			panic(err)
		}

		actions = append(actions, &Action{
			X:         x,
			Y:         y,
			Color:     color,
			Timestamp: timestamp,
		})
	}

	fmt.Println("Sorting data...")
	sort.Slice(actions, func(a int, b int) bool {
		return actions[a].Timestamp.Before(actions[b].Timestamp)
	})

	fmt.Println("Rendering", len(actions), "actions...")
	canvas := image.NewRGBA(image.Rect(0, 0, 1000, 1000))

	totalFrames := 1000
	if len(actions)%totalFrames != 0 {
		totalFrames--
	}

	frameInterval := len(actions) / totalFrames
	currentFrame := 0

	for index, action := range actions {
		canvas.Set(action.X, action.Y, palette[action.Color])
		if index%frameInterval == 0 {
			fmt.Println("Rendering frame", currentFrame)
			out, err := os.Create(filepath.Join("frames", fmt.Sprintf("%03d", currentFrame)+".png"))
			if err != nil {
				panic(err)
			}

			if err := png.Encode(out, canvas); err != nil {
				panic(err)
			}

			currentFrame++
		}
	}

	fmt.Println("Rendering last frame")
	out, err := os.Create(filepath.Join("frames", fmt.Sprintf("%03d", currentFrame)+".png"))
	if err != nil {
		panic(err)
	}

	if err := png.Encode(out, canvas); err != nil {
		panic(err)
	}

	fmt.Println("Rendering video")
	cmd := exec.Command("ffmpeg", "-y", "-framerate", "60", "-i", "frames/%03d.png", "-c:v", "libx264", "-pix_fmt", "yuv420p", "place.mp4")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}