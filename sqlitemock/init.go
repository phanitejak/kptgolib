package sqlitemock

import (
	"errors"
	"fmt"

	_ "github.com/glebarez/sqlite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type MockSqlite struct {
	DB              *gorm.DB
	ExpectError     bool
	ExpectErrorOnce bool
}

// New creates a new in-memory SQLite DB and returns a MockSqlite wrapper
func New(withForeignKeys bool) (*MockSqlite, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	if withForeignKeys {
		// Enable foreign key constraints
		db.Exec("PRAGMA foreign_keys = ON")
	}

	return &MockSqlite{DB: db}, nil
}

func MustNew(withForeignKeys bool) *MockSqlite {
	db, err := New(withForeignKeys)
	if err != nil {
		panic(err)
	}
	return db
}

// InitializeSchema runs GORM AutoMigrate for provided models
func (m *MockSqlite) InitializeSchema(models ...any) error {
	if len(models) == 0 {
		return nil
	}
	return m.DB.AutoMigrate(models...)
}

// Seed inserts sample data into the mock DB
func (m *MockSqlite) Seed(data ...any) error {
	if m.wantError() {
		return errors.New("forced mock error during seed")
	}
	for _, item := range data {
		if err := m.DB.Create(item).Error; err != nil {
			return err
		}
	}
	return nil
}

func (m *MockSqlite) Reset(models ...any) error {
	// Drop tables
	for _, model := range models {
		if err := m.DB.Migrator().DropTable(model); err != nil {
			return err
		}
	}

	// Recreate schema
	return m.InitializeSchema(models...)
}

func (m *MockSqlite) WithExpectError(fn func()) {
	m.ExpectError = true
	defer func() { m.ExpectError = false }()
	fn()
}

func (m *MockSqlite) wantError() bool {
	expErr := m.ExpectError || m.ExpectErrorOnce
	if m.ExpectErrorOnce {
		m.ExpectErrorOnce = false
	}
	return expErr
}
