package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"

	"github.com/veandco/go-sdl2/sdl"
)

const w = 64
const h = 32

type chip8 struct {
	opcode uint16
	memory [4096]byte

	// register
	v [16]byte

	index uint16
	pc    uint16

	gfx [w * h]byte

	delayTimer byte
	soundTimer byte

	stack [16]uint16
	sp    int

	key [16]byte

	drawFlag bool

	// graphics
	renderer *sdl.Renderer
	zoom     int32
}

// Initialize registers and memory once
func (c *chip8) initialize() {
	c.pc = 0x200
	c.opcode = 0
	c.index = 0
	c.sp = 0
	c.zoom = 10

	for i := 0; i < len(fontSet); i++ {
		c.memory[i] = fontSet[i]
	}
}

func (c *chip8) emulateCycle() {
	// Fetch Opcode
	c.opcode = uint16(c.memory[c.pc])<<8 | uint16(c.memory[c.pc+1])
	x := byte((c.opcode & 0x0F00) >> 8)
	y := byte((c.opcode & 0x00F0) >> 4)
	nn := byte(c.opcode & 0x00FF)
	nnn := uint16(c.opcode & 0x0FFF)
	// fmt.Printf("%X\n", c.opcode)

	// Decode Opcode
	// https://en.wikipedia.org/wiki/CHIP-8#Opcode_table
	// http://www.multigesture.net/wp-content/uploads/mirror/goldroad/chip8_instruction_set.shtml
	switch c.opcode & 0xF000 {
	case 0x0000:
		switch c.opcode & 0x000F {
		case 0x0000:
			// 00E0 Clear screen
			for i := 0; i < len(c.gfx); i++ {
				c.gfx[i] = 0
			}
			c.pc += 2
			c.drawFlag = true
			break
		case 0x000E:
			// 00EE Returns from a subroutine
			c.sp--
			c.pc = c.stack[c.sp]
			break
		default:
			fmt.Printf("Unknown opcode [0x0000]: 0x%X\n", c.opcode)
		}
	case 0x1000:
		// 1NNN Jumps to address NNN
		// c.pc = c.opcode & 0x0FFF
		c.pc = nnn
		break
	case 0x2000:
		// 2NNN Calls subroutine at NNN
		c.stack[c.sp] = c.pc
		c.sp++
		// c.pc = c.opcode & 0x0FFF
		c.pc = nnn
		break
	case 0x3000:
		// 3XNN Skips the next instruction if VX equals NN. (Usually
		// the next instruction is a jump to skip a code block)
		if c.v[x] == nn {
			c.pc += 2
		}
		c.pc += 2
		break
	case 0x4000:
		// 4XNN Skips the next instruction if VX doesn't equal NN.
		// (Usually the next instruction is a jump to skip a code
		// block)
		if c.v[x] != nn {
			c.pc += 2
		}
		c.pc += 2
		break
	case 0x5000:
		// 5XY0 Skips the next instruction if VX equals VY. (Usually
		// the next instruction is a jump to skip a code block)
		if c.v[x] != c.v[y] {
			c.pc += 2
		}
		c.pc += 2
		break
	case 0x6000:
		// 6XNN Sets VX to NN
		c.v[x] = nn
		c.pc += 2
		break
	case 0x7000:
		// 7XNN Adds NN to VX. (Carry flag is not changed)
		// c.v[c.opcode&0x0F00>>8] += byte(c.opcode)
		c.v[x] += nn
		c.pc += 2
		break
	case 0x8000:
		switch c.opcode & 0x000F {
		case 0x0000:
			// 0x8XY0 Sets VX to the value of VY
			c.v[x] = c.v[y]
			c.pc += 2
		case 0x0001:
			// 0x8XY1 Sets VX to VX or VY. (Bitwise OR operation)
			c.v[x] = (c.v[x] | c.v[y])
			c.pc += 2
		case 0x0002:
			// 0x8XY2 Sets VX to VX and VY. (Bitwise AND operation)
			c.v[x] = (c.v[x] & c.v[y])
			c.pc += 2
		case 0x0003:
			// 0x8XY3 Sets VX to VX xor VY
			c.v[x] = (c.v[x] ^ c.v[y])
			c.pc += 2
		case 0x0004:
			// 0x8XY4 Adds VY to VX. VF is set to 1 when there's a
			// carry, and to 0 when there isn't
			if c.v[y] > (0xFF - c.v[x]) {
				c.v[0xF] = 1
			} else {
				c.v[0xF] = 0
			}
			c.v[x] += c.v[y]
			c.pc += 2
			break
		case 0x0005:
			// 0x8XY5 VY is subtracted from VX. VF is set to 0 when
			// there's a borrow, and 1 when there isn't
			if c.v[x] > c.v[y] {
				c.v[0xF] = 0x1
			} else {
				c.v[0xF] = 0x0
			}
			c.v[x] -= c.v[y]
			c.pc += 2
		case 0x0006:
			// 0x8XY6 Stores the least significant bit of VX in VF
			// and then shifts VX to the right by 1
			if c.opcode&0x1 >= 1 {
				c.v[0xF] = 1
			} else {
				c.v[0xF] = 0
			}
			c.v[x] = c.v[x] >> 1
			c.pc += 2
		case 0x0007:
			// 0x8XY7 Sets VX to VY minus VX. VF is set to 0 when
			// there's a borrow, and 1 when there isn't
			if c.v[y] > c.v[x] {
				c.v[0xF] = 0x1
			} else {
				c.v[0xF] = 0x0
			}
			c.v[x] = c.v[y] - c.v[x]
			c.pc += 2
		case 0x000E:
			// 0x8XYE Stores the most significant bit of VX in VF
			// and then shifts VX to the left by 1
			if c.opcode&0x80 == 0x80 {
				c.v[0xF] = 1
			} else {
				c.v[0xF] = 0
			}
			c.v[x] = c.v[x] << 1
			c.pc += 2
		default:
			fmt.Printf("Unknown opcode [0x8000]: 0x%X\n", c.opcode)
		}
	case 0x9000:
		// 9XY0 Skips the next instruction if VX doesn't equal VY.
		// (Usually the next instruction is a jump to skip a code
		// block)
		if c.v[x] != c.v[y] {
			c.pc += 2
		}
		c.pc += 2
	case 0xA000:
		// ANNN set index to NNN position
		c.index = nnn
		c.pc += 2
		break
	case 0xB000:
		// BNNN Jumps to the address NNN plus V0
		c.pc = nnn + uint16(c.v[0])
	case 0xC000:
		// CXNN Sets VX to the result of a bitwise and operation on a
		// random number (Typically: 0 to 255) and NN
		r := byte(rand.Intn(255))
		c.v[x] = r & nn
		c.pc += 2
	case 0xD000:
		// DXYN Draws a sprite at coordinate (VX, VY) that has a width
		// of 8 pixels and a height of N+1 pixels. Each row of 8 pixels
		// is read as bit-coded starting from memory location I; I
		// value doesn’t change after the execution of this
		// instruction. As described above, VF is set to 1 if any
		// screen pixels are flipped from set to unset when the sprite
		// is drawn, and to 0 if that doesn’t happen

		height := c.opcode & 0x000F

		var pixel byte
		c.v[0xF] = 0

		for yline := uint16(0); yline < height; yline++ {
			pixel = c.memory[c.index+yline]
			for xline := uint16(0); xline < 8; xline++ {
				if (pixel & (0x80 >> xline)) != 0 {
					if c.gfx[(uint16(c.v[x])+xline)+((uint16(c.v[y])+yline)*w)] == 1 {
						c.v[0xF] = 1
					}
					c.gfx[(uint16(c.v[x])+xline)+((uint16(c.v[y])+yline)*w)] ^= 1
				}
			}

		}

		c.drawFlag = true
		c.pc += 2
		break
	case 0xE000:
		switch c.opcode & 0x00FF {
		case 0x009E:
			// EX9E Skips the next instruction if the key stored in
			// VX is pressed. (Usually the next instruction is a
			// jump to skip a code block)
			if c.key[c.v[x]] != 0 {
				c.pc += 2
			}
			c.pc += 2
			break
		case 0x00A1:
			// EXA1 Skips the next instruction if the key stored in
			// VX isn't pressed. (Usually the next instruction is a
			// jump to skip a code block)
			if c.key[c.v[x]] != 1 {
				c.pc += 2
			}
			c.pc += 2
		default:
			fmt.Printf("Unknown opcode [0xE000]: 0x%X\n", c.opcode)
		}
		break
	case 0xF000:
		switch c.opcode & 0x00FF {
		case 0x0007:
			// FX07 Sets VX to the value of the delay timer
			c.v[x] = c.delayTimer
			c.pc += 2
		case 0x000A:
			// FX0A A key press is awaited, and then stored in VX.
			// (Blocking Operation. All instruction halted until
			// next key event)
			pressed := false
			for i := 0; i < 16; i++ {
				if c.key[i] == 1 {
					c.v[x] = byte(i)
					pressed = true
				}
			}
			if !pressed {
				return
			}
			c.pc += 2
		case 0x0015:
			// FX15 Sets the delay timer to VX
			c.delayTimer = c.v[x]
			c.pc += 2
		case 0x0018:
			// FX18 Sets the sound timer to VX
			c.soundTimer = c.v[x]
			c.pc += 2
		case 0x001E:
			// FX1E Adds VX to I. VF is not affected
			c.index += uint16(c.v[x])
			c.pc += 2
		case 0x0029:
			// FX29 Sets I to the location of the sprite for the character in VX. Characters 0-F (in hexadecimal) are represented by a 4x5 font
			c.index = uint16(c.v[x]) * 5
			c.pc += 2
			break
		case 0x0033:
			c.memory[c.index] = c.v[x] / 100
			c.memory[c.index+1] = (c.v[x] / 10) % 10
			c.memory[c.index+2] = (c.v[x] / 100) % 10
			c.pc += 2
			break
		case 0x0055:
			// FX55 Stores V0 to VX (including VX) in memory
			// starting at address I. The offset from I is
			// increased by 1 for each value written, but I itself
			// is left unmodified
			for i := uint16(0); i <= uint16(x); i++ {
				c.memory[c.index+i] = c.v[i]
			}
			c.pc += 2
		case 0x0065:
			// 0xFX65 Fills V0 to VX (including VX) with values
			// from memory starting at address I. The offset from I
			// is increased by 1 for each value written, but I
			// itself is left unmodified
			for i := uint16(0); i < uint16(x)+1; i++ {
				c.v[i] = c.memory[c.index+i]
			}
			c.pc += 2
			break
		default:
			fmt.Printf("Unknown opcode [0xF000]: 0x%X\n", c.opcode)
		}
		break
	default:
		fmt.Printf("Unknown opcode: 0x%X\n", c.opcode)
	}

	// Update timers
	if c.delayTimer > 0 {
		c.delayTimer--
	}
	if c.soundTimer > 0 {
		if c.soundTimer == 1 {
			fmt.Printf("Beep!\n")
		}
		c.soundTimer--
	}
}

