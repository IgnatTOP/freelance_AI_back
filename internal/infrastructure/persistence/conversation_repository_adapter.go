package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/ignatzorin/freelance-backend/internal/domain/entity"
	"github.com/ignatzorin/freelance-backend/internal/pkg/apperror"
	"github.com/jmoiron/sqlx"
)

type ConversationRepositoryAdapter struct {
	db *sqlx.DB
}

func NewConversationRepositoryAdapter(db *sqlx.DB) *ConversationRepositoryAdapter {
	return &ConversationRepositoryAdapter{db: db}
}

func (r *ConversationRepositoryAdapter) Create(ctx context.Context, conv *entity.Conversation) error {
	query := `INSERT INTO conversations (id, order_id, client_id, freelancer_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.ExecContext(ctx, query, conv.ID, conv.OrderID, conv.ClientID, conv.FreelancerID, conv.CreatedAt, conv.UpdatedAt)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось создать беседу")
	}
	return nil
}

func (r *ConversationRepositoryAdapter) FindByID(ctx context.Context, id uuid.UUID) (*entity.Conversation, error) {
	var c conversationRow
	query := `SELECT id, order_id, client_id, freelancer_id, created_at, updated_at FROM conversations WHERE id = $1`
	if err := r.db.GetContext(ctx, &c, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, apperror.ErrConversationNotFound
		}
		return nil, apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось получить беседу")
	}
	return c.toEntity(), nil
}

func (r *ConversationRepositoryAdapter) FindByParticipants(ctx context.Context, orderID, clientID, freelancerID uuid.UUID) (*entity.Conversation, error) {
	var c conversationRow
	query := `SELECT id, order_id, client_id, freelancer_id, created_at, updated_at 
		FROM conversations WHERE order_id = $1 AND client_id = $2 AND freelancer_id = $3`
	if err := r.db.GetContext(ctx, &c, query, orderID, clientID, freelancerID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось получить беседу")
	}
	return c.toEntity(), nil
}

func (r *ConversationRepositoryAdapter) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Conversation, error) {
	var rows []conversationRow
	query := `SELECT id, order_id, client_id, freelancer_id, created_at, updated_at 
		FROM conversations WHERE client_id = $1 OR freelancer_id = $1 ORDER BY updated_at DESC`
	if err := r.db.SelectContext(ctx, &rows, query, userID); err != nil {
		return nil, apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось получить беседы")
	}
	result := make([]*entity.Conversation, len(rows))
	for i, row := range rows {
		result[i] = row.toEntity()
	}
	return result, nil
}

type conversationRow struct {
	ID           uuid.UUID `db:"id"`
	OrderID      uuid.UUID `db:"order_id"`
	ClientID     uuid.UUID `db:"client_id"`
	FreelancerID uuid.UUID `db:"freelancer_id"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

func (c *conversationRow) toEntity() *entity.Conversation {
	return &entity.Conversation{
		ID:           c.ID,
		OrderID:      c.OrderID,
		ClientID:     c.ClientID,
		FreelancerID: c.FreelancerID,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}

type MessageRepositoryAdapter struct {
	db *sqlx.DB
}

func NewMessageRepositoryAdapter(db *sqlx.DB) *MessageRepositoryAdapter {
	return &MessageRepositoryAdapter{db: db}
}

func (r *MessageRepositoryAdapter) Create(ctx context.Context, msg *entity.Message) error {
	query := `INSERT INTO messages (id, conversation_id, sender_id, content, is_edited, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.ExecContext(ctx, query, msg.ID, msg.ConversationID, msg.SenderID, msg.Content, msg.IsEdited, msg.CreatedAt, msg.UpdatedAt)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось создать сообщение")
	}
	return nil
}

func (r *MessageRepositoryAdapter) Update(ctx context.Context, msg *entity.Message) error {
	query := `UPDATE messages SET content = $2, is_edited = $3, updated_at = $4 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, msg.ID, msg.Content, msg.IsEdited, msg.UpdatedAt)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось обновить сообщение")
	}
	return nil
}

