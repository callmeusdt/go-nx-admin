// Package dbutil holds helpers that keep hand-written SQL portable across
// SQLite, MySQL and PostgreSQL.
package dbutil

import (
	"strings"

	"gorm.io/gorm"
)

// ContainsFold narrows q to rows whose column contains term, ignoring case,
// with "%" and "_" in term matched literally. MySQL and SQLite compare LIKE
// case-insensitively while PostgreSQL does not, and an unescaped "%" would act
// as a wildcard, so the comparison is made explicit. column must be a trusted,
// code-supplied identifier, never request input.
func ContainsFold(q *gorm.DB, column, term string) *gorm.DB {
	return q.Where("LOWER("+column+") LIKE ? ESCAPE '!'", "%"+EscapeLike(strings.ToLower(term))+"%")
}

// EscapeLike escapes the LIKE metacharacters with "!", which needs no extra
// quoting in any of the three dialects (unlike backslash in MySQL).
func EscapeLike(s string) string {
	return strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(s)
}