func (c *chip8) loadGame(filepath string) error {
	buffer, err := ioutil.ReadFile(filepath)
	if err != nil {
		return err
	}

	for i := 0; i < len(buffer); i++ {
		// 0x200 == 512
		c.memory[512+i] = buffer[i]
	}
	return nil
}

var fontSet = [80]byte{
	0xF0, 0x90, 0x90, 0x90, 0xF0, // 0
	0x20, 0x60, 0x20, 0x20, 0x70, // 1
	0xF0, 0x10, 0xF0, 0x80, 0xF0, // 2
	0xF0, 0x10, 0xF0, 0x10, 0xF0, // 3
	0x90, 0x90, 0xF0, 0x10, 0x10, // 4
	0xF0, 0x80, 0xF0, 0x10, 0xF0, // 5
	0xF0, 0x80, 0xF0, 0x90, 0xF0, // 6
	0xF0, 0x10, 0x20, 0x40, 0x40, // 7
	0xF0, 0x90, 0xF0, 0x90, 0xF0, // 8
	0xF0, 0x90, 0xF0, 0x10, 0xF0, // 9
	0xF0, 0x90, 0xF0, 0x90, 0x90, // A
	0xE0, 0x90, 0xE0, 0x90, 0xE0, // B
	0xF0, 0x80, 0x80, 0x80, 0xF0, // C
	0xE0, 0x90, 0x90, 0x90, 0xE0, // D
	0xF0, 0x80, 0xF0, 0x80, 0xF0, // E
	0xF0, 0x80, 0xF0, 0x80, 0x80, // F
}

