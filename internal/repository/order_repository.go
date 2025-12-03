package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/ignatzorin/freelance-backend/internal/models"
)

// OrderRepository отвечает за работу с заказами и откликами.
type OrderRepository struct {
	db *sqlx.DB
}

// Ошибки уровня репозитория.
var (
	ErrOrderNotFound        = errors.New("order not found")
	ErrProposalNotFound     = errors.New("proposal not found")
	ErrConversationNotFound = errors.New("conversation not found")
)

// NewOrderRepository создаёт новый экземпляр.
func NewOrderRepository(db *sqlx.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// GetByID возвращает заказ по идентификатору.
func (r *OrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Order, error) {
	var order models.Order
	query := `
		SELECT id, client_id, freelancer_id, category_id, title, description, budget_min, budget_max, final_amount, status, deadline_at, ai_summary,
		       best_recommendation_proposal_id, best_recommendation_justification, ai_analysis_updated_at,
		       created_at, updated_at
		FROM orders
		WHERE id = $1
	`
	if err := r.db.GetContext(ctx, &order, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("order repository: get by id %w", err)
	}
	return &order, nil
}

// GetByIDWithDetails возвращает заказ со всеми связанными данными (требования и вложения) одним запросом.
// Оптимизированная версия для избежания N+1 проблем.
func (r *OrderRepository) GetByIDWithDetails(ctx context.Context, id uuid.UUID) (*models.Order, []models.OrderRequirement, []models.OrderAttachment, error) {
	var order models.Order
	orderQuery := `
		SELECT id, client_id, freelancer_id, category_id, title, description, budget_min, budget_max, final_amount, status, deadline_at, ai_summary,
		       best_recommendation_proposal_id, best_recommendation_justification, ai_analysis_updated_at,
		       created_at, updated_at
		FROM orders
		WHERE id = $1
	`
	if err := r.db.GetContext(ctx, &order, orderQuery, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, nil, ErrOrderNotFound
		}
		return nil, nil, nil, fmt.Errorf("order repository: get by id %w", err)
	}

	// Загружаем требования и вложения параллельно (или последовательно, но одним запросом каждый)
	var requirements []models.OrderRequirement
	reqQuery := `SELECT id, order_id, skill, level FROM order_requirements WHERE order_id = $1 ORDER BY skill`
	if err := r.db.SelectContext(ctx, &requirements, reqQuery, id); err != nil {
		return nil, nil, nil, fmt.Errorf("order repository: get requirements %w", err)
	}

	// Загружаем вложения с JOIN для получения информации о медиа
	var attachments []models.OrderAttachment
	query := `
		SELECT
			oa.id,
			oa.order_id,
			oa.media_id,
			oa.created_at,
			mf.id,
			mf.user_id,
			mf.file_path,
			mf.file_type,
			mf.file_size,
			mf.is_public,
			mf.created_at
		FROM order_attachments oa
		JOIN media_files mf ON mf.id = oa.media_id
		WHERE oa.order_id = $1
		ORDER BY oa.created_at
	`

	rows, err := r.db.QueryxContext(ctx, query, id)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("order repository: get attachments %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var attachment models.OrderAttachment
		var media models.MediaFile
		var mediaUserID *uuid.UUID

		if err := rows.Scan(
			&attachment.ID,
			&attachment.OrderID,
			&attachment.MediaID,
			&attachment.CreatedAt,
			&media.ID,
			&mediaUserID,
			&media.FilePath,
			&media.FileType,
			&media.FileSize,
			&media.IsPublic,
			&media.CreatedAt,
		); err != nil {
			return nil, nil, nil, fmt.Errorf("order repository: scan attachment %w", err)
		}

		media.UserID = mediaUserID
		attachment.Media = &media
		attachments = append(attachments, attachment)
	}

	return &order, requirements, attachments, nil
}

