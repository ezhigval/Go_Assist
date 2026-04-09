package organizer

import "context"

// NewOrganizerFacade создаёт готовый к использованию экземпляр органайзера
// Возвращает OrganizerAPI, который можно безопасно передавать в telegram, web, cli
func NewOrganizerFacade(storage Storage) OrganizerAPI {
	cfg := LoadConfig()
	org := NewOrganizer(cfg, storage)

	// Инициализация фоновых процессов (очистка, напоминания и т.д.)
	// STUB: Background workers require cancellable goroutines for reminders, external calendar sync, and batch AI analysis with organizer bus-only publishing.
	return org
}

// QuickStart вспомогательная функция для демо/тестов
func QuickStart(ctx context.Context) OrganizerAPI {
	memStorage := &InMemoryStorage{} // Простая реализация для теста
	api := NewOrganizerFacade(memStorage)
	api.Start(ctx)
	return api
}

// InMemoryStorage заглушка хранилища (заменится на databases.DB)
type InMemoryStorage struct{}

func (s *InMemoryStorage) Save(entity interface{}) error                          { return nil }
func (s *InMemoryStorage) GetByID(t EntityType, id string, out interface{}) error { return nil }
func (s *InMemoryStorage) List(t EntityType, out interface{}) error               { return nil }
func (s *InMemoryStorage) Delete(t EntityType, id string) error                   { return nil }
