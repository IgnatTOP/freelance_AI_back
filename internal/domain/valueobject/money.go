package valueobject

import (
	"fmt"

	"github.com/ignatzorin/freelance-backend/internal/pkg/apperror"
)

type Money struct {
	Amount   float64
	Currency string
}

func NewMoney(amount float64, currency string) (Money, error) {
	if amount < 0 {
		return Money{}, apperror.New(apperror.ErrCodeValidation, "сумма не может быть отрицательной")
	}
	if currency == "" {
		currency = "USD"
	}
	return Money{Amount: amount, Currency: currency}, nil
}

type Budget struct {
	Min Money
	Max Money
}

func NewBudget(min, max float64) (Budget, error) {
	if min < 0 || max < 0 {
		return Budget{}, apperror.New(apperror.ErrCodeValidation, "бюджет не может быть отрицательным")
	}
	if min > max {
		return Budget{}, apperror.New(apperror.ErrCodeValidation, "минимальный бюджет не может превышать максимальный")
	}
	
	minMoney, _ := NewMoney(min, "USD")
	maxMoney, _ := NewMoney(max, "USD")
	
	return Budget{Min: minMoney, Max: maxMoney}, nil
}

func (b Budget) IsInRange(amount float64) bool {
	return amount >= b.Min.Amount && amount <= b.Max.Amount
}

func (b Budget) String() string {
	return fmt.Sprintf("%s %.2f - %.2f", b.Min.Currency, b.Min.Amount, b.Max.Amount)
}
