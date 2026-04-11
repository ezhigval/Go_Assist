package databases

import (
	"context"
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	migrationsDir     = "migrations"
	migrationsLockKey = int64(2026041101)
)

var migrationFilePattern = regexp.MustCompile(`^(\d{6})_([a-z0-9_]+)\.(up|down)\.sql$`)

//go:embed migrations/*.sql
var embeddedMigrations embed.FS

const schemaMigrationsSQL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version BIGINT PRIMARY KEY,
	name TEXT NOT NULL,
	checksum VARCHAR(64) NOT NULL,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`

type migrationSpec struct {
	Version    int64
	Name       string
	UpSQL      string
	DownSQL    string
	UpChecksum string
}

type appliedMigration struct {
	Version   int64
	Name      string
	Checksum  string
	AppliedAt time.Time
}

// MigrationStatus описывает состояние миграции в раннере databases/.
type MigrationStatus struct {
	Version       int64
	Name          string
	Applied       bool
	AppliedAt     *time.Time
	ChecksumMatch bool
	MissingFile   bool
}

// State возвращает агрегированное состояние для CLI/логов.
func (s MigrationStatus) State() string {
	switch {
	case s.MissingFile:
		return "orphaned"
	case !s.Applied:
		return "pending"
	case !s.ChecksumMatch:
		return "drift"
	default:
		return "applied"
	}
}

// runMigrations применяет все versioned migrations; dev auto-migrate использует тот же раннер, что и deploy step.
func (db *DB) runMigrations(ctx context.Context) error {
	return db.ApplyMigrations(ctx)
}

// ApplyMigrations применяет все неподнятые миграции в порядке версий.
func (db *DB) ApplyMigrations(ctx context.Context) error {
	specs, err := loadMigrationSpecs(embeddedMigrations)
	if err != nil {
		return err
	}

	return db.withMigrationLock(ctx, "apply", func(conn *pgxpool.Conn) error {
		if err := ensureSchemaMigrationsTable(ctx, conn); err != nil {
			return err
		}

		applied, err := listAppliedMigrations(ctx, conn)
		if err != nil {
			return err
		}

		appliedByVersion := make(map[int64]appliedMigration, len(applied))
		for _, item := range applied {
			appliedByVersion[item.Version] = item
		}

		appliedCount := 0
		for _, spec := range specs {
			if current, ok := appliedByVersion[spec.Version]; ok {
				if current.Checksum != spec.UpChecksum {
					return fmt.Errorf("migration %06d_%s checksum drift: applied=%s current=%s", spec.Version, spec.Name, current.Checksum, spec.UpChecksum)
				}
				continue
			}

			if err := applyMigration(ctx, conn, spec); err != nil {
				return err
			}
			appliedCount++
		}

		if appliedCount == 0 {
			log.Println("🗄️  Migrations are up to date")
			return nil
		}

		log.Printf("✅ Applied %d migration(s)", appliedCount)
		return nil
	})
}

// RollbackMigrations откатывает последние N применённых миграций.
func (db *DB) RollbackMigrations(ctx context.Context, steps int) error {
	if steps <= 0 {
		return fmt.Errorf("rollback migrations: steps must be positive")
	}

	specs, err := loadMigrationSpecs(embeddedMigrations)
	if err != nil {
		return err
	}

	specsByVersion := make(map[int64]migrationSpec, len(specs))
	for _, spec := range specs {
		specsByVersion[spec.Version] = spec
	}

	return db.withMigrationLock(ctx, "rollback", func(conn *pgxpool.Conn) error {
		if err := ensureSchemaMigrationsTable(ctx, conn); err != nil {
			return err
		}

		applied, err := listAppliedMigrations(ctx, conn)
		if err != nil {
			return err
		}
		if len(applied) == 0 {
			log.Println("🗄️  No applied migrations to rollback")
			return nil
		}
		if steps > len(applied) {
			return fmt.Errorf("rollback migrations: requested %d step(s), but only %d migration(s) are applied", steps, len(applied))
		}

		for i := 0; i < steps; i++ {
			item := applied[len(applied)-1-i]
			spec, ok := specsByVersion[item.Version]
			if !ok {
				return fmt.Errorf("rollback migrations: applied migration %06d_%s is missing from embedded files", item.Version, item.Name)
			}
			if item.Checksum != spec.UpChecksum {
				return fmt.Errorf("rollback migrations: migration %06d_%s checksum drift detected", item.Version, item.Name)
			}
			if err := rollbackMigration(ctx, conn, spec); err != nil {
				return err
			}
		}

		log.Printf("↩️  Rolled back %d migration(s)", steps)
		return nil
	})
}

// ListMigrationStatus возвращает состояние известных и осиротевших миграций.
func (db *DB) ListMigrationStatus(ctx context.Context) ([]MigrationStatus, error) {
	specs, err := loadMigrationSpecs(embeddedMigrations)
	if err != nil {
		return nil, err
	}

	conn, err := db.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("list migration status: acquire connection: %w", err)
	}
	defer conn.Release()

	if err := ensureSchemaMigrationsTable(ctx, conn); err != nil {
		return nil, err
	}

	applied, err := listAppliedMigrations(ctx, conn)
	if err != nil {
		return nil, err
	}

	return buildMigrationStatuses(specs, applied), nil
}

func loadMigrationSpecs(fsys fs.FS) ([]migrationSpec, error) {
	entries, err := fs.ReadDir(fsys, migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("load migration specs: read dir %q: %w", migrationsDir, err)
	}

	type partialSpec struct {
		version int64
		name    string
		upSQL   string
		downSQL string
	}

	partials := make(map[int64]*partialSpec, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		matches := migrationFilePattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			return nil, fmt.Errorf("load migration specs: unexpected migration filename %q", entry.Name())
		}

		version, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("load migration specs: parse version from %q: %w", entry.Name(), err)
		}

		name := matches[2]
		direction := matches[3]
		contentBytes, err := fs.ReadFile(fsys, path.Join(migrationsDir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("load migration specs: read %q: %w", entry.Name(), err)
		}
		content := strings.TrimSpace(string(contentBytes))
		if content == "" {
			return nil, fmt.Errorf("load migration specs: %q is empty", entry.Name())
		}

		spec := partials[version]
		if spec == nil {
			spec = &partialSpec{version: version, name: name}
			partials[version] = spec
		}
		if spec.name != name {
			return nil, fmt.Errorf("load migration specs: version %06d uses conflicting names %q and %q", version, spec.name, name)
		}

		switch direction {
		case "up":
			if spec.upSQL != "" {
				return nil, fmt.Errorf("load migration specs: duplicate up migration for version %06d", version)
			}
			spec.upSQL = content
		case "down":
			if spec.downSQL != "" {
				return nil, fmt.Errorf("load migration specs: duplicate down migration for version %06d", version)
			}
			spec.downSQL = content
		default:
			return nil, fmt.Errorf("load migration specs: unsupported direction %q", direction)
		}
	}

	if len(partials) == 0 {
		return nil, fmt.Errorf("load migration specs: no migration files found in %q", migrationsDir)
	}

	specs := make([]migrationSpec, 0, len(partials))
	for _, partial := range partials {
		if partial.upSQL == "" || partial.downSQL == "" {
			return nil, fmt.Errorf("load migration specs: version %06d_%s must have both up and down SQL", partial.version, partial.name)
		}

		specs = append(specs, migrationSpec{
			Version:    partial.version,
			Name:       partial.name,
			UpSQL:      partial.upSQL,
			DownSQL:    partial.downSQL,
			UpChecksum: checksumSQL(partial.upSQL),
		})
	}

	sort.Slice(specs, func(i, j int) bool {
		return specs[i].Version < specs[j].Version
	})

	return specs, nil
}

func checksumSQL(sql string) string {
	sum := sha256.Sum256([]byte(sql))
	return fmt.Sprintf("%x", sum[:])
}

func ensureSchemaMigrationsTable(ctx context.Context, conn *pgxpool.Conn) error {
	if _, err := conn.Exec(ctx, schemaMigrationsSQL); err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", err)
	}
	return nil
}

func listAppliedMigrations(ctx context.Context, conn *pgxpool.Conn) ([]appliedMigration, error) {
	rows, err := conn.Query(ctx, `
		SELECT version, name, checksum, applied_at
		FROM schema_migrations
		ORDER BY version ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list applied migrations: %w", err)
	}
	defer rows.Close()

	var items []appliedMigration
	for rows.Next() {
		var item appliedMigration
		if err := rows.Scan(&item.Version, &item.Name, &item.Checksum, &item.AppliedAt); err != nil {
			return nil, fmt.Errorf("list applied migrations: scan row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list applied migrations: %w", err)
	}

	return items, nil
}

