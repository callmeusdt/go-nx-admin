package app

import (
	"gorm.io/gorm"

	"go-nx-admin/internal/dbutil"
)

// ContainsFold narrows q to rows whose column contains term. Matching ignores
// case on SQLite, MySQL and PostgreSQL alike (LIKE alone is case-sensitive only
// on PostgreSQL), and "%" and "_" in term are matched literally.
//
// column must be a trusted identifier written in code, never request input.
// Quote reserved words yourself, or use GORM's clause.Column for them.
func ContainsFold(q *gorm.DB, column, term string) *gorm.DB {
	return dbutil.ContainsFold(q, column, term)
}

// EscapeLike escapes the LIKE metacharacters for use with ESCAPE '!'.
func EscapeLike(s string) string { return dbutil.EscapeLike(s) }
