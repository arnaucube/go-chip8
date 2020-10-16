package main

import (
	"flag"
	"fmt"
	"io/ioutil"
)

type chip8 struct {
	opcode uint16
	memory [4096]byte

	// register
	v [16]byte

	index uint16
	pc    uint16

	gfx [64 * 32]byte

	delayTimer byte
	soundTimer byte

	stack [16]uint16
	sp    int

	key [16]byte

	drawFlag bool
}

// Initialize registers and memory once
func (c *chip8) initialize() {
	c.pc = 0x200
	c.opcode = 0
	c.index = 0
	c.sp = 0
}

func (c *chip8) emulateCycle() {
	// Fetch Opcode
	c.opcode = uint16(c.memory[c.pc])<<8 | uint16(c.memory[c.pc+1])

	// Decode Opcode
	// https://en.wikipedia.org/wiki/CHIP-8#Opcode_table
	// http://www.multigesture.net/wp-content/uploads/mirror/goldroad/chip8_instruction_set.shtml
	switch c.opcode & 0xF000 {
	case 0x0000:
		switch c.opcode & 0x000F {
		case 0x0000: // 0x00E0
			// clear screen
			// TODO
			c.pc += 2
			break
		case 0x000E: // 0x00EE
			// TODO
			break
		default:
			fmt.Printf("Unknown opcode [0x0000]: 0x%X\n", c.opcode)
		}
	case 0x2000:
		c.stack[c.sp] = c.pc
		c.sp++
		c.pc = c.opcode & 0x0FFF
		break
	case 0x6000:
		pos := (c.opcode & 0x0F00) >> 8
		c.v[pos] = byte(c.opcode)
		c.pc += 2
		break
	case 0x7000:
		c.v[c.opcode&0x0F00>>8] += byte(c.opcode)
		c.pc += 2
		break
	case 0x8000:
		switch c.opcode & 0x000F {
		case 0x0004: // 0x8XY4
			if c.v[(c.opcode&0x00F0)>>4] > (0xFF - c.v[c.opcode&0x0F00]) {
				c.v[0xF] = 1
			} else {
				c.v[0xF] = 0
			}
			c.v[(c.opcode&0x0F00)>>8] += c.v[(c.opcode&0x00F0)>>4]
			c.pc += 2
			break
		default:
			fmt.Printf("Unknown opcode [0x8000]: 0x%X\n", c.opcode)
		}
	case 0xA000:
		// set index to NNN position
		c.index = c.opcode & 0x0FFF
		c.pc += 2
		break
	case 0xD000:
		x := uint16(c.v[(c.opcode&0x0F00)>>8])
		y := uint16(c.v[(c.opcode&0x00F0)>>4])
		height := c.opcode & 0x000F

		var pixel byte
		c.v[0xF] = 0
		for yline := uint16(0); yline < height; yline++ {
			pixel = c.memory[c.index+yline]
			for xline := uint16(0); xline < 8; xline++ {
				if (pixel & (0x80 >> xline)) != 0 {
					if c.gfx[(x+xline+((y+yline)*64))] == 1 {
						c.v[0xF] = 1
					}
					c.gfx[x+xline+((y+yline)*64)] ^= 1
				}
			}

		}
		c.drawFlag = true
		c.pc += 2
		break
	case 0xE000:
		switch c.opcode & 0x00FF {
		case 0x009E:
			if c.key[c.v[(c.opcode&0x0F00)>>8]] != 0 {
				c.pc += 4
			} else {
				c.pc += 2
			}
			break
		}
		break
	case 0xF000:
		x := (c.opcode & 0x0F00) >> 8
		fmt.Printf("F: 0x%X\n", c.opcode)
		switch c.opcode & 0x00FF {
		case 0x0029:
			// TODO
			// c.index = uint16(c.v[x]) *
			c.pc += 2
			break
		case 0x0033:
			c.memory[c.index] = c.v[x] / 100
			c.memory[c.index+1] = (c.v[x] / 10) % 10
			c.memory[c.index+2] = (c.v[x] / 100) % 10
			c.pc += 2
			break
		case 0x0065:
			for i := uint16(0); i < x+1; i++ {
				c.v[i] = c.memory[c.index+i]
			}
			c.pc += 2
			break
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

func (c *chip8) drawGraphics() {

}

func (c *chip8) setKeys() {

}

func main() {
	filepath := flag.String("file", "file-path", "file path of the input file")

	flag.Parse()

	// setupGraphics()
	// setupInput()

	var c chip8
	c.initialize()
	err := c.loadGame(*filepath)
	if err != nil {
		panic(err)
	}

	for {
		c.emulateCycle()
		if c.drawFlag {
			c.drawGraphics()
			c.setKeys()
		}
	}
}