func (c *chip8) setupInput() {
}

func (c *chip8) setupGraphics() {
	fmt.Println(c.zoom)
	window, err := sdl.CreateWindow("go-chip8", sdl.WINDOWPOS_UNDEFINED,
		sdl.WINDOWPOS_UNDEFINED, w*c.zoom, h*c.zoom, sdl.WINDOW_SHOWN)
	if err != nil {
		panic(err)
	}
	c.renderer, err = sdl.CreateRenderer(window, -1, 0)
	if err != nil {
		panic(err)
	}
}

func (c *chip8) drawGraphics() {
	// x := 0
	// y := 0
	// for i := 0; i < len(c.gfx); i++ {
	//         if i%w == 0 {
	//                 x = 0
	//                 y++
	//                 fmt.Println("")
	//         }
	//         if c.gfx[i]^1 == 0 {
	//                 c.renderer.DrawPoint(int32(x), int32(y))
	//                 fmt.Print("x")
	//         } else {
	//                 fmt.Print(" ")
	//         }
	//         x++
	// }

	// for y := 0; y < h; y++ {
	//         for x := 0; x < w; x++ {
	//                 pixel := c.gfx[y*w+x]
	//                 if pixel != 0 {
	//                         c.renderer.DrawPoint(int32(x), int32(y))
	//                 }
	//         }
	// }

	c.renderer.SetDrawColor(0, 0, 0, 1)
	c.renderer.Clear()
	c.renderer.SetDrawColor(255, 255, 255, 1)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			pixel := c.gfx[y*w+x]
			if pixel != 0 {
				c.renderer.FillRect(&sdl.Rect{
					X: int32(x) * c.zoom,
					Y: int32(y) * c.zoom,
					W: c.zoom,
					H: c.zoom,
				})
			}
		}
	}

	c.renderer.Present()
	c.drawFlag = false
}

