package files

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"modulr/events"
)

// Service реализует FilesAPI и подписан на v1.transport.file.received.
type Service struct {
	cfg  Config
	bus  *events.Bus
	idem events.IdempotencyStore

	mu      sync.Mutex
	started bool
}

// NewService создаёт модуль files.
func NewService(cfg Config, bus *events.Bus, idem events.IdempotencyStore) *Service {
	return &Service{cfg: cfg, bus: bus, idem: idem}
}

// Start создаёт каталог и подписки.
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return nil
	}
	if err := os.MkdirAll(s.cfg.LocalDir, 0o755); err != nil {
		return fmt.Errorf("files: mkdir: %w", err)
	}
	if s.bus != nil {
		s.bus.Subscribe(events.V1TransportFileRecv, s.onTransportFile)
	}
	s.started = true
	log.Println("files: module started")
	return nil
}

func (s *Service) onTransportFile(evt events.Event) {
	if s.idem != nil && evt.ID != "" && s.idem.Seen("file-"+evt.ID) {
		return
	}
	p, ok := evt.Payload.(TransportFilePayload)
	if !ok {
		if m, ok2 := evt.Payload.(map[string]any); ok2 {
			p = transportFromMap(m)
			ok = p.Data != nil && len(p.Data) > 0
		}
	}
	if !ok {
		return
	}
	ctx := context.Background()
	ref, err := s.Store(ctx, p.FileName, p.MIME, p.Data)
	if err != nil {
		log.Printf("files: store failed: %v", err)
		return
	}
	if s.idem != nil && evt.ID != "" {
		s.idem.MarkSeen("file-" + evt.ID)
	}
	if s.bus != nil {
		s.bus.Publish(events.Event{
			Name:    events.V1FilesUploaded,
			Payload: ref,
			Source:  "files",
			TraceID: evt.TraceID,
			Context: map[string]any{
				"chat_id": p.ChatID,
				"user_id": p.UserID,
			},
		})
	}
}

func transportFromMap(m map[string]any) TransportFilePayload {
	p := TransportFilePayload{}
	if v, ok := m["file_name"].(string); ok {
		p.FileName = v
	}
	if v, ok := m["mime"].(string); ok {
		p.MIME = v
	}
	if v, ok := m["data"].([]byte); ok {
		p.Data = v
	}
	if v, ok := m["chat_id"].(int64); ok {
		p.ChatID = v
	}
	if v, ok := m["user_id"].(string); ok {
		p.UserID = v
	}
	return p
}

// Store сохраняет байты на диск (локальный backend).
func (s *Service) Store(ctx context.Context, name, mime string, data []byte) (*FileRef, error) {
	_ = ctx
	if len(data) == 0 {
		return nil, fmt.Errorf("files: empty data")
	}
	sum := sha256.Sum256(data)
	id := hex.EncodeToString(sum[:8])
	if name == "" {
		name = id
	}
	sub := time.Now().UTC().Format("2006/01/02")
	dir := filepath.Join(s.cfg.LocalDir, sub)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, id+"_"+filepath.Base(name))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return nil, err
	}
	ref := &FileRef{
		ID:        id,
		LocalPath: path,
		Size:      int64(len(data)),
		MIME:      mime,
		Name:      name,
		CreatedAt: time.Now(),
	}
	return ref, nil
}

// Stop заглушка (файлы на диске остаются).
func (s *Service) Stop() error {
	log.Println("files: module stopped")
	return nil
}
