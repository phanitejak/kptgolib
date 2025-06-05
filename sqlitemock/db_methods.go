package sqlitemock

import (
	"errors"

	"gorm.io/gorm"
)

func (m *MockSqlite) Select(out any, conds ...any) error {
	if m.wantError() {
		return errors.New("forced mock error during select")
	}
	return m.DB.Find(out, conds...).Error
}

func (m *MockSqlite) First(out any, conds ...any) error {
	if m.wantError() {
		return errors.New("forced mock error during first")
	}
	return m.DB.First(out, conds...).Error
}

func (m *MockSqlite) Update(model any, updates map[string]any) error {
	if m.wantError() {
		return errors.New("forced mock error during update")
	}
	return m.DB.Model(model).Updates(updates).Error
}

func (m *MockSqlite) Delete(model any, conds ...any) error {
	if m.wantError() {
		return errors.New("forced mock error during delete")
	}
	return m.DB.Delete(model, conds...).Error
}

func (m *MockSqlite) Exec(sql string, values ...any) error {
	if m.wantError() {
		return errors.New("forced mock error during exec")
	}
	return m.DB.Exec(sql, values...).Error
}

func (m *MockSqlite) Truncate(models ...any) error {
	for _, model := range models {
		if err := m.DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Model(model).Delete(nil).Error; err != nil {
			return err
		}
	}
	return nil
}