// Create сохраняет заказ и связанные требования/вложения в одной транзакции.
func (r *OrderRepository) Create(ctx context.Context, order *models.Order, requirements []models.OrderRequirement, attachmentIDs []uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("order repository: begin tx %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	query := `
		INSERT INTO orders (client_id, title, description, budget_min, budget_max, status, deadline_at, ai_summary)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`

	if err = tx.QueryRowxContext(
		ctx,
		query,
		order.ClientID,
		order.Title,
		order.Description,
		order.BudgetMin,
		order.BudgetMax,
		order.Status,
		order.DeadlineAt,
		order.AISummary,
	).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt); err != nil {
		return fmt.Errorf("order repository: insert order %w", err)
	}

	if len(requirements) > 0 {
		// Batch INSERT для requirements (устранение N+1)
		reqQuery := `INSERT INTO order_requirements (order_id, skill, level) VALUES `
		reqValues := make([]interface{}, 0, len(requirements)*3)

		for i, req := range requirements {
			if i > 0 {
				reqQuery += ", "
			}
			reqQuery += fmt.Sprintf("($%d, $%d, $%d)", i*3+1, i*3+2, i*3+3)
			reqValues = append(reqValues, order.ID, req.Skill, req.Level)
		}

		if _, err = tx.ExecContext(ctx, reqQuery, reqValues...); err != nil {
			return fmt.Errorf("order repository: batch insert requirements %w", err)
		}
	}

	if len(attachmentIDs) > 0 {
		// Batch INSERT для attachments (устранение N+1)
		attQuery := `INSERT INTO order_attachments (order_id, media_id) VALUES `
		attValues := make([]interface{}, 0, len(attachmentIDs)*2)

		for i, mediaID := range attachmentIDs {
			if i > 0 {
				attQuery += ", "
			}
			attQuery += fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2)
			attValues = append(attValues, order.ID, mediaID)
		}
		attQuery += " ON CONFLICT DO NOTHING"

		if _, err = tx.ExecContext(ctx, attQuery, attValues...); err != nil {
			return fmt.Errorf("order repository: batch insert attachments %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("order repository: commit %w", err)
	}

	return nil
}

// Update изменяет заказ и его требования/вложения.
func (r *OrderRepository) Update(ctx context.Context, order *models.Order, requirements []models.OrderRequirement, attachmentIDs []uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("order repository: begin tx %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	query := `
		UPDATE orders
		SET title = $1,
		    description = $2,
		    budget_min = $3,
		    budget_max = $4,
		    status = $5::order_status,
		    deadline_at = $6,
		    ai_summary = $7,
		    freelancer_id = $8,
		    updated_at = NOW()
		WHERE id = $9 AND client_id = $10
		RETURNING updated_at
	`

	var updatedAt time.Time
	err = tx.QueryRowxContext(
		ctx,
		query,
		order.Title,
		order.Description,
		order.BudgetMin,
		order.BudgetMax,
		order.Status,
		order.DeadlineAt,
		order.AISummary,
		order.FreelancerID,
		order.ID,
		order.ClientID,
	).Scan(&updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrOrderNotFound
		}
		return fmt.Errorf("order repository: update order %w", err)
	}
	order.UpdatedAt = updatedAt

	if _, err = tx.ExecContext(ctx, `DELETE FROM order_requirements WHERE order_id = $1`, order.ID); err != nil {
		return fmt.Errorf("order repository: clear requirements %w", err)
	}

	if len(requirements) > 0 {
		// Batch INSERT для requirements (устранение N+1)
		reqQuery := `INSERT INTO order_requirements (order_id, skill, level) VALUES `
		reqValues := make([]interface{}, 0, len(requirements)*3)

		for i, req := range requirements {
			if i > 0 {
				reqQuery += ", "
			}
			reqQuery += fmt.Sprintf("($%d, $%d, $%d)", i*3+1, i*3+2, i*3+3)
			reqValues = append(reqValues, order.ID, req.Skill, req.Level)
		}

		if _, err = tx.ExecContext(ctx, reqQuery, reqValues...); err != nil {
			return fmt.Errorf("order repository: batch insert requirements %w", err)
		}
	}

	if _, err = tx.ExecContext(ctx, `DELETE FROM order_attachments WHERE order_id = $1`, order.ID); err != nil {
		return fmt.Errorf("order repository: clear attachments %w", err)
	}

	if len(attachmentIDs) > 0 {
		// Batch INSERT для attachments (устранение N+1)
		attQuery := `INSERT INTO order_attachments (order_id, media_id) VALUES `
		attValues := make([]interface{}, 0, len(attachmentIDs)*2)

		for i, mediaID := range attachmentIDs {
			if i > 0 {
				attQuery += ", "
			}
			attQuery += fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2)
			attValues = append(attValues, order.ID, mediaID)
		}
		attQuery += " ON CONFLICT DO NOTHING"

		if _, err = tx.ExecContext(ctx, attQuery, attValues...); err != nil {
			return fmt.Errorf("order repository: batch insert attachments %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("order repository: commit %w", err)
	}

	return nil
}

// ListFilterParams содержит параметры фильтрации и поиска заказов.
type ListFilterParams struct {
	Status    string
	Search    string
	Skills    []string
	BudgetMin *float64
	BudgetMax *float64
	SortBy    string // "date", "budget", "proposals"
	SortOrder string // "asc", "desc"
	Limit     int
	Offset    int
}

// ListResult содержит список заказов и метаданные пагинации.
type ListResult struct {
	Orders  []models.Order
	Total   int
	Limit   int
	Offset  int
	HasMore bool
}

// List возвращает список заказов с пагинацией, фильтрацией и поиском.
func (r *OrderRepository) List(ctx context.Context, params ListFilterParams) (*ListResult, error) {
	// Базовый запрос для подсчёта общего количества
	countQuery := `
		SELECT COUNT(DISTINCT o.id)
		FROM orders o
		LEFT JOIN order_requirements or_req ON o.id = or_req.order_id
		WHERE 1=1
	`

	// Запрос для получения данных с подсчетом предложений
	query := `
		SELECT DISTINCT o.*,
			COALESCE(proposal_counts.count, 0) as proposals_count
		FROM orders o
		LEFT JOIN order_requirements or_req ON o.id = or_req.order_id
		LEFT JOIN (
			SELECT order_id, COUNT(*) as count
			FROM proposals
			GROUP BY order_id
		) proposal_counts ON o.id = proposal_counts.order_id
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	// Применяем фильтры к обоим запросам
	// Фильтр: показываем только заказы, где исполнитель еще не определен
	// (не in_progress, не completed, и нет accepted proposals)
	excludeClause := `
		AND o.status NOT IN ('in_progress', 'completed')
		AND NOT EXISTS (
			SELECT 1 FROM proposals p 
			WHERE p.order_id = o.id AND p.status = 'accepted'
		)
	`
	query += excludeClause
	countQuery += excludeClause

	// Фильтр по статусу
	if params.Status != "" {
		clause := fmt.Sprintf(" AND o.status = $%d::order_status", argIndex)
		query += clause
		countQuery += clause
		args = append(args, params.Status)
		argIndex++
	}

	// Поиск по тексту
	if params.Search != "" {
		clause := fmt.Sprintf(" AND (o.title ILIKE $%d OR o.description ILIKE $%d)", argIndex, argIndex)
		query += clause
		countQuery += clause
		args = append(args, "%"+params.Search+"%")
		argIndex++
	}

	// Фильтр по навыкам
	if len(params.Skills) > 0 {
		clause := fmt.Sprintf(" AND or_req.skill = ANY($%d)", argIndex)
		query += clause
		countQuery += clause
		args = append(args, pq.Array(params.Skills))
		argIndex++
	}

	// Фильтр по бюджету
	if params.BudgetMin != nil {
		clause := fmt.Sprintf(" AND (o.budget_max IS NULL OR o.budget_max >= $%d)", argIndex)
		query += clause
		countQuery += clause
		args = append(args, *params.BudgetMin)
		argIndex++
	}
	if params.BudgetMax != nil {
		clause := fmt.Sprintf(" AND (o.budget_min IS NULL OR o.budget_min <= $%d)", argIndex)
		query += clause
		countQuery += clause
		args = append(args, *params.BudgetMax)
		argIndex++
	}

	// Сортировка
	sortBy := params.SortBy
	if sortBy == "" {
		sortBy = "date"
	}
	sortOrder := params.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	switch sortBy {
	case "budget":
		query += fmt.Sprintf(" ORDER BY COALESCE(o.budget_min, o.budget_max, 0) %s", sortOrder)
	case "proposals":
		query += `
			ORDER BY (
				SELECT COUNT(*) FROM proposals p WHERE p.order_id = o.id
			) ` + sortOrder
	default: // "date"
		query += fmt.Sprintf(" ORDER BY o.created_at %s", sortOrder)
	}

	// Подсчитываем общее количество
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, fmt.Errorf("order repository: count %w", err)
	}

	// Пагинация
	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := params.Offset
	if offset < 0 {
		offset = 0
	}

	query += fmt.Sprintf(" LIMIT $%d", argIndex)
	args = append(args, limit)
	argIndex++
	query += fmt.Sprintf(" OFFSET $%d", argIndex)
	args = append(args, offset)
	argIndex++

	// Используем структуру для маппинга с proposals_count
	type OrderWithCount struct {
		models.Order
		ProposalsCount *int `db:"proposals_count"`
	}

	var ordersWithCount []OrderWithCount
	if err := r.db.SelectContext(ctx, &ordersWithCount, query, args...); err != nil {
		return nil, fmt.Errorf("order repository: list %w", err)
	}

	// Преобразуем обратно в models.Order
	orders := make([]models.Order, len(ordersWithCount))
	for i, oc := range ordersWithCount {
		orders[i] = oc.Order
		orders[i].ProposalsCount = oc.ProposalsCount
	}

	hasMore := offset+limit < total

	return &ListResult{
		Orders:  orders,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: hasMore,
	}, nil
}

