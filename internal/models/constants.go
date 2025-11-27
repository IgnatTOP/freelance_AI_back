package models

// OrderStatus константы статусов заказов
const (
	OrderStatusDraft      = "draft"
	OrderStatusPublished  = "published"
	OrderStatusInProgress = "in_progress"
	OrderStatusCompleted  = "completed"
	OrderStatusCancelled  = "cancelled"
)

// ProposalStatus константы статусов предложений
const (
	ProposalStatusPending     = "pending"
	ProposalStatusShortlisted = "shortlisted"
	ProposalStatusAccepted    = "accepted"
	ProposalStatusRejected    = "rejected"
)

// ExperienceLevel константы уровней опыта
const (
	ExperienceLevelJunior = "junior"
	ExperienceLevelMiddle = "middle"
	ExperienceLevelSenior = "senior"
)

// ValidOrderStatuses список валидных статусов заказов
var ValidOrderStatuses = map[string]struct{}{
	OrderStatusDraft:      {},
	OrderStatusPublished:  {},
	OrderStatusInProgress: {},
	OrderStatusCompleted:  {},
	OrderStatusCancelled:  {},
}

// ValidProposalStatuses список валидных статусов предложений
var ValidProposalStatuses = map[string]struct{}{
	ProposalStatusPending:     {},
	ProposalStatusShortlisted: {},
	ProposalStatusAccepted:    {},
	ProposalStatusRejected:    {},
}

// ValidExperienceLevels список валидных уровней опыта
var ValidExperienceLevels = map[string]struct{}{
	ExperienceLevelJunior: {},
	ExperienceLevelMiddle: {},
	ExperienceLevelSenior: {},
}
