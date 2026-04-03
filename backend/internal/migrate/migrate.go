package migrate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Migration struct {
	Version string
	Name    string
	SQL     string
}

func Apply(ctx context.Context, pool *pgxpool.Pool, migrationDir string) ([]Migration, error) {
	if pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	if err := ensureTrackingTable(ctx, pool); err != nil {
		return nil, err
	}

	migrations, err := loadMigrations(migrationDir)
	if err != nil {
		return nil, err
	}

	appliedVersions, err := appliedVersions(ctx, pool)
	if err != nil {
		return nil, err
	}

	applied := make([]Migration, 0, len(migrations))
	for _, migration := range migrations {
		if appliedVersions[migration.Version] {
			continue
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return applied, fmt.Errorf("begin migration transaction: %w", err)
		}

		if _, err := tx.Exec(ctx, migration.SQL); err != nil {
			_ = tx.Rollback(ctx)
			return applied, fmt.Errorf("apply migration %s: %w", migration.Name, err)
		}

		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version, name) VALUES ($1, $2)`, migration.Version, migration.Name); err != nil {
			_ = tx.Rollback(ctx)
			return applied, fmt.Errorf("record migration %s: %w", migration.Name, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return applied, fmt.Errorf("commit migration %s: %w", migration.Name, err)
		}

		applied = append(applied, migration)
	}

	return applied, nil
}

func ensureTrackingTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)
	if err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", err)
	}
	return nil
}

func appliedVersions(ctx context.Context, pool *pgxpool.Pool) (map[string]bool, error) {
	rows, err := pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("query applied migrations: %w", err)
	}
	defer rows.Close()

	versions := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("scan migration version: %w", err)
		}
		versions[version] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate applied migrations: %w", err)
	}

	return versions, nil
}

func loadMigrations(migrationDir string) ([]Migration, error) {
	entries, err := os.ReadDir(migrationDir)
	if err != nil {
		return nil, fmt.Errorf("read migration directory %q: %w", migrationDir, err)
	}

	migrations := make([]Migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".sql" {
			continue
		}
		content, err := os.ReadFile(filepath.Join(migrationDir, name))
		if err != nil {
			return nil, fmt.Errorf("read migration %q: %w", name, err)
		}
		version := strings.TrimSuffix(name, filepath.Ext(name))
		migrations = append(migrations, Migration{Version: version, Name: name, SQL: string(content)})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Name < migrations[j].Name
	})

	return migrations, nil
}
