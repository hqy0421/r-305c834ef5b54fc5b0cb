package trdsql

import (
	"strings"
)

// sqlClauseKeywords are keywords that start a new SQL clause.
// Identifiers following them must never be mistaken for column names.
var sqlClauseKeywords = map[string]bool{
	"SELECT": true, "FROM": true, "WHERE": true, "GROUP": true, "BY": true,
	"HAVING": true, "WINDOW": true, "ORDER": true, "LIMIT": true, "OFFSET": true,
	"FETCH": true, "UNION": true, "EXCEPT": true, "INTERSECT": true, "JOIN": true,
	"INNER": true, "LEFT": true, "RIGHT": true, "FULL": true, "CROSS": true,
	"OUTER": true, "ON": true, "USING": true, "AS": true, "AND": true, "OR": true,
	"NOT": true, "IN": true, "IS": true, "NULL": true, "LIKE": true, "BETWEEN": true,
	"EXISTS": true, "CASE": true, "WHEN": true, "THEN": true, "ELSE": true, "END": true,
	"INTO": true, "VALUES": true, "SET": true, "UPDATE": true, "TABLE": true,
	"INSERT": true, "DELETE": true, "DISTINCT": true, "ALL": true, "WITH": true,
	"RECURSIVE": true, "ASC": true, "DESC": true, "CAST": true, "OVER": true,
	"PARTITION": true, "LATERAL": true, "NATURAL": true,
}

// sqlReservedIdentifiers are standalone words that are never column names.
var sqlReservedIdentifiers = map[string]bool{
	"TRUE": true, "FALSE": true,
}

// QueryColumns returns the column names explicitly referenced by the query.
// It returns nil (no pruning, import every field) when:
//   - the query contains a "*" (SELECT *, t.*, COUNT(*) ...),
//   - the query contains a subquery,
//   - the query cannot be analyzed conservatively.
//
// Qualified names such as "t.device_id" contribute the unqualified part
// "device_id". Quoted identifiers are unquoted. The set includes columns
// referenced anywhere in the query (SELECT list, WHERE, GROUP BY, ORDER BY,
// ON ...) so that filtering and grouping columns are always retained.
func QueryColumns(query string) []string {
	tokens := SQLFields(query)

	// Any "*" means the result set may require every column.
	for _, token := range tokens {
		if token == "*" || strings.HasSuffix(token, ".*") || strings.Contains(token, "(*)") {
			return nil
		}
	}

	// A nested SELECT is a subquery; only analyze flat, conservatively safe queries.
	selectCount := 0
	for _, token := range tokens {
		if strings.EqualFold(token, "SELECT") || strings.EqualFold(token, "*SELECT") {
			selectCount++
			if selectCount > 1 {
				return nil
			}
		}
	}

	seen := make(map[string]bool)
	columns := make([]string, 0)
	tableAliases, tableNames := queryTableReferences(tokens)
	skipNext := false
	for i, token := range tokens {
		if token == "" || isWhitespaceOnly(token) {
			continue
		}
		// Punctuation emitted by SQLFields.
		switch token {
		case ",", ";", "=", "(", ")":
			skipNext = false
			continue
		}
		upper := strings.ToUpper(token)
		switch {
		case upper == "FROM" || upper == "*FROM" || upper == "JOIN" || upper == "TABLE" || upper == "INTO" || upper == "UPDATE":
			// The next identifier is a table name, not a column.
			skipNext = true
			continue
		case upper == "AS":
			// Skip the alias identifier that follows.
			skipNext = true
			continue
		case sqlClauseKeywords[upper] || isSQLKeyWords(token):
			skipNext = false
			continue
		}

		if skipNext {
			// A table name (possibly schema-qualified) or an alias.
			skipNext = false
			continue
		}
		// An identifier directly followed by "(" is a function name
		// (for example upper(col)); skip the name but keep scanning
		// its argument list, which may reference columns.
		if nextToken(tokens, i) == "(" {
			continue
		}
		// Table and file names (for example "b.jsonl") are never columns.
		if tableNames[token] {
			continue
		}
		name := columnNameToken(token)
		if name == "" || sqlReservedIdentifiers[strings.ToUpper(name)] || tableAliases[name] {
			continue
		}
		if !seen[name] {
			seen[name] = true
			columns = append(columns, name)
		}
	}
	return columns
}