func (c *chip8) setKeys() {
	for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
		switch t := event.(type) {
		case *sdl.QuitEvent:
			println("Quit")
			os.Exit(0)
			// running = false
			// break
		case *sdl.KeyboardEvent:
			switch t.Type {
			case sdl.KEYDOWN:
				if keyHex, ok := validKeys[t.Keysym.Sym]; ok {
					fmt.Println("down", t.Keysym.Sym)
					c.key[keyHex] = 1
					fmt.Println(keyHex, c.key[keyHex])
				}
			case sdl.KEYUP:
				if keyHex, ok := validKeys[t.Keysym.Sym]; ok {
					fmt.Println("up", t.Keysym.Sym)
					c.key[keyHex] = 0
					fmt.Println(keyHex, c.key[keyHex])
				}
				if t.Keysym.Sym == sdl.K_ESCAPE {
					fmt.Println("EXIT")
					os.Exit(0)
				}
			}
		}
	}

}

var validKeys = map[sdl.Keycode]byte{
	sdl.K_0: 0x00,
	sdl.K_1: 0x01,
	sdl.K_2: 0x02,
	sdl.K_3: 0x03,
	sdl.K_4: 0x04,
	sdl.K_5: 0x05,
	sdl.K_6: 0x06,
	sdl.K_7: 0x07,
	sdl.K_8: 0x08,
	sdl.K_9: 0x09,
	sdl.K_a: 0x0a,
	sdl.K_b: 0x0b,
	sdl.K_c: 0x0c,
	sdl.K_d: 0x0d,
	sdl.K_e: 0x0e,
	sdl.K_f: 0x0f,
}

func main() {
	filepath := flag.String("file", "file-path", "file path of the input file")

	flag.Parse()

	var c chip8
	c.initialize()
	c.setupGraphics()
	c.setupInput()

	err := c.loadGame(*filepath)
	if err != nil {
		panic(err)
	}

	for {
		c.emulateCycle()
		if c.drawFlag {
			c.drawGraphics()
		}
		c.setKeys()
	}
}