func (r *MessageRepositoryAdapter) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM messages WHERE id = $1`, id)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось удалить сообщение")
	}
	return nil
}

func (r *MessageRepositoryAdapter) FindByID(ctx context.Context, id uuid.UUID) (*entity.Message, error) {
	var m messageRow
	query := `SELECT id, conversation_id, sender_id, content, is_edited, created_at, updated_at FROM messages WHERE id = $1`
	if err := r.db.GetContext(ctx, &m, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, apperror.New(apperror.ErrCodeNotFound, "сообщение не найдено")
		}
		return nil, apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось получить сообщение")
	}
	return m.toEntity(), nil
}

func (r *MessageRepositoryAdapter) FindByConversationID(ctx context.Context, conversationID uuid.UUID, limit, offset int) ([]*entity.Message, error) {
	var rows []messageRow
	query := `SELECT id, conversation_id, sender_id, content, is_edited, created_at, updated_at 
		FROM messages WHERE conversation_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	if err := r.db.SelectContext(ctx, &rows, query, conversationID, limit, offset); err != nil {
		return nil, apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось получить сообщения")
	}
	result := make([]*entity.Message, len(rows))
	for i, row := range rows {
		result[i] = row.toEntity()
	}
	return result, nil
}

func (r *MessageRepositoryAdapter) GetLastMessage(ctx context.Context, conversationID uuid.UUID) (*entity.Message, error) {
	var m messageRow
	query := `SELECT id, conversation_id, sender_id, content, is_edited, created_at, updated_at 
		FROM messages WHERE conversation_id = $1 ORDER BY created_at DESC LIMIT 1`
	if err := r.db.GetContext(ctx, &m, query, conversationID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось получить сообщение")
	}
	return m.toEntity(), nil
}

func (r *MessageRepositoryAdapter) AddReaction(ctx context.Context, reaction *entity.MessageReaction) error {
	query := `INSERT INTO message_reactions (id, message_id, user_id, emoji, created_at) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (message_id, user_id) DO UPDATE SET emoji = $4`
	_, err := r.db.ExecContext(ctx, query, reaction.ID, reaction.MessageID, reaction.UserID, reaction.Emoji, reaction.CreatedAt)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось добавить реакцию")
	}
	return nil
}

func (r *MessageRepositoryAdapter) RemoveReaction(ctx context.Context, messageID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM message_reactions WHERE message_id = $1 AND user_id = $2`, messageID, userID)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось удалить реакцию")
	}
	return nil
}

func (r *MessageRepositoryAdapter) GetReactions(ctx context.Context, messageID uuid.UUID) ([]*entity.MessageReaction, error) {
	var rows []reactionRow
	query := `SELECT id, message_id, user_id, emoji, created_at FROM message_reactions WHERE message_id = $1`
	if err := r.db.SelectContext(ctx, &rows, query, messageID); err != nil {
		return nil, apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось получить реакции")
	}
	result := make([]*entity.MessageReaction, len(rows))
	for i, row := range rows {
		result[i] = &entity.MessageReaction{ID: row.ID, MessageID: row.MessageID, UserID: row.UserID, Emoji: row.Emoji, CreatedAt: row.CreatedAt}
	}
	return result, nil
}

type messageRow struct {
	ID             uuid.UUID `db:"id"`
	ConversationID uuid.UUID `db:"conversation_id"`
	SenderID       uuid.UUID `db:"sender_id"`
	Content        string    `db:"content"`
	IsEdited       bool      `db:"is_edited"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

func (m *messageRow) toEntity() *entity.Message {
	return &entity.Message{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		SenderID:       m.SenderID,
		Content:        m.Content,
		IsEdited:       m.IsEdited,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

type reactionRow struct {
	ID        uuid.UUID `db:"id"`
	MessageID uuid.UUID `db:"message_id"`
	UserID    uuid.UUID `db:"user_id"`
	Emoji     string    `db:"emoji"`
	CreatedAt time.Time `db:"created_at"`
}
