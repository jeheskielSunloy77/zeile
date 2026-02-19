package repository

import (
	"context"
	"reflect"
	"strings"
	"sync"
	"unicode"

	"github.com/google/uuid"
	"github.com/jeheskielSunloy77/zeile/internal/domain"
	"gorm.io/gorm"
)

// MockResourceRepository is a simple in-memory implementation of
// repository.ResourceRepository[T] for use in tests across entities.
type MockResourceRepository[T domain.BaseModel] struct {
	mu      sync.RWMutex
	data    map[uuid.UUID]T
	deleted map[uuid.UUID]T
	cacheEn bool
}

func NewMockResourceRepository[T domain.BaseModel](cacheEnabled bool) *MockResourceRepository[T] {
	return &MockResourceRepository[T]{
		data:    make(map[uuid.UUID]T),
		deleted: make(map[uuid.UUID]T),
		cacheEn: cacheEnabled,
	}
}

func (m *MockResourceRepository[T]) EvictCache(ctx context.Context, id uuid.UUID) {
	// no-op for mock
}

func (m *MockResourceRepository[T]) CacheEnabled() bool {
	return m.cacheEn
}

func (m *MockResourceRepository[T]) Store(ctx context.Context, entity *T) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := (*entity).GetID()
	m.data[id] = *entity
	// if it existed in deleted, remove tombstone
	delete(m.deleted, id)
	return nil
}

func (m *MockResourceRepository[T]) GetByID(ctx context.Context, id uuid.UUID, preloads []string) (*T, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if v, ok := m.data[id]; ok {
		return &v, nil
	}
	// not found
	return nil, gorm.ErrRecordNotFound
}

func (m *MockResourceRepository[T]) GetMany(ctx context.Context, opts GetManyOptions) ([]T, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]T, 0, len(m.data))
	for _, v := range m.data {
		list = append(list, v)
	}
	return list, int64(len(list)), nil
}

func (m *MockResourceRepository[T]) Update(ctx context.Context, entity T, updates ...map[string]any) (*T, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := entity.GetID()
	if _, ok := m.data[id]; !ok {
		return nil, gorm.ErrRecordNotFound
	}
	if len(updates) > 0 && updates[0] != nil {
		applyUpdates(&entity, updates[0])
	}
	m.data[id] = entity
	return &entity, nil
}

func (m *MockResourceRepository[T]) Destroy(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if v, ok := m.data[id]; ok {
		// soft-delete: move to deleted map
		m.deleted[id] = v
		delete(m.data, id)
		return nil
	}
	return gorm.ErrRecordNotFound
}

func (m *MockResourceRepository[T]) Kill(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, id)
	delete(m.deleted, id)
	return nil
}

func (m *MockResourceRepository[T]) Restore(ctx context.Context, id uuid.UUID) (*T, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if v, ok := m.deleted[id]; ok {
		m.data[id] = v
		delete(m.deleted, id)
		return &v, nil
	}
	return nil, gorm.ErrRecordNotFound
}

// applyUpdates maps update keys to struct fields and applies values via reflection.
func applyUpdates[T any](entity *T, updates map[string]any) {
	if entity == nil || len(updates) == 0 {
		return
	}

	rv := reflect.ValueOf(entity)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return
	}
	rv = rv.Elem()
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return
	}

	fieldIndex := buildFieldIndex(rv.Type())
	for key, value := range updates {
		if idx, ok := fieldIndex[normalizeFieldKey(key)]; ok {
			field := rv.Field(idx)
			setFieldValue(field, value)
		}
	}
}

func buildFieldIndex(rt reflect.Type) map[string]int {
	index := make(map[string]int, rt.NumField()*3)
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if field.PkgPath != "" {
			continue
		}
		addKey(index, field.Name, i)
		addKey(index, toSnakeCase(field.Name), i)
		if jsonTag := strings.Split(field.Tag.Get("json"), ",")[0]; jsonTag != "" && jsonTag != "-" {
			addKey(index, jsonTag, i)
		}
		if gormTag := field.Tag.Get("gorm"); gormTag != "" {
			for _, part := range strings.Split(gormTag, ";") {
				if strings.HasPrefix(part, "column:") {
					addKey(index, strings.TrimPrefix(part, "column:"), i)
				}
			}
		}
	}
	return index
}

func addKey(index map[string]int, key string, fieldIndex int) {
	key = normalizeFieldKey(key)
	if key == "" {
		return
	}
	if _, exists := index[key]; !exists {
		index[key] = fieldIndex
	}
}

func normalizeFieldKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}

func toSnakeCase(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	var b strings.Builder
	b.Grow(len(runes) + 4)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := runes[i-1]
				nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
				if unicode.IsLower(prev) || (unicode.IsUpper(prev) && nextLower) {
					b.WriteByte('_')
				}
			}
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func setFieldValue(field reflect.Value, value any) {
	if !field.IsValid() || !field.CanSet() {
		return
	}

	if value == nil {
		field.Set(reflect.Zero(field.Type()))
		return
	}

	val := reflect.ValueOf(value)
	if !val.IsValid() {
		return
	}

	if field.Kind() == reflect.Pointer {
		setPointerFieldValue(field, val)
		return
	}

	if val.Type().AssignableTo(field.Type()) {
		field.Set(val)
		return
	}
	if val.Type().ConvertibleTo(field.Type()) {
		field.Set(val.Convert(field.Type()))
		return
	}
	if val.Kind() == reflect.Pointer && !val.IsNil() {
		elem := val.Elem()
		if elem.Type().AssignableTo(field.Type()) {
			field.Set(elem)
			return
		}
		if elem.Type().ConvertibleTo(field.Type()) {
			field.Set(elem.Convert(field.Type()))
		}
	}
}

func setPointerFieldValue(field reflect.Value, val reflect.Value) {
	if val.Type().AssignableTo(field.Type()) {
		field.Set(val)
		return
	}
	if val.Type().ConvertibleTo(field.Type()) {
		field.Set(val.Convert(field.Type()))
		return
	}

	elemType := field.Type().Elem()
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			field.Set(reflect.Zero(field.Type()))
			return
		}
		val = val.Elem()
	}
	if val.Type().AssignableTo(elemType) {
		ptr := reflect.New(elemType)
		ptr.Elem().Set(val)
		field.Set(ptr)
		return
	}
	if val.Type().ConvertibleTo(elemType) {
		ptr := reflect.New(elemType)
		ptr.Elem().Set(val.Convert(elemType))
		field.Set(ptr)
	}
}
