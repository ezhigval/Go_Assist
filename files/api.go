package files

import "context"

// FilesAPI публичный контракт модуля файлов.
type FilesAPI interface {
	Store(ctx context.Context, name, mime string, data []byte) (*FileRef, error)
	Start(ctx context.Context) error
	Stop() error
}
