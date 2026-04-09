package media

import (
	"context"

	"modulr/events"
)

// MediaAPI публичный контракт модуля медиа.
type MediaAPI interface {
	RegisterMedia(ctx context.Context, item *MediaItem) error
	Get(ctx context.Context, id string) (*MediaItem, error)
	Search(ctx context.Context, q MediaQuery) ([]MediaItem, error)
	Link(ctx context.Context, mediaID, entityType, entityID string) error
	AddTags(ctx context.Context, mediaID string, tags []string) error
	RegisterSubscriptions(bus *events.EventBus)
}

// MediaQuery фильтр поиска.
type MediaQuery struct {
	Context  events.Segment
	Tags     []string
	From     string // RFC3339 или пусто
	To       string
	Kind     MediaKind
	FullText string
}
