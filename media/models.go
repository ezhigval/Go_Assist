package media

import "modulr/events"

// MediaKind тип медиа.
type MediaKind string

const (
	KindPhoto MediaKind = "photo"
	KindVideo MediaKind = "video"
	KindAudio MediaKind = "audio"
)

// Metadata технические метаданные файла.
type Metadata struct {
	FileID   string `json:"file_id"`
	MIME     string `json:"mime"`
	Size     int64  `json:"size"`
	Duration int64  `json:"duration_sec,omitempty"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	Checksum string `json:"checksum,omitempty"`
}

// MediaItem зарегистрированный медиа-объект и связи с доменными сущностями.
type MediaItem struct {
	events.EntityBase
	Kind         MediaKind         `json:"kind"`
	Meta         Metadata          `json:"meta"`
	LinkedTo     map[string]string `json:"linked_to"` // тип сущности -> id
	StorageRef   string            `json:"storage_ref"`
	ThumbnailRef string            `json:"thumbnail_ref,omitempty"`
}
