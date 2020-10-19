package main

import (
	"flag"
	"fmt"
	"go-chip8/chip8"
	"os"

	"github.com/veandco/go-sdl2/sdl"
)

// SdlEmulator represents the Chip8 emulator with Sdl frontend
type SdlEmulator struct {
	w        int
	h        int
	renderer *sdl.Renderer
	zoom     int32
	chip8    chip8.Chip8
}

// NewSdlEmulator creates a new SdlEmulator
func NewSdlEmulator(w, h int, zoom int32) SdlEmulator {
	c := chip8.NewChip8()

	window, err := sdl.CreateWindow("go-chip8", sdl.WINDOWPOS_UNDEFINED,
		sdl.WINDOWPOS_UNDEFINED, int32(w)*zoom, int32(h)*zoom, sdl.WINDOW_SHOWN)
	if err != nil {
		panic(err)
	}
	renderer, err := sdl.CreateRenderer(window, -1, 0)
	if err != nil {
		panic(err)
	}

	return SdlEmulator{
		w:        w,
		h:        h,
		renderer: renderer,
		zoom:     zoom,
		chip8:    c,
	}
}

func (e *SdlEmulator) drawGraphics() {
	e.renderer.SetDrawColor(0, 0, 0, 1)
	e.renderer.Clear()
	e.renderer.SetDrawColor(255, 255, 255, 1)
	for y := 0; y < e.h; y++ {
		for x := 0; x < e.w; x++ {
			pixel := e.chip8.Gfx[y*e.w+x]
			if pixel != 0 {
				e.renderer.FillRect(&sdl.Rect{
					X: int32(x) * e.zoom,
					Y: int32(y) * e.zoom,
					W: e.zoom,
					H: e.zoom,
				})
			}
		}
	}

	e.renderer.Present()
	e.chip8.DrawFlag = false
}

func (e *SdlEmulator) setKeys() {
	for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
		switch t := event.(type) {
		case *sdl.QuitEvent:
			fmt.Println("Quit")
			os.Exit(0)
		case *sdl.KeyboardEvent:
			switch t.Type {
			case sdl.KEYDOWN:
				if keyHex, ok := validKeys[t.Keysym.Sym]; ok {
					fmt.Println("down", t.Keysym.Sym)
					e.chip8.Key[keyHex] = 1
					fmt.Println(keyHex, e.chip8.Key[keyHex])
				}
			case sdl.KEYUP:
				if keyHex, ok := validKeys[t.Keysym.Sym]; ok {
					fmt.Println("up", t.Keysym.Sym)
					e.chip8.Key[keyHex] = 0
					fmt.Println(keyHex, e.chip8.Key[keyHex])
				}
				if t.Keysym.Sym == sdl.K_ESCAPE {
					fmt.Println("EXIT")
					os.Exit(0)
				}
			}
		}
	}

}

/*
	Key pad:     Keyboard keys:
	1 2 3 c      1 2 3 4
	4 5 6 d      q w e r
	7 8 9 e      a s d f
	a 0 b f      z x c v
*/
var validKeys = map[sdl.Keycode]byte{
	sdl.K_1: 0x01,
	sdl.K_2: 0x02,
	sdl.K_3: 0x03,
	sdl.K_4: 0x0c,
	sdl.K_q: 0x04,
	sdl.K_w: 0x05,
	sdl.K_e: 0x06,
	sdl.K_r: 0x0d,
	sdl.K_a: 0x07,
	sdl.K_s: 0x08,
	sdl.K_d: 0x09,
	sdl.K_f: 0x0e,
	sdl.K_z: 0x0a,
	sdl.K_x: 0x00,
	sdl.K_c: 0x0b,
	sdl.K_v: 0x0f,
}

func main() {
	filepath := flag.String("file", "file-path", "file path of the input file")

	flag.Parse()

	emulator := NewSdlEmulator(chip8.W, chip8.H, 8)

	err := emulator.chip8.LoadGame(*filepath)
	if err != nil {
		panic(err)
	}

	for {
		emulator.chip8.EmulateCycle()
		if emulator.chip8.DrawFlag {
			emulator.drawGraphics()
		}
		emulator.setKeys()
		sdl.Delay(200 / 60)
	}
}
