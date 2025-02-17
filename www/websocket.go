package www

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	ws "github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var upgrader = ws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Client struct {
	hub    *Hub
	conn   *ws.Conn
	send   chan []byte
	name   string
	active bool
}

func NewClient(hub *Hub, w http.ResponseWriter, r *http.Request, name string) (*Client, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		name:   name,
		active: true,
	}, nil
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		c.hub.Unregister <- c
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(ws.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(ws.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(ws.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Maintains the set of active clients and broadcasts messages to clients
type Hub struct {
	Broadcast  chan []byte
	Register   chan *Client
	Unregister chan *Client
	clients    map[*Client]bool
	mutex      sync.Mutex
	logger     *slog.Logger
}

func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		logger:     logger,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.logger.Debug("registering client", "clientName", client.name)

			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()

		case client := <-h.Unregister:
			h.logger.Debug("unregistering client", "clientName", client.name)

			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mutex.Unlock()

		case message := <-h.Broadcast:
			h.mutex.Lock()
			for client := range h.clients {
				client.send <- message
			}
			h.mutex.Unlock()
		}
	}
}
