package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gofiber/websocket/v2"
	"vps-panel/internal/services/monitor"
)

type Hub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mutex      sync.RWMutex
}

var WSHub *Hub

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

func (h *Hub) Run() {
	go h.broadcastStats()

	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
			}
			h.mutex.Unlock()

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
					h.mutex.RUnlock()
					h.unregister <- client
					h.mutex.RLock()
				}
			}
			h.mutex.RUnlock()
		}
	}
}

func (h *Hub) broadcastStats() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		h.mutex.RLock()
		clientCount := len(h.clients)
		h.mutex.RUnlock()

		if clientCount == 0 {
			continue
		}

		stats, err := monitor.GetSystemStats()
		if err != nil {
			continue
		}

		data, err := json.Marshal(stats)
		if err != nil {
			continue
		}

		h.broadcast <- data
	}
}

func (h *Hub) Register(conn *websocket.Conn) {
	h.register <- conn
}

func (h *Hub) Unregister(conn *websocket.Conn) {
	h.unregister <- conn
}

func HandleWebSocket(c *websocket.Conn) {
	WSHub.Register(c)
	defer WSHub.Unregister(c)

	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			break
		}
	}
}

func InitHub() {
	WSHub = NewHub()
	go WSHub.Run()
}
