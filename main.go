package main

import (
	"flag"
	"fmt"
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
	return nil
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