// CreateProposal добавляет отклик и возвращает его идентификатор.
func (r *OrderRepository) CreateProposal(ctx context.Context, proposal *models.Proposal) error {
	query := `
		INSERT INTO proposals (order_id, freelancer_id, cover_letter, proposed_amount, status, ai_feedback)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	return r.db.QueryRowxContext(
		ctx,
		query,
		proposal.OrderID,
		proposal.FreelancerID,
		proposal.CoverLetter,
		proposal.ProposedAmount,
		proposal.Status,
		proposal.AIFeedback,
	).Scan(&proposal.ID, &proposal.CreatedAt, &proposal.UpdatedAt)
}

// GetProposalByID возвращает отклик по идентификатору.
func (r *OrderRepository) GetProposalByID(ctx context.Context, id uuid.UUID) (*models.Proposal, error) {
	var proposal models.Proposal
	if err := r.db.GetContext(ctx, &proposal, `SELECT * FROM proposals WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProposalNotFound
		}
		return nil, fmt.Errorf("order repository: get proposal %w", err)
	}
	return &proposal, nil
}

// UpdateProposalStatus меняет статус отклика.
func (r *OrderRepository) UpdateProposalStatus(ctx context.Context, id uuid.UUID, status string) (*models.Proposal, error) {
	query := `
		UPDATE proposals
		SET status = $2,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, order_id, freelancer_id, cover_letter, proposed_amount, status, ai_feedback, created_at, updated_at
	`

	var proposal models.Proposal
	if err := r.db.QueryRowxContext(ctx, query, id, status).StructScan(&proposal); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProposalNotFound
		}
		return nil, fmt.Errorf("order repository: update proposal status %w", err)
	}
	return &proposal, nil
}

