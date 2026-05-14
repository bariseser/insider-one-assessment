package repository_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"insider-one-assessment/internal/repository"
	postgresdb "insider-one-assessment/pkgs/postgres"

	_ "github.com/jackc/pgx/v5/stdlib"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	ctx                    context.Context
	cancelCtx              context.CancelFunc
	pgContainer            testcontainers.Container
	db                     *sql.DB
	dbClient               postgresdb.IPostgresInstance
	notificationRepository repository.INotificationRepository
	templateRepository     repository.ITemplateRepository
)

func TestRepository(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Repository Suite")
}

var _ = BeforeSuite(func() {
	ctx, cancelCtx = context.WithCancel(context.Background())

	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "postgres",
			"POSTGRES_PASSWORD": "postgres",
			"POSTGRES_DB":       "insider_assessment_test",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").WithStartupTimeout(2 * time.Minute),
	}

	var err error
	pgContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	Expect(err).NotTo(HaveOccurred())

	host, err := pgContainer.Host(ctx)
	Expect(err).NotTo(HaveOccurred())

	port, err := pgContainer.MappedPort(ctx, "5432/tcp")
	Expect(err).NotTo(HaveOccurred())

	databaseURL := fmt.Sprintf("postgres://postgres:postgres@%s:%s/insider_assessment_test?sslmode=disable", host, port.Port())

	Eventually(func() error {
		db, err = sql.Open("pgx", databaseURL)
		if err != nil {
			return err
		}
		return db.PingContext(ctx)
	}, 30*time.Second, 500*time.Millisecond).Should(Succeed())

	Expect(applyMigrations(ctx, db)).To(Succeed())
})

var _ = AfterSuite(func() {
	if cancelCtx != nil {
		cancelCtx()
	}
	if db != nil {
		_ = db.Close()
	}
	if pgContainer != nil {
		_ = pgContainer.Terminate(context.Background())
	}
})

var _ = BeforeEach(func() {
	Expect(cleanupTables(ctx, db)).To(Succeed())
	var err error
	dbClient, err = postgresdb.NewClientFromSQLDB(db, false, postgresdb.Options{})
	Expect(err).NotTo(HaveOccurred())
	notificationRepository = repository.NewNotificationRepository(dbClient)
	templateRepository = repository.NewTemplateRepository(dbClient)
})

func applyMigrations(ctx context.Context, db *sql.DB) error {
	entries, err := os.ReadDir(filepath.Join("..", "..", "db", "migrations"))
	if err != nil {
		return fmt.Errorf("read migrations directory: %w", err)
	}

	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		paths = append(paths, filepath.Join("..", "..", "db", "migrations", entry.Name()))
	}

	sort.Strings(paths)

	for _, path := range paths {
		payload, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", path, err)
		}
		if _, err := db.ExecContext(ctx, string(payload)); err != nil {
			return fmt.Errorf("apply migration %s: %w", path, err)
		}
	}

	return nil
}

func cleanupTables(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		TRUNCATE TABLE
			notification_attempts,
			outbox_events,
			idempotency_keys,
			notifications,
			notification_batches,
			message_templates
		RESTART IDENTITY CASCADE
	`)
	return err
}
