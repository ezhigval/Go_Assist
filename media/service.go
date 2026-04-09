package media

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"modulr/events"
)

// Service реализует MediaAPI.
type Service struct {
	storage events.Storage
	bus     *events.EventBus
}

// NewService создаёт сервис медиа.
func NewService(st events.Storage, bus *events.EventBus) *Service {
	return &Service{storage: st, bus: bus}
}

// RegisterSubscriptions: внешние файлы → метаданные media.
func (s *Service) RegisterSubscriptions(bus *events.EventBus) {
	if bus == nil {
		return
	}
	s.bus = bus
	bus.Subscribe(events.V1FilesUploaded, s.onFilesUploaded)
}

func (s *Service) onFilesUploaded(evt events.Event) {
	ctx := context.Background()
	m, ok := anyToMap(evt.Payload)
	if !ok {
		return
	}
	ref, _ := m["local_path"].(string)
	if ref == "" {
		ref, _ = m["LocalPath"].(string)
	}
	id, _ := m["id"].(string)
	if id == "" {
		id, _ = m["ID"].(string)
	}
	item := &MediaItem{
		Kind:       KindPhoto,
		StorageRef: ref,
	}
	item.ID = id
	if item.ID == "" {
		item.ID = fmt.Sprintf("med_%d", time.Now().UnixNano())
	}
	item.Context = events.ParseSegmentFromAny(m["scope"])
	if item.Context == "" && evt.Context != nil {
		item.Context = events.ParseSegmentFromAny(evt.Context["scope"])
	}
	if item.Context == "" {
		item.Context = events.DefaultSegment()
	}
	item.Tags = []string{"ingest", "files_bridge"}
	item.Meta = Metadata{FileID: item.ID, MIME: strOr(m["mime"], strOr(m["MIME"], "application/octet-stream"))}
	if sz, ok := asInt64(m["size"]); ok {
		item.Meta.Size = sz
	}
	if sz, ok := asInt64(m["Size"]); ok && item.Meta.Size == 0 {
		item.Meta.Size = sz
	}
	item.LinkedTo = map[string]string{}
	item.EntityBase.CreatedAt = time.Now()
	item.EntityBase.UpdatedAt = time.Now()
	if err := s.RegisterMedia(ctx, item); err != nil {
		log.Printf("media: files.uploaded→register: %v", err)
	}
}

func strOr(v any, def string) string {
	s, _ := v.(string)
	if s == "" {
		return def
	}
	return s
}

func asInt64(v any) (int64, bool) {
	switch x := v.(type) {
	case int64:
		return x, true
	case float64:
		return int64(x), true
	case int:
		return int64(x), true
	default:
		return 0, false
	}
}

func anyToMap(p any) (map[string]any, bool) {
	if m, ok := p.(map[string]any); ok {
		return m, true
	}
	b, err := json.Marshal(p)
	if err != nil {
		return nil, false
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, false
	}
	return m, true
}

// RegisterMedia сохраняет медиа и публикует uploaded.
func (s *Service) RegisterMedia(ctx context.Context, item *MediaItem) error {
	if item == nil {
		return fmt.Errorf("media: nil item")
	}
	if item.ID == "" {
		item.ID = fmt.Sprintf("med_%d", time.Now().UnixNano())
	}
	now := time.Now()
	item.CreatedAt = now
	item.UpdatedAt = now
	if item.LinkedTo == nil {
		item.LinkedTo = make(map[string]string)
	}
	key := "media:item:" + item.ID
	if err := s.storage.PutJSON(ctx, key, item); err != nil {
		return err
	}
	if s.bus != nil {
		s.bus.Publish(events.Event{
			Name:    EventUploaded,
			Payload: item,
			Source:  "media",
			Context: map[string]any{"segment": string(item.Context)},
		})
	}
	return nil
}

// Get возвращает медиа по id.
func (s *Service) Get(ctx context.Context, id string) (*MediaItem, error) {
	var item MediaItem
	if err := s.storage.GetJSON(ctx, "media:item:"+id, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

// Search фильтрует по тегам/сегменту/дате/типу.
func (s *Service) Search(ctx context.Context, q MediaQuery) ([]MediaItem, error) {
	keys, err := s.storage.ListPrefix(ctx, "media:item:")
	if err != nil {
		return nil, err
	}
	var out []MediaItem
	for _, k := range keys {
		var it MediaItem
		if err := s.storage.GetJSON(ctx, k, &it); err != nil {
			continue
		}
		if q.Context != "" && it.Context != q.Context {
			continue
		}
		if q.Kind != "" && it.Kind != q.Kind {
			continue
		}
		if len(q.Tags) > 0 && !hasAllTags(it.Tags, q.Tags) {
			continue
		}
		if q.FullText != "" && !strings.Contains(strings.ToLower(it.Meta.FileID+" "+it.StorageRef), strings.ToLower(q.FullText)) {
			continue
		}
		out = append(out, it)
	}
	return out, nil
}

func hasAllTags(have []string, need []string) bool {
	set := make(map[string]struct{}, len(have))
	for _, t := range have {
		set[strings.ToLower(t)] = struct{}{}
	}
	for _, n := range need {
		if _, ok := set[strings.ToLower(n)]; !ok {
			return false
		}
	}
	return true
}

// Link привязывает медиа к сущности и публикует linked.
func (s *Service) Link(ctx context.Context, mediaID, entityType, entityID string) error {
	it, err := s.Get(ctx, mediaID)
	if err != nil {
		return err
	}
	it.LinkedTo[entityType] = entityID
	it.UpdatedAt = time.Now()
	if err := s.storage.PutJSON(ctx, "media:item:"+it.ID, it); err != nil {
		return err
	}
	if s.bus != nil {
		s.bus.Publish(events.Event{
			Name:    EventLinked,
			Payload: map[string]any{"media_id": mediaID, "entity_type": entityType, "entity_id": entityID},
			Source:  "media",
		})
	}
	return nil
}

// AddTags добавляет теги и публикует tagged.
func (s *Service) AddTags(ctx context.Context, mediaID string, tags []string) error {
	it, err := s.Get(ctx, mediaID)
	if err != nil {
		return err
	}
	it.Tags = append(it.Tags, tags...)
	it.UpdatedAt = time.Now()
	if err := s.storage.PutJSON(ctx, "media:item:"+it.ID, it); err != nil {
		return err
	}
	if s.bus != nil {
		s.bus.Publish(events.Event{
			Name:    EventTagged,
			Payload: map[string]any{"media_id": mediaID, "tags": tags},
			Source:  "media",
		})
	}
	return nil
}