// ListProposals возвращает отклики по заказу.
func (r *OrderRepository) ListProposals(ctx context.Context, orderID uuid.UUID) ([]models.Proposal, error) {
	query := `
		SELECT * FROM proposals
		WHERE order_id = $1
		ORDER BY created_at DESC
	`

	var proposals []models.Proposal
	if err := r.db.SelectContext(ctx, &proposals, query, orderID); err != nil {
		return nil, fmt.Errorf("order repository: list proposals %w", err)
	}

	return proposals, nil
}

// CreateConversation создаёт чат для заказа.
func (r *OrderRepository) CreateConversation(ctx context.Context, conv *models.Conversation) error {
	query := `
		INSERT INTO conversations (order_id, client_id, freelancer_id)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`

	return r.db.QueryRowxContext(
		ctx,
		query,
		conv.OrderID,
		conv.ClientID,
		conv.FreelancerID,
	).Scan(&conv.ID, &conv.CreatedAt)
}

// GetConversationByParticipants возвращает чат между клиентом и исполнителем.
func (r *OrderRepository) GetConversationByParticipants(ctx context.Context, orderID uuid.UUID, clientID, freelancerID uuid.UUID) (*models.Conversation, error) {
	var conv models.Conversation
	err := r.db.GetContext(
		ctx,
		&conv,
		`SELECT * FROM conversations WHERE order_id = $1 AND client_id = $2 AND freelancer_id = $3`,
		orderID,
		clientID,
		freelancerID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrConversationNotFound
		}
		return nil, fmt.Errorf("order repository: get conversation %w", err)
	}
	return &conv, nil
}

// GetConversationByID возвращает чат по ID.
func (r *OrderRepository) GetConversationByID(ctx context.Context, id uuid.UUID) (*models.Conversation, error) {
	var conv models.Conversation
	if err := r.db.GetContext(ctx, &conv, `SELECT * FROM conversations WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrConversationNotFound
		}
		return nil, fmt.Errorf("order repository: get conversation by id %w", err)
	}
	return &conv, nil
}

// ListMyConversations возвращает все чаты пользователя (как клиента и как исполнителя).
// Возвращает только чаты, где есть accepted proposal (т.е. активные чаты).
func (r *OrderRepository) ListMyConversations(ctx context.Context, userID uuid.UUID) ([]models.Conversation, error) {
	query := `
		SELECT DISTINCT c.*
		FROM conversations c
		INNER JOIN proposals p ON c.order_id = p.order_id 
			AND c.freelancer_id = p.freelancer_id
			AND p.status = 'accepted'
		WHERE (c.client_id = $1 OR c.freelancer_id = $1)
		ORDER BY c.created_at DESC
	`
	var conversations []models.Conversation
	if err := r.db.SelectContext(ctx, &conversations, query, userID); err != nil {
		return nil, fmt.Errorf("order repository: list my conversations %w", err)
	}
	return conversations, nil
}

// GetLastMessageForConversation возвращает последнее сообщение в чате.
func (r *OrderRepository) GetLastMessageForConversation(ctx context.Context, conversationID uuid.UUID) (*models.Message, error) {
	var message models.Message
	query := `
		SELECT * FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`
	if err := r.db.GetContext(ctx, &message, query, conversationID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Нет сообщений - это нормально
		}
		return nil, fmt.Errorf("order repository: get last message %w", err)
	}
	return &message, nil
}

// AddMessage добавляет сообщение в чат.
func (r *OrderRepository) AddMessage(ctx context.Context, msg *models.Message) error {
	query := `
		INSERT INTO messages (conversation_id, author_type, author_id, content, parent_message_id, ai_metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	var metadata interface{}
	if len(msg.AIMetadata) == 0 {
		metadata = nil
	} else {
		metadata = string(msg.AIMetadata)
	}

	var updatedAt time.Time
	err := r.db.QueryRowxContext(
		ctx,
		query,
		msg.ConversationID,
		msg.AuthorType,
		msg.AuthorID,
		msg.Content,
		msg.ParentMessageID,
		metadata,
	).Scan(&msg.ID, &msg.CreatedAt, &updatedAt)
	if err != nil {
		return err
	}
	msg.UpdatedAt = &updatedAt
	return nil
}

// ListMessages возвращает сообщения чата с пагинацией.
// Сообщения возвращаются в хронологическом порядке (старые первыми).
func (r *OrderRepository) ListMessages(ctx context.Context, conversationID uuid.UUID, limit, offset int) ([]models.Message, error) {
	query := `
		SELECT * FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC
	`
	args := []interface{}{conversationID}
	argIndex := 2

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
		argIndex++
	}

	if offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, offset)
	}

	var messages []models.Message
	if err := r.db.SelectContext(ctx, &messages, query, args...); err != nil {
		return nil, fmt.Errorf("order repository: list messages %w", err)
	}

	// Загружаем вложения и реакции для всех сообщений
	if len(messages) > 0 {
		messageIDs := make([]uuid.UUID, len(messages))
		for i := range messages {
			messageIDs[i] = messages[i].ID
		}

		// Загружаем вложения
		attachments, err := r.GetMessageAttachmentsByMessageIDs(ctx, messageIDs)
		if err == nil {
			attachmentsMap := make(map[uuid.UUID][]models.MessageAttachment)
			for _, att := range attachments {
				attachmentsMap[att.MessageID] = append(attachmentsMap[att.MessageID], att)
			}
			for i := range messages {
				messages[i].Attachments = attachmentsMap[messages[i].ID]
			}
		}

		// Загружаем реакции
		reactions, err := r.GetMessageReactionsByMessageIDs(ctx, messageIDs)
		if err == nil {
			reactionsMap := make(map[uuid.UUID][]models.MessageReaction)
			for _, react := range reactions {
				reactionsMap[react.MessageID] = append(reactionsMap[react.MessageID], react)
			}
			for i := range messages {
				messages[i].Reactions = reactionsMap[messages[i].ID]
			}
		}
	}

	return messages, nil
}

