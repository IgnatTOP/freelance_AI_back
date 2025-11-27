package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/google/uuid"
)

// NotificationSaver интерфейс для сохранения уведомлений в БД.
type NotificationSaver interface {
	CreateNotification(ctx context.Context, userID uuid.UUID, event string, data interface{}) error
}

// Hub управляет всеми WebSocket клиентами.
type Hub struct {
	mu                sync.RWMutex
	clients           map[uuid.UUID]map[*Client]struct{}
	register          chan *Client
	unregister        chan *Client
	broadcast         chan message
	notificationSaver NotificationSaver
	ctx               context.Context
}

type message struct {
	userID  uuid.UUID
	payload []byte
}

// NewHub создаёт новый хаб.
func NewHub(ctx context.Context) *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]map[*Client]struct{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan message, 32),
		ctx:        ctx,
	}
}

// SetNotificationSaver устанавливает сервис для сохранения уведомлений.
func (h *Hub) SetNotificationSaver(saver NotificationSaver) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.notificationSaver = saver
}

// Run запускает главный цикл хаба.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.addClient(client)
		case client := <-h.unregister:
			h.removeClient(client)
		case msg := <-h.broadcast:
			h.send(msg.userID, msg.payload)
		}
	}
}

// Register добавляет клиента.
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister удаляет клиента.
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// BroadcastToUser отправляет сообщение конкретному пользователю и сохраняет уведомление в БД.
func (h *Hub) BroadcastToUser(userID uuid.UUID, event string, data any) error {
	// Сообщение для клиента строго следует контракту WebSocket API:
	// поле "type" содержит имя события, "data" — полезную нагрузку.
	payload := map[string]any{
		"type": event,
		"data": data,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("ws: не удалось сериализовать сообщение: %w", err)
	}

	// Сохраняем уведомление в БД, если установлен notification saver
	h.mu.RLock()
	saver := h.notificationSaver
	ctx := h.ctx
	h.mu.RUnlock()

	if saver != nil {
		// Сохраняем асинхронно, чтобы не блокировать отправку (с panic recovery)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("WebSocket notification save panic recovered: %v\nStack trace:\n%s\n", r, debug.Stack())
				}
			}()
			if err := saver.CreateNotification(ctx, userID, event, data); err != nil {
				// Логируем ошибку, но не прерываем отправку через WebSocket
				fmt.Printf("ws: не удалось сохранить уведомление: %v\n", err)
			}
		}()
	}

	h.broadcast <- message{userID: userID, payload: raw}
	return nil
}

func (h *Hub) addClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client.userID]; !ok {
		h.clients[client.userID] = make(map[*Client]struct{})
	}
	h.clients[client.userID][client] = struct{}{}
}

func (h *Hub) removeClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.clients[client.userID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.clients, client.userID)
		}
	}
}

func (h *Hub) send(userID uuid.UUID, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients[userID] {
		select {
		case client.send <- payload:
		default:
			// Закрываем клиент асинхронно с panic recovery
			go func(c *Client) {
				defer func() {
					if r := recover(); r != nil {
						fmt.Printf("WebSocket client close panic recovered: %v\nStack trace:\n%s\n", r, debug.Stack())
					}
				}()
				c.Close()
			}(client)
		}
	}
}
