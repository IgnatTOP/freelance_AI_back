package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Conversation описывает чат между клиентом и исполнителем.
type Conversation struct {
	ID           uuid.UUID  `db:"id" json:"id"`
	OrderID      *uuid.UUID `db:"order_id" json:"order_id,omitempty"`
	ClientID     uuid.UUID  `db:"client_id" json:"client_id"`
	FreelancerID uuid.UUID  `db:"freelancer_id" json:"freelancer_id"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
}

// Message описывает сообщение в чате.
type Message struct {
	ID             uuid.UUID  `db:"id" json:"id"`
	ConversationID uuid.UUID  `db:"conversation_id" json:"conversation_id"`
	AuthorType     string     `db:"author_type" json:"author_type"`
	AuthorID       *uuid.UUID `db:"author_id" json:"author_id,omitempty"`
	Content        string     `db:"content" json:"content"`
	ParentMessageID *uuid.UUID `db:"parent_message_id" json:"parent_message_id,omitempty"`
	AIMetadata     json.RawMessage `db:"ai_metadata" json:"ai_metadata,omitempty"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt      *time.Time `db:"updated_at" json:"updated_at,omitempty"`
	// Связанные данные (загружаются отдельно)
	Attachments    []MessageAttachment `json:"attachments,omitempty"`
	Reactions       []MessageReaction    `json:"reactions,omitempty"`
	ParentMessage   *Message             `json:"parent_message,omitempty"`
}

// MessageAttachment описывает вложение к сообщению.
type MessageAttachment struct {
	ID        uuid.UUID  `db:"id" json:"id"`
	MessageID uuid.UUID  `db:"message_id" json:"message_id"`
	MediaID   uuid.UUID  `db:"media_id" json:"media_id"`
	Media     *MediaFile `json:"media,omitempty"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
}

// MessageReaction описывает реакцию на сообщение.
type MessageReaction struct {
	ID        uuid.UUID `db:"id" json:"id"`
	MessageID uuid.UUID `db:"message_id" json:"message_id"`
	UserID    uuid.UUID `db:"user_id" json:"user_id"`
	Emoji     string   `db:"emoji" json:"emoji"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// Notification описывает событие, отправленное пользователю.
type Notification struct {
	ID        uuid.UUID       `db:"id" json:"id"`
	UserID    uuid.UUID       `db:"user_id" json:"user_id"`
	Payload   json.RawMessage `db:"payload" json:"payload"`
	IsRead    bool            `db:"is_read" json:"is_read"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
}

// AISession фиксирует взаимодействия пользователя с помощником.
type AISession struct {
	ID         uuid.UUID `db:"id" json:"id"`
	UserID     uuid.UUID `db:"user_id" json:"user_id"`
	Context    []byte    `db:"context" json:"context"`
	Suggestion string    `db:"suggestion" json:"suggestion"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

// ChatSummary создаёт краткое резюме переписки в чате.
type ChatSummary struct {
	Summary       string   `json:"summary"`
	NextSteps     []string `json:"next_steps"`
	Agreements    []string `json:"agreements"`
	OpenQuestions []string `json:"open_questions"`
}

// PriceTimelineRecommendation рекомендует цену и сроки для отклика.
type PriceTimelineRecommendation struct {
	RecommendedAmount *float64 `json:"recommended_amount"`
	MinAmount         *float64 `json:"min_amount"`
	MaxAmount         *float64 `json:"max_amount"`
	RecommendedDays  *int     `json:"recommended_days"`
	MinDays           *int     `json:"min_days"`
	MaxDays           *int     `json:"max_days"`
	Explanation       string   `json:"explanation"`
}

// OrderQualityEvaluation оценивает качество заказа и даёт рекомендации.
type OrderQualityEvaluation struct {
	Score          int      `json:"score"`
	Strengths      []string `json:"strengths"`
	Weaknesses     []string `json:"weaknesses"`
	Recommendations []string `json:"recommendations"`
}

// SuitableFreelancer находит подходящих фрилансеров для заказа.
type SuitableFreelancer struct {
	UserID      uuid.UUID `json:"user_id"`
	MatchScore  float64   `json:"match_score"`
	Explanation string    `json:"explanation"`
}

// RecommendedOrder описывает рекомендованный заказ с оценкой совпадения.
type RecommendedOrder struct {
	OrderID     uuid.UUID `json:"order_id"`
	MatchScore  float64   `json:"match_score"`
	Explanation string    `json:"explanation"`
}