// GetMessageByID возвращает сообщение по идентификатору.
func (r *OrderRepository) GetMessageByID(ctx context.Context, messageID uuid.UUID) (*models.Message, error) {
	var message models.Message
	if err := r.db.GetContext(ctx, &message, `SELECT * FROM messages WHERE id = $1`, messageID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("order repository: message not found")
		}
		return nil, fmt.Errorf("order repository: get message by id %w", err)
	}

	// Загружаем вложения
	attachments, err := r.GetMessageAttachments(ctx, messageID)
	if err == nil {
		message.Attachments = attachments
	}

	// Загружаем реакции
	reactions, err := r.GetMessageReactions(ctx, messageID)
	if err == nil {
		message.Reactions = reactions
	}

	return &message, nil
}

// UpdateMessage обновляет содержимое сообщения.
func (r *OrderRepository) UpdateMessage(ctx context.Context, messageID uuid.UUID, newContent string) error {
	result, err := r.db.ExecContext(ctx, `UPDATE messages SET content = $1, updated_at = NOW() WHERE id = $2`, newContent, messageID)
	if err != nil {
		return fmt.Errorf("order repository: update message %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("order repository: update message rows affected %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("order repository: message not found")
	}

	return nil
}

// DeleteMessage мягко удаляет сообщение (устанавливает content в "[Сообщение удалено]").
func (r *OrderRepository) DeleteMessage(ctx context.Context, messageID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `UPDATE messages SET content = '[Сообщение удалено]' WHERE id = $1`, messageID)
	if err != nil {
		return fmt.Errorf("order repository: delete message %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("order repository: delete message rows affected %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("order repository: message not found")
	}

	return nil
}

// ListAttachments возвращает вложения заказа.
func (r *OrderRepository) ListAttachments(ctx context.Context, orderID uuid.UUID) ([]models.OrderAttachment, error) {
	query := `
		SELECT
			oa.id,
			oa.order_id,
			oa.media_id,
			oa.created_at,
			mf.id,
			mf.user_id,
			mf.file_path,
			mf.file_type,
			mf.file_size,
			mf.is_public,
			mf.created_at
		FROM order_attachments oa
		JOIN media_files mf ON mf.id = oa.media_id
		WHERE oa.order_id = $1
		ORDER BY oa.created_at
	`

	rows, err := r.db.QueryxContext(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("order repository: list attachments %w", err)
	}
	defer rows.Close()

	var attachments []models.OrderAttachment

	for rows.Next() {
		var attachment models.OrderAttachment
		var media models.MediaFile
		var mediaUserID *uuid.UUID

		if err := rows.Scan(
			&attachment.ID,
			&attachment.OrderID,
			&attachment.MediaID,
			&attachment.CreatedAt,
			&media.ID,
			&mediaUserID,
			&media.FilePath,
			&media.FileType,
			&media.FileSize,
			&media.IsPublic,
			&media.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("order repository: scan attachment %w", err)
		}

		media.UserID = mediaUserID
		attachment.Media = &media
		attachments = append(attachments, attachment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("order repository: attachments rows %w", err)
	}

	return attachments, nil
}

// GetUserOrderStats возвращает статистику заказов пользователя.
func (r *OrderRepository) GetUserOrderStats(ctx context.Context, userID uuid.UUID) (map[string]int, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'published') as open,
			COUNT(*) FILTER (WHERE status = 'in_progress') as in_progress,
			COUNT(*) FILTER (WHERE status = 'completed') as completed,
			COALESCE(SUM(proposal_count), 0) as total_proposals
		FROM orders
		LEFT JOIN (
			SELECT order_id, COUNT(*) as proposal_count
			FROM proposals
			GROUP BY order_id
		) p ON orders.id = p.order_id
		WHERE orders.client_id = $1
	`

	var result struct {
		Total          int `db:"total"`
		Open           int `db:"open"`
		InProgress     int `db:"in_progress"`
		Completed      int `db:"completed"`
		TotalProposals int `db:"total_proposals"`
	}

	if err := r.db.GetContext(ctx, &result, query, userID); err != nil {
		return nil, fmt.Errorf("order repository: get user order stats %w", err)
	}

	return map[string]int{
		"total":           result.Total,
		"open":            result.Open,
		"in_progress":     result.InProgress,
		"completed":       result.Completed,
		"total_proposals": result.TotalProposals,
	}, nil
}

// GetUserProposalStats возвращает статистику предложений пользователя.
func (r *OrderRepository) GetUserProposalStats(ctx context.Context, userID uuid.UUID) (map[string]int, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'pending') as pending,
			COUNT(*) FILTER (WHERE status = 'accepted') as accepted,
			COUNT(*) FILTER (WHERE status = 'rejected') as rejected
		FROM proposals
		WHERE freelancer_id = $1
	`

	var result struct {
		Total    int `db:"total"`
		Pending  int `db:"pending"`
		Accepted int `db:"accepted"`
		Rejected int `db:"rejected"`
	}

	if err := r.db.GetContext(ctx, &result, query, userID); err != nil {
		return nil, fmt.Errorf("order repository: get user proposal stats %w", err)
	}

	return map[string]int{
		"total":    result.Total,
		"pending":  result.Pending,
		"accepted": result.Accepted,
		"rejected": result.Rejected,
	}, nil
}

// ListMyProposals возвращает все предложения текущего пользователя.
func (r *OrderRepository) ListMyProposals(ctx context.Context, userID uuid.UUID) ([]models.Proposal, error) {
	query := `
		SELECT * FROM proposals
		WHERE freelancer_id = $1
		ORDER BY created_at DESC
	`

	var proposals []models.Proposal
	if err := r.db.SelectContext(ctx, &proposals, query, userID); err != nil {
		return nil, fmt.Errorf("order repository: list my proposals %w", err)
	}

	return proposals, nil
}

// GetMyProposalForOrder возвращает предложение пользователя для конкретного заказа.
func (r *OrderRepository) GetMyProposalForOrder(ctx context.Context, orderID, freelancerID uuid.UUID) (*models.Proposal, error) {
	var proposal models.Proposal
	query := `
		SELECT * FROM proposals
		WHERE order_id = $1 AND freelancer_id = $2
		ORDER BY created_at DESC
		LIMIT 1
	`

	if err := r.db.GetContext(ctx, &proposal, query, orderID, freelancerID); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrProposalNotFound
		}
		return nil, fmt.Errorf("order repository: get my proposal for order %w", err)
	}

	return &proposal, nil
}

