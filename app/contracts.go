package app

import (
	"strings"

	coreevents "modulr/core/events"
	"modulr/events"
)

// Канонические action/control event names для поддерживаемого runtime path v1.0.
const (
	EventTrackerCreateReminder    events.Name = "v1.tracker.create_reminder"
	EventTrackerCreateTask        events.Name = "v1.tracker.create_task"
	EventFinanceCreateTxn         events.Name = "v1.finance.create_transaction"
	EventKnowledgeSaveQuery       events.Name = "v1.knowledge.save_query"
	EventKnowledgeSaveNote        events.Name = "v1.knowledge.save_note"
	EventOrchestratorOutcome      events.Name = events.Name(coreevents.V1OrchestratorDecisionOutcome)
	EventOrchestratorFallback     events.Name = events.Name(coreevents.V1OrchestratorFallback)
	EventTransportResponseTimeout events.Name = "v1.transport.response.timeout"
)

// HumanAction возвращает transport-friendly описание поддерживаемого action event.
func HumanAction(name string) string {
	switch events.Name(name) {
	case EventTrackerCreateReminder:
		return "создано напоминание"
	case EventTrackerCreateTask:
		return "создана задача"
	case EventFinanceCreateTxn:
		return "зарегистрирована транзакция"
	case EventKnowledgeSaveQuery:
		return "запрос сохранён в knowledge"
	case EventKnowledgeSaveNote:
		return "заметка сохранена в knowledge"
	default:
		if strings.TrimSpace(name) == "" {
			return "действие выполнено"
		}
		return name
	}
}