func applyMigration(ctx context.Context, conn *pgxpool.Conn, spec migrationSpec) (err error) {
	log.Printf("🗄️  Applying migration %06d_%s", spec.Version, spec.Name)

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("apply migration %06d_%s: begin tx: %w", spec.Version, spec.Name, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	if err = executeMigrationScript(ctx, tx, spec.UpSQL); err != nil {
		return fmt.Errorf("apply migration %06d_%s: execute up SQL: %w", spec.Version, spec.Name, err)
	}

	if _, err = tx.Exec(ctx, `
		INSERT INTO schema_migrations (version, name, checksum)
		VALUES ($1, $2, $3)
	`, spec.Version, spec.Name, spec.UpChecksum); err != nil {
		return fmt.Errorf("apply migration %06d_%s: record migration: %w", spec.Version, spec.Name, err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("apply migration %06d_%s: commit: %w", spec.Version, spec.Name, err)
	}
	return nil
}

func rollbackMigration(ctx context.Context, conn *pgxpool.Conn, spec migrationSpec) (err error) {
	log.Printf("🗄️  Rolling back migration %06d_%s", spec.Version, spec.Name)

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("rollback migration %06d_%s: begin tx: %w", spec.Version, spec.Name, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	if err = executeMigrationScript(ctx, tx, spec.DownSQL); err != nil {
		return fmt.Errorf("rollback migration %06d_%s: execute down SQL: %w", spec.Version, spec.Name, err)
	}

	if _, err = tx.Exec(ctx, "DELETE FROM schema_migrations WHERE version = $1", spec.Version); err != nil {
		return fmt.Errorf("rollback migration %06d_%s: delete migration record: %w", spec.Version, spec.Name, err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("rollback migration %06d_%s: commit: %w", spec.Version, spec.Name, err)
	}
	return nil
}

func executeMigrationScript(ctx context.Context, tx pgx.Tx, sql string) error {
	results, err := tx.Conn().PgConn().Exec(ctx, sql).ReadAll()
	if err != nil {
		return err
	}
	for _, result := range results {
		if result.Err != nil {
			return result.Err
		}
	}
	return nil
}

func (db *DB) withMigrationLock(ctx context.Context, action string, fn func(conn *pgxpool.Conn) error) error {
	conn, err := db.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("%s migrations: acquire connection: %w", action, err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", migrationsLockKey); err != nil {
		return fmt.Errorf("%s migrations: advisory lock: %w", action, err)
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, unlockErr := conn.Exec(unlockCtx, "SELECT pg_advisory_unlock($1)", migrationsLockKey); unlockErr != nil {
			log.Printf("⚠️  unlock migrations advisory lock failed: %v", unlockErr)
		}
	}()

	return fn(conn)
}

func buildMigrationStatuses(specs []migrationSpec, applied []appliedMigration) []MigrationStatus {
	statuses := make([]MigrationStatus, 0, len(specs)+len(applied))
	specsByVersion := make(map[int64]migrationSpec, len(specs))
	appliedByVersion := make(map[int64]appliedMigration, len(applied))

	for _, spec := range specs {
		specsByVersion[spec.Version] = spec
	}
	for _, item := range applied {
		appliedByVersion[item.Version] = item
	}

	for _, spec := range specs {
		status := MigrationStatus{
			Version:       spec.Version,
			Name:          spec.Name,
			ChecksumMatch: true,
		}

		if item, ok := appliedByVersion[spec.Version]; ok {
			appliedAt := item.AppliedAt
			status.Applied = true
			status.AppliedAt = &appliedAt
			status.ChecksumMatch = item.Checksum == spec.UpChecksum
		}

		statuses = append(statuses, status)
	}

	for _, item := range applied {
		if _, ok := specsByVersion[item.Version]; ok {
			continue
		}
		appliedAt := item.AppliedAt
		statuses = append(statuses, MigrationStatus{
			Version:       item.Version,
			Name:          item.Name,
			Applied:       true,
			AppliedAt:     &appliedAt,
			ChecksumMatch: false,
			MissingFile:   true,
		})
	}

	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].Version < statuses[j].Version
	})

	return statuses
}
