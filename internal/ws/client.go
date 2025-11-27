package ws

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

// Client представляет одно подключение WebSocket.
type Client struct {
	conn   *websocket.Conn
	hub    *Hub
	userID uuid.UUID
	send   chan []byte
}

// NewClient создаёт нового клиента.
func NewClient(conn *websocket.Conn, hub *Hub, userID uuid.UUID) *Client {
	return &Client{
		conn:   conn,
		hub:    hub,
		userID: userID,
		send:   make(chan []byte, 16),
	}
}

// Run запускает обработку входящих и исходящих сообщений.
func (c *Client) Run(ctx context.Context) {
	go c.writePumpSafe()
	c.readPump(ctx)
}

// writePumpSafe запускает writePump с обработкой panic
func (c *Client) writePumpSafe() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("WebSocket writePump panic recovered: %v\nStack trace:\n%s\n", r, debug.Stack())
			c.Close()
		}
	}()
	c.writePump()
}

// Close закрывает соединение.
func (c *Client) Close() {
	c.hub.Unregister(c)
	c.conn.Close()
}

func (c *Client) readPump(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("WebSocket readPump panic recovered: %v\nStack trace:\n%s\n", r, debug.Stack())
		}
		c.Close()
	}()

	// Увеличиваем лимит чтения для больших сообщений (512KB)
	c.conn.SetReadLimit(512 * 1024)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Читаем сообщения, но не обрабатываем их (клиент только получает сообщения от сервера)
			_, _, err := c.conn.ReadMessage()
			if err != nil {
				// Логируем ошибку только если это не нормальное закрытие
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					// Можно добавить логирование здесь при необходимости
				}
				return
			}
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
