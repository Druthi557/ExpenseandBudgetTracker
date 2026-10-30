package service

import (
	"context"

	"pebt/backend/internal/store"
)

// ListCategories returns the user's categories with an optional type filter
// (FR2.2/FR2.6).
func (s *Service) ListCategories(ctx context.Context, userID int64, typeRaw string) ([]store.Category, *Error) {
	var typ *string
	if typeRaw != "" {
		t, e := validateCategoryType(Text(typeRaw))
		if e != nil {
			return nil, e
		}
		typ = &t
	}
	cats, err := s.Store.ListCategories(ctx, userID, typ)
	if err != nil {
		return nil, internal(s.Log, err)
	}
	return cats, nil
}

// CreateCategory validates and creates a category (FR2.1/FR2.5).
func (s *Service) CreateCategory(ctx context.Context, userID int64, nameRaw, typeRaw Text) (*store.Category, *Error) {
	name, e := validateCategoryName(nameRaw)
	if e != nil {
		return nil, e
	}
	typ, e := validateCategoryType(typeRaw)
	if e != nil {
		return nil, e
	}
	id, err := s.Store.CreateCategory(ctx, userID, name, typ)
	if err != nil {
		return nil, s.mapDBError(err, "")
	}
	return &store.Category{ID: id, UserID: userID, Name: name, Type: typ}, nil
}

// UpdateCategory renames and optionally retypes a category (FR2.3, D27).
// Retyping a referenced category is blocked by the composite foreign keys
// and mapped to 409 CATEGORY_IN_USE.
func (s *Service) UpdateCategory(ctx context.Context, userID int64, idRaw string, nameRaw, typeRaw Text) (*store.Category, *Error) {
	id, e := parseID(idRaw)
	if e != nil {
		return nil, e
	}
	name, e := validateCategoryName(nameRaw)
	if e != nil {
		return nil, e
	}
	typ, e := validateCategoryType(typeRaw)
	if e != nil {
		return nil, e
	}
	ok, err := s.Store.UpdateCategory(ctx, userID, id, name, typ)
	if err != nil {
		return nil, s.mapDBError(err, "Category is referenced by transactions or budgets and cannot be changed")
	}
	if !ok {
		return nil, notFound("Category")
	}
	return &store.Category{ID: id, UserID: userID, Name: name, Type: typ}, nil
}

// DeleteCategory removes an unused category (FR2.4). Referenced categories
// are blocked by ON DELETE RESTRICT → 409 CATEGORY_IN_USE (AC12).
func (s *Service) DeleteCategory(ctx context.Context, userID int64, idRaw string) *Error {
	id, e := parseID(idRaw)
	if e != nil {
		return e
	}
	ok, err := s.Store.DeleteCategory(ctx, userID, id)
	if err != nil {
		return s.mapDBError(err, "Category is referenced by transactions or budgets and cannot be deleted")
	}
	if !ok {
		return notFound("Category")
	}
	return nil
}
