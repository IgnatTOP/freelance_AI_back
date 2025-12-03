package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/ignatzorin/freelance-backend/internal/domain/entity"
	"github.com/ignatzorin/freelance-backend/internal/domain/repository"
	"github.com/ignatzorin/freelance-backend/internal/domain/valueobject"
	"github.com/ignatzorin/freelance-backend/internal/pkg/apperror"
	oldRepo "github.com/ignatzorin/freelance-backend/internal/repository"
	"github.com/jmoiron/sqlx"
)

type OrderRepositoryAdapter struct {
	db      *sqlx.DB
	oldRepo *oldRepo.OrderRepository
}

func NewOrderRepositoryAdapter(db *sqlx.DB) *OrderRepositoryAdapter {
	return &OrderRepositoryAdapter{
		db:      db,
		oldRepo: oldRepo.NewOrderRepository(db),
	}
}

func (r *OrderRepositoryAdapter) Create(ctx context.Context, order *entity.Order) error {
	query := `
		INSERT INTO orders (id, client_id, title, description, budget_min, budget_max, status, deadline_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	
	_, err := r.db.ExecContext(ctx, query,
		order.ID,
		order.ClientID,
		order.Title,
		order.Description,
		order.Budget.Min.Amount,
		order.Budget.Max.Amount,
		string(order.Status),
		order.DeadlineAt,
		order.CreatedAt,
		order.UpdatedAt,
	)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось создать заказ")
	}
	
	for _, req := range order.Requirements {
		if err := r.CreateRequirement(ctx, &req); err != nil {
			return err
		}
	}
	
	for _, att := range order.Attachments {
		if err := r.CreateAttachment(ctx, &att); err != nil {
			return err
		}
	}
	
	return nil
}

func (r *OrderRepositoryAdapter) Update(ctx context.Context, order *entity.Order) error {
	query := `
		UPDATE orders 
		SET title = $2, description = $3, budget_min = $4, budget_max = $5, 
		    status = $6, deadline_at = $7, ai_summary = $8, 
		    best_recommendation_proposal_id = $9, best_recommendation_justification = $10,
		    ai_analysis_updated_at = $11, freelancer_id = $12, updated_at = $13
		WHERE id = $1
	`
	
	_, err := r.db.ExecContext(ctx, query,
		order.ID,
		order.Title,
		order.Description,
		order.Budget.Min.Amount,
		order.Budget.Max.Amount,
		string(order.Status),
		order.DeadlineAt,
		order.AISummary,
		order.BestRecommendationProposalID,
		order.BestRecommendationJustification,
		order.AIAnalysisUpdatedAt,
		order.FreelancerID,
		order.UpdatedAt,
	)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось обновить заказ")
	}
	
	return nil
}

func (r *OrderRepositoryAdapter) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM orders WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось удалить заказ")
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось проверить результат удаления")
	}
	
	if rows == 0 {
		return apperror.ErrOrderNotFound
	}
	
	return nil
}

func (r *OrderRepositoryAdapter) FindByID(ctx context.Context, id uuid.UUID) (*entity.Order, error) {
	var order entity.Order
	var budgetMin, budgetMax float64
	var status string
	
	query := `
		SELECT id, client_id, freelancer_id, title, description, budget_min, budget_max, status, deadline_at, 
		       ai_summary, best_recommendation_proposal_id, best_recommendation_justification, 
		       ai_analysis_updated_at, created_at, updated_at
		FROM orders
		WHERE id = $1
	`
	
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&order.ID,
		&order.ClientID,
		&order.FreelancerID,
		&order.Title,
		&order.Description,
		&budgetMin,
		&budgetMax,
		&status,
		&order.DeadlineAt,
		&order.AISummary,
		&order.BestRecommendationProposalID,
		&order.BestRecommendationJustification,
		&order.AIAnalysisUpdatedAt,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, apperror.ErrOrderNotFound
	}
	if err != nil {
		return nil, apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось получить заказ")
	}
	
	budget, _ := valueobject.NewBudget(budgetMin, budgetMax)
	order.Budget = budget
	
	orderStatus, _ := valueobject.NewOrderStatus(status)
	order.Status = orderStatus
	
	return &order, nil
}

func (r *OrderRepositoryAdapter) FindByIDWithDetails(ctx context.Context, id uuid.UUID) (*entity.Order, error) {
	order, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	
	requirements, err := r.FindRequirements(ctx, id)
	if err != nil {
		return nil, err
	}
	order.Requirements = requirements
	
	attachments, err := r.FindAttachments(ctx, id)
	if err != nil {
		return nil, err
	}
	order.Attachments = attachments
	
	return order, nil
}

func (r *OrderRepositoryAdapter) FindByClientID(ctx context.Context, clientID uuid.UUID) ([]*entity.Order, error) {
	query := `
		SELECT id, client_id, freelancer_id, title, description, budget_min, budget_max, status, deadline_at, 
		       ai_summary, best_recommendation_proposal_id, best_recommendation_justification, 
		       ai_analysis_updated_at, created_at, updated_at
		FROM orders
		WHERE client_id = $1
		ORDER BY created_at DESC
	`
	
	rows, err := r.db.QueryContext(ctx, query, clientID)
	if err != nil {
		return nil, apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось получить заказы")
	}
	defer rows.Close()
	
	var orders []*entity.Order
	for rows.Next() {
		var order entity.Order
		var budgetMin, budgetMax float64
		var status string
		
		err := rows.Scan(
			&order.ID,
			&order.ClientID,
			&order.FreelancerID,
			&order.Title,
			&order.Description,
			&budgetMin,
			&budgetMax,
			&status,
			&order.DeadlineAt,
			&order.AISummary,
			&order.BestRecommendationProposalID,
			&order.BestRecommendationJustification,
			&order.AIAnalysisUpdatedAt,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось прочитать заказ")
		}
		
		budget, _ := valueobject.NewBudget(budgetMin, budgetMax)
		order.Budget = budget
		
		orderStatus, _ := valueobject.NewOrderStatus(status)
		order.Status = orderStatus
		
		orders = append(orders, &order)
	}
	
	return orders, nil
}

func (r *OrderRepositoryAdapter) List(ctx context.Context, filter repository.OrderFilter) ([]*entity.Order, int, error) {
	baseQuery := `FROM orders WHERE 1=1`
	args := []interface{}{}
	argNum := 1

	if filter.Status != "" {
		baseQuery += fmt.Sprintf(" AND status = $%d", argNum)
		args = append(args, filter.Status)
		argNum++
	}

	if filter.Search != "" {
		baseQuery += fmt.Sprintf(" AND (title ILIKE $%d OR description ILIKE $%d)", argNum, argNum)
		args = append(args, "%"+filter.Search+"%")
		argNum++
	}

	if filter.BudgetMin != nil {
		baseQuery += fmt.Sprintf(" AND budget_max >= $%d", argNum)
		args = append(args, *filter.BudgetMin)
		argNum++
	}

	if filter.BudgetMax != nil {
		baseQuery += fmt.Sprintf(" AND budget_min <= $%d", argNum)
		args = append(args, *filter.BudgetMax)
		argNum++
	}

	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось посчитать заказы")
	}

	sortBy := "created_at"
	if filter.SortBy != "" {
		sortBy = filter.SortBy
	}
	sortOrder := "DESC"
	if filter.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	selectQuery := fmt.Sprintf(`SELECT id, client_id, freelancer_id, title, description, budget_min, budget_max, status, deadline_at, 
		ai_summary, best_recommendation_proposal_id, best_recommendation_justification, 
		ai_analysis_updated_at, created_at, updated_at %s ORDER BY %s %s LIMIT $%d OFFSET $%d`,
		baseQuery, sortBy, sortOrder, argNum, argNum+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось получить заказы")
	}
	defer rows.Close()

	var orders []*entity.Order
	for rows.Next() {
		var order entity.Order
		var budgetMin, budgetMax float64
		var status string

		err := rows.Scan(
			&order.ID, &order.ClientID, &order.FreelancerID, &order.Title, &order.Description,
			&budgetMin, &budgetMax, &status, &order.DeadlineAt,
			&order.AISummary, &order.BestRecommendationProposalID, &order.BestRecommendationJustification,
			&order.AIAnalysisUpdatedAt, &order.CreatedAt, &order.UpdatedAt,
		)
		if err != nil {
			return nil, 0, apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось прочитать заказ")
		}

		budget, _ := valueobject.NewBudget(budgetMin, budgetMax)
		order.Budget = budget
		orderStatus, _ := valueobject.NewOrderStatus(status)
		order.Status = orderStatus

		orders = append(orders, &order)
	}

	return orders, total, nil
}

func (r *OrderRepositoryAdapter) CreateRequirement(ctx context.Context, req *entity.OrderRequirement) error {
	query := `INSERT INTO order_requirements (id, order_id, skill, level) VALUES ($1, $2, $3, $4)`
	_, err := r.db.ExecContext(ctx, query, req.ID, req.OrderID, req.Skill, req.Level)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось создать требование")
	}
	return nil
}

func (r *OrderRepositoryAdapter) UpdateRequirements(ctx context.Context, orderID uuid.UUID, requirements []entity.OrderRequirement) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось начать транзакцию")
	}
	defer tx.Rollback()
	
	if _, err := tx.ExecContext(ctx, `DELETE FROM order_requirements WHERE order_id = $1`, orderID); err != nil {
		return apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось удалить старые требования")
	}
	
	for _, req := range requirements {
		if _, err := tx.ExecContext(ctx, `INSERT INTO order_requirements (id, order_id, skill, level) VALUES ($1, $2, $3, $4)`,
			req.ID, orderID, req.Skill, req.Level); err != nil {
			return apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось создать требование")
		}
	}
	
	return tx.Commit()
}

func (r *OrderRepositoryAdapter) FindRequirements(ctx context.Context, orderID uuid.UUID) ([]entity.OrderRequirement, error) {
	query := `SELECT id, order_id, skill, level FROM order_requirements WHERE order_id = $1 ORDER BY skill`
	
	var requirements []entity.OrderRequirement
	err := r.db.SelectContext(ctx, &requirements, query, orderID)
	if err != nil {
		return nil, apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось получить требования")
	}
	
	return requirements, nil
}

func (r *OrderRepositoryAdapter) CreateAttachment(ctx context.Context, att *entity.OrderAttachment) error {
	query := `INSERT INTO order_attachments (id, order_id, media_id, created_at) VALUES ($1, $2, $3, $4)`
	_, err := r.db.ExecContext(ctx, query, att.ID, att.OrderID, att.MediaID, att.CreatedAt)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось создать вложение")
	}
	return nil
}

func (r *OrderRepositoryAdapter) UpdateAttachments(ctx context.Context, orderID uuid.UUID, attachments []entity.OrderAttachment) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось начать транзакцию")
	}
	defer tx.Rollback()
	
	if _, err := tx.ExecContext(ctx, `DELETE FROM order_attachments WHERE order_id = $1`, orderID); err != nil {
		return apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось удалить старые вложения")
	}
	
	for _, att := range attachments {
		if _, err := tx.ExecContext(ctx, `INSERT INTO order_attachments (id, order_id, media_id, created_at) VALUES ($1, $2, $3, $4)`,
			att.ID, att.OrderID, att.MediaID, att.CreatedAt); err != nil {
			return apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось создать вложение")
		}
	}
	
	return tx.Commit()
}

func (r *OrderRepositoryAdapter) FindAttachments(ctx context.Context, orderID uuid.UUID) ([]entity.OrderAttachment, error) {
	query := `SELECT id, order_id, media_id, created_at FROM order_attachments WHERE order_id = $1 ORDER BY created_at`
	
	var attachments []entity.OrderAttachment
	err := r.db.SelectContext(ctx, &attachments, query, orderID)
	if err != nil {
		return nil, apperror.Wrap(err, apperror.ErrCodeDatabaseError, "не удалось получить вложения")
	}
	
	return attachments, nil
}
