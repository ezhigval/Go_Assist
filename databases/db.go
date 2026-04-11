package databases

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB — ядро подключения к PostgreSQL, реализует DatabaseAPI
type DB struct {
	pool *pgxpool.Pool
	cfg  Config
}

// InitDB создаёт экземпляр БД, инициализирует пул и применяет миграции
func InitDB(ctx context.Context, cfg Config) (*DB, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse DSN: %w", err)
	}
	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	db := &DB{pool: pool, cfg: cfg}

	if cfg.AutoMigrate {
		if err := db.runMigrations(ctx); err != nil {
			pool.Close()
			return nil, fmt.Errorf("migrations failed: %w", err)
		}
	} else {
		log.Println("🗄️  Auto migrations disabled (DB_AUTO_MIGRATE=false)")
	}

	return db, nil
}

// Start проверяет подключение (pgx инициализируется лениво, но ping полезен для readiness)
func (db *DB) Start(ctx context.Context) error {
	if err := db.pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping DB failed: %w", err)
	}
	log.Printf("🟢 PostgreSQL connected to %s@%s:%s/%s", db.cfg.User, db.cfg.Host, db.cfg.Port, db.cfg.Name)
	status, err := db.InspectStorageRLS(ctx)
	if err != nil {
		log.Printf("⚠️ storage RLS inspection failed: %v", err)
		return nil
	}
	log.Printf(
		"🛡️ storage RLS effective=%t user=%s superuser=%t bypassrls=%t journal_enabled=%t stats_enabled=%t",
		status.Effective(),
		status.CurrentUser,
		status.RoleSuperuser,
		status.RoleBypassRLS,
		status.Journal.TableRLSEnabled,
		status.Stats.TableRLSEnabled,
	)
	for _, warning := range status.Warnings() {
		log.Printf("⚠️ storage RLS: %s", warning)
	}
	return nil
}

// Stop корректно закрывает пул соединений
func (db *DB) Stop() error {
	if db.pool != nil {
		db.pool.Close()
		log.Println("🔴 PostgreSQL connection pool closed")
	}
	return nil
}

// Ping экспортирует проверку здоровья БД (для k8s readiness/liveness)
func (db *DB) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}
