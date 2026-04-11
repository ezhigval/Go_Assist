package databases

import (
	"io/fs"
	"testing"
	"testing/fstest"
	"time"
)

func TestLoadMigrationSpecsOrdersAndValidatesPairs(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"migrations/000002_add_index.up.sql":   {Data: []byte("CREATE INDEX idx ON foo(bar);")},
		"migrations/000002_add_index.down.sql": {Data: []byte("DROP INDEX IF EXISTS idx;")},
		"migrations/000001_init.up.sql":        {Data: []byte("CREATE TABLE foo(id BIGINT);")},
		"migrations/000001_init.down.sql":      {Data: []byte("DROP TABLE IF EXISTS foo;")},
	}

	specs, err := loadMigrationSpecs(fsys)
	if err != nil {
		t.Fatalf("loadMigrationSpecs returned error: %v", err)
	}

	if len(specs) != 2 {
		t.Fatalf("expected 2 specs, got %d", len(specs))
	}
	if specs[0].Version != 1 || specs[0].Name != "init" {
		t.Fatalf("unexpected first migration: %+v", specs[0])
	}
	if specs[1].Version != 2 || specs[1].Name != "add_index" {
		t.Fatalf("unexpected second migration: %+v", specs[1])
	}
	if specs[0].UpChecksum == "" || specs[1].UpChecksum == "" {
		t.Fatalf("expected non-empty checksums")
	}
}

func TestLoadMigrationSpecsRejectsMissingDirectionPair(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"migrations/000001_init.up.sql": {Data: []byte("CREATE TABLE foo(id BIGINT);")},
	}

	_, err := loadMigrationSpecs(fsys)
	if err == nil {
		t.Fatal("expected error for missing down migration")
	}
}

func TestLoadMigrationSpecsRejectsUnexpectedFilename(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"migrations/README.txt": {Data: []byte("not allowed")},
	}

	_, err := loadMigrationSpecs(fsys)
	if err == nil {
		t.Fatal("expected error for unexpected filename")
	}
}

func TestBuildMigrationStatuses(t *testing.T) {
	t.Parallel()

	specs := []migrationSpec{
		{Version: 1, Name: "init", UpChecksum: checksumSQL("up-1")},
		{Version: 2, Name: "add_journal", UpChecksum: checksumSQL("up-2")},
	}
	appliedAt := time.Date(2026, time.April, 11, 10, 0, 0, 0, time.UTC)
	applied := []appliedMigration{
		{Version: 1, Name: "init", Checksum: checksumSQL("up-1"), AppliedAt: appliedAt},
		{Version: 3, Name: "removed_locally", Checksum: checksumSQL("up-3"), AppliedAt: appliedAt},
	}

	statuses := buildMigrationStatuses(specs, applied)
	if len(statuses) != 3 {
		t.Fatalf("expected 3 statuses, got %d", len(statuses))
	}

	if statuses[0].State() != "applied" {
		t.Fatalf("expected version 1 to be applied, got %s", statuses[0].State())
	}
	if statuses[1].State() != "pending" {
		t.Fatalf("expected version 2 to be pending, got %s", statuses[1].State())
	}
	if statuses[2].State() != "orphaned" {
		t.Fatalf("expected version 3 to be orphaned, got %s", statuses[2].State())
	}
}

func TestLoadMigrationSpecsNeedsFiles(t *testing.T) {
	t.Parallel()

	emptyFS := fstest.MapFS{
		"migrations/.keep": {Data: []byte{}},
	}
	_, err := loadMigrationSpecs(fs.FS(emptyFS))
	if err == nil {
		t.Fatal("expected error when no migration files exist")
	}
}
