package handlers

import (
	"strings"

	"gorm.io/gorm"
)

// containsFold filters rows whose column contains term, ignoring case, with
// "%" and "_" in term matched literally. MySQL and SQLite compare LIKE
// case-insensitively while PostgreSQL does not, and an unescaped "%" would act
// as a wildcard, so the comparison is made explicit. column must be a
// trusted, code-supplied identifier, never request input.
func containsFold(q *gorm.DB, column, term string) *gorm.DB {
	return q.Where("LOWER("+column+") LIKE ? ESCAPE '!'", "%"+escapeLike(strings.ToLower(term))+"%")
}

// escapeLike escapes the LIKE metacharacters with "!", which needs no extra
// quoting in any of the three dialects (unlike backslash in MySQL).
func escapeLike(s string) string {
	return strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(s)
}
