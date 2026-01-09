package publicfiles

import (
	"sync"
	"time"

	dtos "github.com/Open-Source-Life/AxolotlDrive/DTOs"
	"github.com/gofiber/contrib/websocket"
	"github.com/google/uuid"
)

type Client struct {
	ID   string
	Conn *websocket.Conn
	Send chan interface{}
	Subs map[string]bool
	mu   sync.RWMutex
}

type WebSocketHub struct {
	clients    map[*Client]bool
	broadcast  chan interface{}
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan interface{}, 100),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
		case msg := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- msg:
				default:
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *WebSocketHub) Broadcast(msg dtos.WebSocketMessage) {
	select {
	case h.broadcast <- msg:
	default:
	}
}

func (h *WebSocketHub) HandleConnection(c *websocket.Conn) {
	client := &Client{
		ID:   uuid.New().String(),
		Conn: c,
		Send: make(chan interface{}, 10),
		Subs: make(map[string]bool),
	}
	h.register <- client

	connMsg := dtos.WebSocketMessage{
		EventType: "connection_established",
		Data: map[string]interface{}{
			"client_id": client.ID,
			"timestamp": time.Now().Unix(),
		},
		Timestamp: time.Now().Unix(),
	}
	client.Send <- connMsg

	go h.readPump(client)
	h.writePump(client)
}

func (h *WebSocketHub) readPump(client *Client) {
	defer func() {
		h.unregister <- client
		client.Conn.Close()
	}()
	for {
		var msg dtos.WebSocketMessage
		if err := client.Conn.ReadJSON(&msg); err != nil {
			break
		}
		switch msg.EventType {
		case "subscribe":
			if data, ok := msg.Data.(map[string]interface{}); ok {
				if paths, ok := data["paths"].([]interface{}); ok {
					client.mu.Lock()
					for _, p := range paths {
						if path, ok := p.(string); ok {
							client.Subs[path] = true
						}
					}
					client.mu.Unlock()
				}
			}
		case "unsubscribe":
			if data, ok := msg.Data.(map[string]interface{}); ok {
				if paths, ok := data["paths"].([]interface{}); ok {
					client.mu.Lock()
					for _, p := range paths {
						if path, ok := p.(string); ok {
							delete(client.Subs, path)
						}
					}
					client.mu.Unlock()
				}
			}
		case "ping":
			pongMsg := dtos.WebSocketMessage{
				EventType: "pong",
				Data:      map[string]interface{}{},
				Timestamp: time.Now().Unix(),
			}
			client.Send <- pongMsg
		}
	}
}

func (h *WebSocketHub) writePump(client *Client) {
	for msg := range client.Send {
		if err := client.Conn.WriteJSON(msg); err != nil {
			break
		}
	}
}