// ListRequirements возвращает требования к заказу.
func (r *OrderRepository) ListRequirements(ctx context.Context, orderID uuid.UUID) ([]models.OrderRequirement, error) {
	query := `
		SELECT id, order_id, skill, level
		FROM order_requirements
		WHERE order_id = $1
		ORDER BY skill
	`

	var requirements []models.OrderRequirement
	if err := r.db.SelectContext(ctx, &requirements, query, orderID); err != nil {
		return nil, fmt.Errorf("order repository: list requirements %w", err)
	}

	return requirements, nil
}

// ListMyOrders возвращает все заказы текущего пользователя (как заказчика и как исполнителя).
func (r *OrderRepository) ListMyOrders(ctx context.Context, userID uuid.UUID) ([]models.Order, []models.Order, error) {
	// Заказы как заказчик с подсчетом предложений
	clientQuery := `
		SELECT o.*,
			COALESCE(proposal_counts.count, 0) as proposals_count
		FROM orders o
		LEFT JOIN (
			SELECT order_id, COUNT(*) as count
			FROM proposals
			GROUP BY order_id
		) proposal_counts ON o.id = proposal_counts.order_id
		WHERE o.client_id = $1
		ORDER BY o.created_at DESC
	`
	type OrderWithCount struct {
		models.Order
		ProposalsCount *int `db:"proposals_count"`
	}
	var clientOrdersWithCount []OrderWithCount
	if err := r.db.SelectContext(ctx, &clientOrdersWithCount, clientQuery, userID); err != nil {
		return nil, nil, fmt.Errorf("order repository: list client orders %w", err)
	}
	clientOrders := make([]models.Order, len(clientOrdersWithCount))
	for i, oc := range clientOrdersWithCount {
		clientOrders[i] = oc.Order
		clientOrders[i].ProposalsCount = oc.ProposalsCount
	}

	// Заказы как исполнитель (где есть принятый отклик или заказ в работе)
	freelancerQuery := `
		SELECT DISTINCT o.*,
			COALESCE(proposal_counts.count, 0) as proposals_count
		FROM orders o
		INNER JOIN proposals p ON o.id = p.order_id
		LEFT JOIN (
			SELECT order_id, COUNT(*) as count
			FROM proposals
			GROUP BY order_id
		) proposal_counts ON o.id = proposal_counts.order_id
		WHERE p.freelancer_id = $1 AND (p.status = 'accepted' OR o.status = 'in_progress')
		ORDER BY o.created_at DESC
	`
	var freelancerOrdersWithCount []OrderWithCount
	if err := r.db.SelectContext(ctx, &freelancerOrdersWithCount, freelancerQuery, userID); err != nil {
		return nil, nil, fmt.Errorf("order repository: list freelancer orders %w", err)
	}
	freelancerOrders := make([]models.Order, len(freelancerOrdersWithCount))
	for i, oc := range freelancerOrdersWithCount {
		freelancerOrders[i] = oc.Order
		freelancerOrders[i].ProposalsCount = oc.ProposalsCount
	}

	return clientOrders, freelancerOrders, nil
}

