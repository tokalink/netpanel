package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"runtime"

	"github.com/creack/pty"
	"github.com/gofiber/websocket/v2"
)

// TerminalMessage represents a message from the frontend
type TerminalMessage struct {
	Type string `json:"type"` // "input" or "resize"
	Data string `json:"data,omitempty"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
}

// TerminalHandler handles the websocket connection for the terminal
func TerminalHandler(c *websocket.Conn) {
	var cmd *exec.Cmd

	// Determine shell based on OS
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd.exe")
	} else {
		cmd = exec.Command("bash")
		// Fallback to sh if bash not found
		if _, err := exec.LookPath("bash"); err != nil {
			cmd = exec.Command("sh")
		}
	}

	// Start PTY
	ptmx, err := pty.Start(cmd)
	if err != nil {
		fmt.Printf("Terminal Error: Failed to start PTY: %v\n", err)
		c.WriteMessage(websocket.TextMessage, []byte("Failed to start terminal: "+err.Error()))
		return
	}
	defer func() {
		fmt.Println("Terminal closing...")
		_ = ptmx.Close()
		_ = cmd.Process.Kill()
	}()

	fmt.Println("Terminal started successfully")

	// Handle window resize
	chResize := make(chan TerminalMessage)
	go func() {
		for msg := range chResize {
			if err := pty.Setsize(ptmx, &pty.Winsize{
				Rows: uint16(msg.Rows),
				Cols: uint16(msg.Cols),
			}); err != nil {
				// Ignore resize errors
			}
		}
	}()

	// Copy PTY output to Websocket
	go func() {
		buffer := make([]byte, 1024)
		for {
			n, err := ptmx.Read(buffer)
			if err != nil {
				if err != io.EOF {
					// PTY closed
				}
				return
			}
			if err := c.WriteMessage(websocket.BinaryMessage, buffer[:n]); err != nil {
				return
			}
		}
	}()

	// Read from Websocket and write to PTY
	for {
		messageType, message, err := c.ReadMessage()
		if err != nil {
			break
		}

		if messageType == websocket.TextMessage {
			var msg TerminalMessage
			if err := json.Unmarshal(message, &msg); err == nil {
				if msg.Type == "resize" {
					chResize <- msg
					continue
				}
				if msg.Type == "input" {
					ptmx.Write([]byte(msg.Data))
				}
			} else {
				// Raw input fallback
				ptmx.Write(message)
			}
		} else if messageType == websocket.BinaryMessage {
			ptmx.Write(message)
		}
	}
}
