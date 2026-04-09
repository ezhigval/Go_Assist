package files

import "time"

// FileRef ссылка на сохранённый объект.
type FileRef struct {
	ID        string
	LocalPath string
	Size      int64
	MIME      string
	Name      string
	CreatedAt time.Time
}

// TransportFilePayload контракт v1.transport.file.received (бот шлёт байты/meta).
type TransportFilePayload struct {
	FileName string
	MIME     string
	Data     []byte
	ChatID   int64
	UserID   string
}