// Delete удаляет заказ по идентификатору.
func (r *OrderRepository) Delete(ctx context.Context, id uuid.UUID, clientID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM orders WHERE id = $1 AND client_id = $2`, id, clientID)
	if err != nil {
		return fmt.Errorf("order repository: delete %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("order repository: delete rows affected %w", err)
	}

	if rowsAffected == 0 {
		return ErrOrderNotFound
	}

	return nil
}

// UpdateAISummary обновляет только AI summary заказа.
func (r *OrderRepository) UpdateAISummary(ctx context.Context, orderID uuid.UUID, clientID uuid.UUID, summary string) error {
	query := `
		UPDATE orders 
		SET ai_summary = $1, updated_at = NOW() 
		WHERE id = $2 AND client_id = $3
	`
	result, err := r.db.ExecContext(ctx, query, summary, orderID, clientID)
	if err != nil {
		return fmt.Errorf("order repository: update ai summary %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("order repository: update ai summary rows affected %w", err)
	}

	if rowsAffected == 0 {
		return ErrOrderNotFound
	}

	return nil
}

// UpdateProposalAIFeedback обновляет AI feedback для отклика.
func (r *OrderRepository) UpdateProposalAIFeedback(ctx context.Context, proposalID uuid.UUID, feedback string) error {
	query := `
		UPDATE proposals 
		SET ai_feedback = $1, updated_at = NOW() 
		WHERE id = $2
	`
	_, err := r.db.ExecContext(ctx, query, feedback, proposalID)
	if err != nil {
		return fmt.Errorf("order repository: update proposal ai feedback %w", err)
	}
	return nil
}

// UpdateBestRecommendation обновляет рекомендацию лучшего исполнителя для заказа.
func (r *OrderRepository) UpdateBestRecommendation(ctx context.Context, orderID uuid.UUID, proposalID *uuid.UUID, justification string) error {
	query := `
		UPDATE orders 
		SET best_recommendation_proposal_id = $1,
		    best_recommendation_justification = $2,
		    ai_analysis_updated_at = NOW(),
		    updated_at = NOW() 
		WHERE id = $3
	`
	_, err := r.db.ExecContext(ctx, query, proposalID, justification, orderID)
	if err != nil {
		return fmt.Errorf("order repository: update best recommendation %w", err)
	}
	return nil
}

// GetProposalsLastUpdateTime возвращает время последнего обновления откликов для заказа.
func (r *OrderRepository) GetProposalsLastUpdateTime(ctx context.Context, orderID uuid.UUID) (*time.Time, error) {
	var lastUpdate time.Time
	query := `
		SELECT COALESCE(MAX(updated_at), MAX(created_at)) 
		FROM proposals 
		WHERE order_id = $1
	`
	err := r.db.GetContext(ctx, &lastUpdate, query, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("order repository: get proposals last update time %w", err)
	}
	return &lastUpdate, nil
}

// GetAverageResponseTimeHours возвращает среднее время ответа в часах для пользователя.
// Для клиентов: среднее время от создания предложения до его принятия.
// Для фрилансеров: среднее время от создания предложения до его принятия.
func (r *OrderRepository) GetAverageResponseTimeHours(ctx context.Context, userID uuid.UUID) (float64, error) {
	var avgHours float64
	query := `
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (p.updated_at - p.created_at)) / 3600), 0) as avg_hours
		FROM proposals p
		INNER JOIN orders o ON p.order_id = o.id
		WHERE p.status = 'accepted' 
		AND (o.client_id = $1 OR p.freelancer_id = $1)
		AND p.updated_at > p.created_at
	`
	err := r.db.GetContext(ctx, &avgHours, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0.0, nil
		}
		return 0.0, fmt.Errorf("order repository: get average response time %w", err)
	}
	return avgHours, nil
}

// AddMessageAttachments добавляет вложения к сообщению.
func (r *OrderRepository) AddMessageAttachments(ctx context.Context, messageID uuid.UUID, mediaIDs []uuid.UUID) error {
	if len(mediaIDs) == 0 {
		return nil
	}

	query := `INSERT INTO message_attachments (message_id, media_id) VALUES `
	values := make([]interface{}, 0, len(mediaIDs)*2)
	argIndex := 1

	for i, mediaID := range mediaIDs {
		if i > 0 {
			query += ", "
		}
		query += fmt.Sprintf("($%d, $%d)", argIndex, argIndex+1)
		values = append(values, messageID, mediaID)
		argIndex += 2
	}
	query += " ON CONFLICT DO NOTHING"

	_, err := r.db.ExecContext(ctx, query, values...)
	if err != nil {
		return fmt.Errorf("order repository: add message attachments %w", err)
	}
	return nil
}

// GetMessageAttachments возвращает вложения сообщения.
func (r *OrderRepository) GetMessageAttachments(ctx context.Context, messageID uuid.UUID) ([]models.MessageAttachment, error) {
	query := `
		SELECT
			ma.id,
			ma.message_id,
			ma.media_id,
			ma.created_at,
			mf.id,
			mf.user_id,
			mf.file_path,
			mf.file_type,
			mf.file_size,
			mf.is_public,
			mf.created_at
		FROM message_attachments ma
		JOIN media_files mf ON mf.id = ma.media_id
		WHERE ma.message_id = $1
		ORDER BY ma.created_at
	`

	rows, err := r.db.QueryxContext(ctx, query, messageID)
	if err != nil {
		return nil, fmt.Errorf("order repository: get message attachments %w", err)
	}
	defer rows.Close()

	var attachments []models.MessageAttachment
	for rows.Next() {
		var attachment models.MessageAttachment
		var media models.MediaFile
		var mediaUserID *uuid.UUID

		if err := rows.Scan(
			&attachment.ID,
			&attachment.MessageID,
			&attachment.MediaID,
			&attachment.CreatedAt,
			&media.ID,
			&mediaUserID,
			&media.FilePath,
			&media.FileType,
			&media.FileSize,
			&media.IsPublic,
			&media.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("order repository: scan message attachment %w", err)
		}

		media.UserID = mediaUserID
		attachment.Media = &media
		attachments = append(attachments, attachment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("order repository: message attachments rows %w", err)
	}

	return attachments, nil
}

// GetMessageAttachmentsByMessageIDs возвращает вложения для нескольких сообщений.
func (r *OrderRepository) GetMessageAttachmentsByMessageIDs(ctx context.Context, messageIDs []uuid.UUID) ([]models.MessageAttachment, error) {
	if len(messageIDs) == 0 {
		return []models.MessageAttachment{}, nil
	}

	query := `
		SELECT
			ma.id,
			ma.message_id,
			ma.media_id,
			ma.created_at,
			mf.id,
			mf.user_id,
			mf.file_path,
			mf.file_type,
			mf.file_size,
			mf.is_public,
			mf.created_at
		FROM message_attachments ma
		JOIN media_files mf ON mf.id = ma.media_id
		WHERE ma.message_id = ANY($1)
		ORDER BY ma.message_id, ma.created_at
	`

	rows, err := r.db.QueryxContext(ctx, query, pq.Array(messageIDs))
	if err != nil {
		return nil, fmt.Errorf("order repository: get message attachments by ids %w", err)
	}
	defer rows.Close()

	var attachments []models.MessageAttachment
	for rows.Next() {
		var attachment models.MessageAttachment
		var media models.MediaFile
		var mediaUserID *uuid.UUID

		if err := rows.Scan(
			&attachment.ID,
			&attachment.MessageID,
			&attachment.MediaID,
			&attachment.CreatedAt,
			&media.ID,
			&mediaUserID,
			&media.FilePath,
			&media.FileType,
			&media.FileSize,
			&media.IsPublic,
			&media.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("order repository: scan message attachment %w", err)
		}

		media.UserID = mediaUserID
		attachment.Media = &media
		attachments = append(attachments, attachment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("order repository: message attachments rows %w", err)
	}

	return attachments, nil
}

// AddMessageReaction добавляет реакцию на сообщение.
func (r *OrderRepository) AddMessageReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) (*models.MessageReaction, error) {
	query := `
		INSERT INTO message_reactions (message_id, user_id, emoji)
		VALUES ($1, $2, $3)
		ON CONFLICT (message_id, user_id) 
		DO UPDATE SET emoji = EXCLUDED.emoji, created_at = NOW()
		RETURNING id, message_id, user_id, emoji, created_at
	`

	var reaction models.MessageReaction
	if err := r.db.QueryRowxContext(ctx, query, messageID, userID, emoji).StructScan(&reaction); err != nil {
		return nil, fmt.Errorf("order repository: add message reaction %w", err)
	}

	return &reaction, nil
}

// RemoveMessageReaction удаляет реакцию пользователя на сообщение.
func (r *OrderRepository) RemoveMessageReaction(ctx context.Context, messageID, userID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM message_reactions WHERE message_id = $1 AND user_id = $2`, messageID, userID)
	if err != nil {
		return fmt.Errorf("order repository: remove message reaction %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("order repository: remove message reaction rows affected %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("order repository: reaction not found")
	}

	return nil
}