// queryTableReferences collects the table aliases and the verbatim table
// (file) name tokens appearing after FROM/JOIN. It handles explicit
// ("tbl AS t"), implicit ("tbl t"), and comma-separated
// ("a.jsonl a, b.jsonl b") table references.
func queryTableReferences(tokens []string) (aliases map[string]bool, names map[string]bool) {
	aliases = make(map[string]bool)
	names = make(map[string]bool)
	phase := phaseOther // what the next identifier means
	inFrom := false     // false while scanning the SELECT list
	for _, token := range tokens {
		if token == "" || isWhitespaceOnly(token) {
			continue
		}
		switch token {
		case ",":
			// Inside a FROM list another table follows; in the SELECT
			// list it merely separates columns.
			if inFrom {
				phase = phaseTable
			}
			continue
		case ";", "=", "(", ")":
			phase = phaseOther
			continue
		}
		upper := strings.ToUpper(token)
		switch {
		case upper == "FROM" || upper == "*FROM" || upper == "JOIN" || upper == "TABLE" || upper == "INTO" || upper == "UPDATE":
			phase = phaseTable
			inFrom = true
			continue
		case upper == "AS":
			if inFrom {
				phase = phaseExplicitAlias
			}
			continue
		case sqlClauseKeywords[upper] || isSQLKeyWords(token):
			// A clause keyword (WHERE, GROUP, ON ...) ends a pending
			// implicit-alias expectation, so the next identifier is a column.
			phase = phaseOther
			continue
		}
		if !inFrom {
			// Identifiers in the SELECT list are columns, never tables.
			continue
		}
		switch phase {
		case phaseTable:
			// Record the verbatim table/file name token.
			names[token] = true
			// An unquoted alias may follow directly.
			phase = phaseImplicitAlias
		case phaseExplicitAlias, phaseImplicitAlias:
			// An alias is a bare identifier: it must not contain a dot,
			// otherwise it is (a fragment of) a qualified table/file name.
			if !strings.Contains(token, ".") {
				if name := columnNameToken(token); name != "" {
					aliases[name] = true
				}
			}
			phase = phaseOther
		}
	}
	return aliases, names
}

// aliasPhase tracks which part of a table reference is being scanned.
type aliasPhase int

const (
	phaseOther aliasPhase = iota
	phaseTable
	phaseExplicitAlias
	phaseImplicitAlias
)

// nextToken returns the first punctuation/word token after idx,
// ignoring whitespace tokens emitted by SQLFields.
func nextToken(tokens []string, idx int) string {
	for j := idx + 1; j < len(tokens); j++ {
		if !isWhitespaceOnly(tokens[j]) {
			return tokens[j]
		}
	}
	return ""
}

// isWhitespaceOnly reports whether the token consists solely of whitespace.
func isWhitespaceOnly(token string) bool {
	if token == "" {
		return true
	}
	for _, r := range token {
		if r != ' ' && r != '\t' && r != '\r' && r != '\n' {
			return false
		}
	}
	return true
}

// columnNameToken reduces a parsed token to an unqualified, unquoted
// column name. "t.col" and "t.`col`" become "col".
// String and numeric literals yield an empty string.
func columnNameToken(token string) string {
	if token == "" {
		return ""
	}
	// String literals are values, not column names.
	if token[0] == '\'' {
		return ""
	}
	// Numeric literals (integer or decimal) are values.
	if isNumericLiteral(token) {
		return ""
	}
	// Quoted identifiers: trim the matching quotes and accept the content.
	if (token[0] == '"' || token[0] == '`') && len(token) >= 2 && token[len(token)-1] == token[0] {
		return token[1 : len(token)-1]
	}
	// Take the last dotted component of a qualified name.
	if idx := strings.LastIndex(token, "."); idx >= 0 {
		token = token[idx+1:]
		if isNumericLiteral(token) {
			return ""
		}
	}
	// Reject anything that is not a plain unquoted identifier.
	for _, r := range token {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' ||
			r >= '0' && r <= '9' || r == '_' || r > 127) {
			return ""
		}
	}
	return token
}

// isNumericLiteral reports whether the token is an integer or decimal number.
func isNumericLiteral(token string) bool {
	if token == "" {
		return false
	}
	dot := false
	for _, r := range token {
		switch {
		case r >= '0' && r <= '9':
		case r == '.' && !dot:
			dot = true
		default:
			return false
		}
	}
	return true
}
