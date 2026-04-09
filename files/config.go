package files

import "os"

// Config хранилище файлов.
type Config struct {
	LocalDir string // корень на диске; позже S3 через другой Backend
}

// LoadConfig из окружения.
func LoadConfig() Config {
	dir := os.Getenv("FILES_LOCAL_DIR")
	if dir == "" {
		dir = "./data/files"
	}
	return Config{LocalDir: dir}
}
