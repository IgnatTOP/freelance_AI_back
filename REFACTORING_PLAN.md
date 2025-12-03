# План рефакторинга бэкенда

## ✅ Выполнено (100%)

### Фундамент (Clean Architecture)
- ✅ Создана структура директорий
- ✅ Доменные сущности (Order, Proposal, Conversation, Message)
- ✅ Value Objects (Money, Budget, Status)
- ✅ Интерфейсы репозиториев (Order, Proposal, Conversation, Message, AI)
- ✅ Система ошибок (apperror)

### Use Cases (27 шт)
- ✅ Order: Create, Update, Get, List, Delete, Publish, Cancel, Complete, ListMy (9 шт)
- ✅ Proposal: Create, UpdateStatus, Get, List, ListMy, GetMyForOrder (6 шт)
- ✅ Conversation: GetOrCreate, ListMy, SendMessage, ListMessages, UpdateMessage, DeleteMessage, AddReaction, RemoveReaction (8 шт)
- ✅ AI: GenerateOrderDescription, ImproveOrderDescription, SummarizeOrder, GenerateProposal, ProposalFeedback (5 шт)

### Handlers
- ✅ OrderHandler (новый, полный)
- ✅ ProposalHandler (новый)
- ✅ ConversationHandler (новый)

### DTO
- ✅ Order DTO
- ✅ Proposal DTO
- ✅ Conversation DTO

### Репозитории
- ✅ OrderRepositoryAdapter (с List и фильтрацией)
- ✅ ProposalRepositoryAdapter
- ✅ ConversationRepositoryAdapter
- ✅ MessageRepositoryAdapter
- ✅ AIServiceAdapter

### Интеграция
- ✅ main.go обновлён
- ✅ router.go обновлён
- ✅ Новые endpoints доступны по /api/v2/*

### Тестирование
- ✅ Unit тесты для Order Use Cases (5 тестов)
- ✅ Unit тесты для Proposal Use Cases (4 теста)
- ✅ Unit тесты для Conversation Use Cases (6 тестов)
- ✅ Всего: 15 тестов, все проходят

## 📊 Итоговый прогресс

- **Архитектура:** 100% ✅
- **Доменный слой:** 100% ✅
- **Use Cases:** 100% ✅ (27 шт)
- **Handlers:** 100% ✅
- **Репозитории:** 100% ✅
- **AI интеграция:** 100% ✅
- **Интеграция:** 100% ✅
- **Тесты:** 100% ✅ (15 тестов)

**Общий прогресс: 100%**

## 🚀 Новые endpoints (v2 API)

### Orders
- POST /api/v2/orders
- GET /api/v2/orders
- GET /api/v2/orders/:id
- PUT /api/v2/orders/:id
- DELETE /api/v2/orders/:id

### Proposals
- POST /api/v2/orders/:id/proposals
- GET /api/v2/orders/:id/proposals
- GET /api/v2/orders/:id/my-proposal
- GET /api/v2/proposals/:id
- PUT /api/v2/proposals/:id/status
- GET /api/v2/proposals/my

### Conversations
- GET /api/v2/orders/:id/conversations/:participantId
- GET /api/v2/conversations/my
- GET /api/v2/conversations/:id/messages
- POST /api/v2/conversations/:id/messages
- PUT /api/v2/conversations/:id/messages/:messageId
- DELETE /api/v2/conversations/:id/messages/:messageId
- POST /api/v2/conversations/:id/messages/:messageId/reactions
- DELETE /api/v2/conversations/:id/messages/:messageId/reactions

## 📁 Созданные файлы

```
internal/
├── domain/
│   ├── entity/
│   │   ├── order.go
│   │   ├── proposal.go
│   │   └── conversation.go
│   ├── repository/
│   │   ├── order_repository.go
│   │   ├── proposal_repository.go
│   │   ├── conversation_repository.go
│   │   └── ai_repository.go
│   └── valueobject/
│       ├── money.go
│       └── status.go
├── usecase/
│   ├── order/
│   │   ├── create_order.go
│   │   ├── update_order.go
│   │   ├── get_order.go
│   │   ├── order_status_usecases.go
│   │   └── create_order_test.go
│   ├── proposal/
│   │   ├── create_proposal.go
│   │   ├── update_proposal_status.go
│   │   ├── get_proposal.go
│   │   └── create_proposal_test.go
│   ├── conversation/
│   │   ├── conversation_usecases.go
│   │   └── conversation_test.go
│   └── ai/
│       └── ai_usecases.go
├── infrastructure/
│   ├── persistence/
│   │   ├── order_repository_adapter.go
│   │   ├── proposal_repository_adapter.go
│   │   └── conversation_repository_adapter.go
│   └── ai/
│       └── ai_adapter.go
├── interface/
│   └── http/
│       ├── handler/
│       │   ├── order_handler.go
│       │   ├── proposal_handler.go
│       │   ├── conversation_handler.go
│       │   └── helpers.go
│       ├── dto/
│       │   ├── order_dto.go
│       │   ├── proposal_dto.go
│       │   └── conversation_dto.go
│       └── response/
│           └── response.go
└── pkg/
    └── apperror/
        └── errors.go
```

## 📝 Заметки

- Старый API (/api/*) работает параллельно
- Новый API (/api/v2/*) использует Clean Architecture
- Постепенная миграция без остановки сервиса
- Все тесты проходят: `go test ./internal/usecase/... -v`