// GetMessageReactions возвращает реакции на сообщение.
func (r *OrderRepository) GetMessageReactions(ctx context.Context, messageID uuid.UUID) ([]models.MessageReaction, error) {
	query := `
		SELECT * FROM message_reactions
		WHERE message_id = $1
		ORDER BY created_at
	`

	var reactions []models.MessageReaction
	if err := r.db.SelectContext(ctx, &reactions, query, messageID); err != nil {
		return nil, fmt.Errorf("order repository: get message reactions %w", err)
	}

	return reactions, nil
}

// GetMessageReactionsByMessageIDs возвращает реакции для нескольких сообщений.
func (r *OrderRepository) GetMessageReactionsByMessageIDs(ctx context.Context, messageIDs []uuid.UUID) ([]models.MessageReaction, error) {
	if len(messageIDs) == 0 {
		return []models.MessageReaction{}, nil
	}

	query := `
		SELECT * FROM message_reactions
		WHERE message_id = ANY($1)
		ORDER BY message_id, created_at
	`

	var reactions []models.MessageReaction
	if err := r.db.SelectContext(ctx, &reactions, query, pq.Array(messageIDs)); err != nil {
		return nil, fmt.Errorf("order repository: get message reactions by ids %w", err)
	}

	return reactions, nil
}

// SetOrderFreelancer устанавливает фрилансера и итоговую сумму для заказа.
func (r *OrderRepository) SetOrderFreelancer(ctx context.Context, orderID, freelancerID uuid.UUID, finalAmount float64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE orders SET freelancer_id = $2, final_amount = $3, updated_at = NOW() WHERE id = $1
	`, orderID, freelancerID, finalAmount)
	return err
}
