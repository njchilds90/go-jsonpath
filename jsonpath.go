// Package jsonpath provides a complete, spec-compliant JSONPath query engine for Go.
//
// JSONPath is a query language for JSON, similar to XPath for XML. This package
// implements the JSONPath specification allowing you to extract, filter, and
// transform values from JSON documents without full unmarshalling overhead.
//
// # Basic Usage
//
//	data := []byte(`{"store":{"book":[{"title":"Go Programming","price":29.99},{"title":"Clean Code","price":34.99}]}}`)
//
//	results, err := jsonpath.Query(data, "$.store.book[*].title")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// results: ["Go Programming", "Clean Code"]
//
// # AI Agent Usage
//
// This package is designed to be safe for use in AI agent pipelines:
//   - All operations are deterministic
//   - No global state
//   - Structured error types
//   - context.Context support for cancellation
package jsonpath

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// Result represents a single match from a JSONPath query.
// It contains the matched value and the normalized path to it.
type Result struct {
	// Path is the normalized JSONPath expression pointing to this result.
	Path string
	// Value is the matched JSON value. Use type assertions or json.Unmarshal to work with it.
	Value interface{}
}

// MarshalJSON implements json.Marshaler for Result.
func (r Result) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"path":  r.Path,
		"value": r.Value,
	})
}

// Option configures JSONPath query behavior.
type Option func(*engine)

// WithMaxDepth sets the maximum recursion depth for recursive descent operators.
// Default is 100. Set to 0 for unlimited (not recommended for untrusted input).
func WithMaxDepth(depth int) Option {
	return func(e *engine) {
		e.maxDepth = depth
	}
}

// WithAllowMissingKeys controls whether missing keys return an error or empty results.
// Default is false (missing keys return empty results, not errors).
func WithAllowMissingKeys(strict bool) Option {
	return func(e *engine) {
		e.strictKeys = strict
	}
}

// Query executes a JSONPath expression against a JSON document and returns all matches.
//
// Example:
//
//	results, err := jsonpath.Query(data, "$.store.book[0].title")
//	results, err := jsonpath.Query(data, "$.store.book[*].price")
//	results, err := jsonpath.Query(data, "$..author")
//	results, err := jsonpath.Query(data, "$.store.book[?(@.price < 30)].title")
func Query(data []byte, path string, opts ...Option) ([]Result, error) {
	return QueryContext(context.Background(), data, path, opts...)
}

// QueryContext executes a JSONPath expression with context support for cancellation.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//	results, err := jsonpath.QueryContext(ctx, data, "$..price")
func QueryContext(ctx context.Context, data []byte, path string, opts ...Option) ([]Result, error) {
	if ctx == nil {
		return nil, &Error{Code: ErrInvalidInput, Message: "context must not be nil"}
	}

	var root interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, &Error{Code: ErrInvalidJSON, Message: "failed to parse JSON", Cause: err}
	}

	return QueryValueContext(ctx, root, path, opts...)
}

// QueryValue executes a JSONPath expression against an already-parsed Go value.
// Accepts any value produced by json.Unmarshal (map[string]interface{}, []interface{}, etc.).
//
// Example:
//
//	var doc interface{}
//	json.Unmarshal(data, &doc)
//	results, err := jsonpath.QueryValue(doc, "$.users[*].name")
func QueryValue(root interface{}, path string, opts ...Option) ([]Result, error) {
	return QueryValueContext(context.Background(), root, path, opts...)
}

// QueryValueContext executes a JSONPath expression against an already-parsed Go value with context support.
func QueryValueContext(ctx context.Context, root interface{}, path string, opts ...Option) ([]Result, error) {
	if ctx == nil {
		return nil, &Error{Code: ErrInvalidInput, Message: "context must not be nil"}
	}

	e := &engine{
		maxDepth: 100,
	}
	for _, opt := range opts {
		opt(e)
	}
	e.ctx = ctx

	tokens, err := tokenize(path)
	if err != nil {
		return nil, err
	}

	results, err := e.evaluate(root, tokens, "$")
	if err != nil {
		return nil, err
	}

	return results, nil
}

// First returns the first result from a JSONPath query, or nil if no results.
// This is a convenience function for queries expected to return a single value.
//
// Example:
//
//	result, err := jsonpath.First(data, "$.user.name")
//	if result != nil {
//	    name := result.Value.(string)
//	}
func First(data []byte, path string, opts ...Option) (*Result, error) {
	results, err := Query(data, path, opts...)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	return &results[0], nil
}

// Values extracts just the values from a query result, discarding path information.
//
// Example:
//
//	prices, err := jsonpath.Values(data, "$.store.book[*].price")
func Values(data []byte, path string, opts ...Option) ([]interface{}, error) {
	results, err := Query(data, path, opts...)
	if err != nil {
		return nil, err
	}
	vals := make([]interface{}, len(results))
	for i, r := range results {
		vals[i] = r.Value
	}
	return vals, nil
}

// Paths extracts just the paths from a query result, discarding value information.
//
// Example:
//
//	paths, err := jsonpath.Paths(data, "$.store.book[*]")
func Paths(data []byte, path string, opts ...Option) ([]string, error) {
	results, err := Query(data, path, opts...)
	if err != nil {
		return nil, err
	}
	paths := make([]string, len(results))
	for i, r := range results {
		paths[i] = r.Path
	}
	return paths, nil
}

// Exists returns true if the JSONPath expression matches at least one value.
//
// Example:
//
//	if ok, _ := jsonpath.Exists(data, "$.user.admin"); ok {
//	    // handle admin
//	}
func Exists(data []byte, path string, opts ...Option) (bool, error) {
	results, err := Query(data, path, opts...)
	if err != nil {
		return false, err
	}
	return len(results) > 0, nil
}

// MustQuery executes a JSONPath query and panics on error.
// Only use this in tests or when the path is a compile-time constant known to be valid.
func MustQuery(data []byte, path string, opts ...Option) []Result {
	results, err := Query(data, path, opts...)
	if err != nil {
		panic(fmt.Sprintf("jsonpath.MustQuery: %v", err))
	}
	return results
}

// Compile pre-compiles a JSONPath expression for repeated use.
// Use this when the same path will be applied to many documents.
//
// Example:
//
//	p, err := jsonpath.Compile("$.store.book[*].title")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	results1, _ := p.Query(doc1)
//	results2, _ := p.Query(doc2)
type CompiledPath struct {
	raw    string
	tokens []token
}

// Compile parses and validates a JSONPath expression, returning a CompiledPath for reuse.
func Compile(path string) (*CompiledPath, error) {
	tokens, err := tokenize(path)
	if err != nil {
		return nil, err
	}
	return &CompiledPath{raw: path, tokens: tokens}, nil
}

// MustCompile compiles a JSONPath expression and panics if invalid.
// Use only for compile-time constant paths.
func MustCompile(path string) *CompiledPath {
	cp, err := Compile(path)
	if err != nil {
		panic(fmt.Sprintf("jsonpath.MustCompile: %v", err))
	}
	return cp
}

// Query executes the pre-compiled path against a JSON document.
func (cp *CompiledPath) Query(data []byte, opts ...Option) ([]Result, error) {
	return cp.QueryContext(context.Background(), data, opts...)
}

// QueryContext executes the pre-compiled path against a JSON document with context.
func (cp *CompiledPath) QueryContext(ctx context.Context, data []byte, opts ...Option) ([]Result, error) {
	if ctx == nil {
		return nil, &Error{Code: ErrInvalidInput, Message: "context must not be nil"}
	}

	var root interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, &Error{Code: ErrInvalidJSON, Message: "failed to parse JSON", Cause: err}
	}

	e := &engine{maxDepth: 100, ctx: ctx}
	for _, opt := range opts {
		opt(e)
	}

	return e.evaluate(root, cp.tokens, "$")
}

// QueryValue executes the pre-compiled path against a parsed Go value.
func (cp *CompiledPath) QueryValue(root interface{}, opts ...Option) ([]Result, error) {
	return cp.QueryValueContext(context.Background(), root, opts...)
}

// QueryValueContext executes the pre-compiled path against a parsed Go value with context.
func (cp *CompiledPath) QueryValueContext(ctx context.Context, root interface{}, opts ...Option) ([]Result, error) {
	e := &engine{maxDepth: 100, ctx: ctx}
	for _, opt := range opts {
		opt(e)
	}
	return e.evaluate(root, cp.tokens, "$")
}

// String returns the original path string.
func (cp *CompiledPath) String() string {
	return cp.raw
}

// --- Internal token types ---

type tokenKind int

const (
	tokenRoot       tokenKind = iota // $
	tokenChild                       // .key or ['key']
	tokenRecursive                   // ..
	tokenWildcard                    // *
	tokenIndex                       // [n]
	tokenSlice                       // [start:end:step]
	tokenFilter                      // [?(...)]
	tokenUnion                       // [key1,key2] or [0,1,2]
)

type token struct {
	kind    tokenKind
	key     string   // for child
	index   int      // for index
	indices []int    // for union of indices
	keys    []string // for union of keys
	slice   [3]*int  // start, end, step (nil = absent)
	filter  string   // for filter expression
}

// --- Tokenizer ---

func tokenize(path string) ([]token, error) {
	if path == "" {
		return nil, &Error{Code: ErrInvalidPath, Message: "path must not be empty"}
	}

	path = strings.TrimSpace(path)

	if path[0] != '$' {
		return nil, &Error{Code: ErrInvalidPath, Message: "path must start with '$'"}
	}

	tokens := []token{{kind: tokenRoot}}
	i := 1

	for i < len(path) {
		switch {
		case path[i] == '.':
			if i+1 < len(path) && path[i+1] == '.' {
				tokens = append(tokens, token{kind: tokenRecursive})
				i += 2
				// after .., if there's a key or wildcard, collect it
				if i < len(path) && path[i] != '[' && path[i] != '.' {
					key, advance := readIdentifier(path[i:])
					if key == "*" {
						tokens = append(tokens, token{kind: tokenWildcard})
					} else if key != "" {
						tokens = append(tokens, token{kind: tokenChild, key: key})
					}
					i += advance
				}
			} else {
				i++
				if i >= len(path) {
					return nil, &Error{Code: ErrInvalidPath, Message: "unexpected end after '.'"}
				}
				key, advance := readIdentifier(path[i:])
				if key == "*" {
					tokens = append(tokens, token{kind: tokenWildcard})
				} else if key != "" {
					tokens = append(tokens, token{kind: tokenChild, key: key})
				} else {
					return nil, &Error{Code: ErrInvalidPath, Message: fmt.Sprintf("expected key after '.' at position %d", i)}
				}
				i += advance
			}
		case path[i] == '[':
			t, advance, err := parseBracket(path[i:])
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, t)
			i += advance
		default:
			return nil, &Error{Code: ErrInvalidPath, Message: fmt.Sprintf("unexpected character '%c' at position %d", path[i], i)}
		}
	}

	return tokens, nil
}

func readIdentifier(s string) (string, int) {
	i := 0
	for i < len(s) && (isAlphaNum(s[i]) || s[i] == '_' || s[i] == '-') {
		i++
	}
	return s[:i], i
}

func isAlphaNum(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func parseBracket(s string) (token, int, error) {
	// s starts with '['
	end := strings.Index(s, "]")
	if end < 0 {
		return token{}, 0, &Error{Code: ErrInvalidPath, Message: "unclosed '['"}
	}

	inner := s[1:end]

	// Filter: [?(...)]
	if strings.HasPrefix(inner, "?(") && strings.HasSuffix(inner, ")") {
		expr := inner[2 : len(inner)-1]
		return token{kind: tokenFilter, filter: expr}, end + 1, nil
	}

	// Wildcard: [*]
	if inner == "*" {
		return token{kind: tokenWildcard}, end + 1, nil
	}

	// Quoted key: ['key'] or ["key"]
	if (strings.HasPrefix(inner, "'") && strings.HasSuffix(inner, "'")) ||
		(strings.HasPrefix(inner, `"`) && strings.HasSuffix(inner, `"`)) {
		key := inner[1 : len(inner)-1]
		return token{kind: tokenChild, key: key}, end + 1, nil
	}

	// Union: [a,b,c]
	if strings.Contains(inner, ",") {
		parts := strings.Split(inner, ",")
		// Try integer union first
		allInts := true
		indices := make([]int, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			n, err := strconv.Atoi(p)
			if err != nil {
				allInts = false
				break
			}
			indices = append(indices, n)
		}
		if allInts {
			return token{kind: tokenUnion, indices: indices}, end + 1, nil
		}
		// String union
		keys := make([]string, len(parts))
		for i, p := range parts {
			p = strings.TrimSpace(p)
			if (strings.HasPrefix(p, "'") && strings.HasSuffix(p, "'")) ||
				(strings.HasPrefix(p, `"`) && strings.HasSuffix(p, `"`)) {
				p = p[1 : len(p)-1]
			}
			keys[i] = p
		}
		return token{kind: tokenUnion, keys: keys}, end + 1, nil
	}

	// Slice: [start:end] or [start:end:step] or [:end] etc.
	if strings.Contains(inner, ":") {
		parts := strings.Split(inner, ":")
		if len(parts) < 2 || len(parts) > 3 {
			return token{}, 0, &Error{Code: ErrInvalidPath, Message: fmt.Sprintf("invalid slice: %s", inner)}
		}
		var slice [3]*int
		for i, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			n, err := strconv.Atoi(p)
			if err != nil {
				return token{}, 0, &Error{Code: ErrInvalidPath, Message: fmt.Sprintf("invalid slice component: %s", p)}
			}
			slice[i] = &n
		}
		return token{kind: tokenSlice, slice: slice}, end + 1, nil
	}

	// Index: [n]
	n, err := strconv.Atoi(strings.TrimSpace(inner))
	if err != nil {
		// Could be a bare key like [key]
		key := strings.TrimSpace(inner)
		if key == "" {
			return token{}, 0, &Error{Code: ErrInvalidPath, Message: "empty brackets"}
		}
		return token{kind: tokenChild, key: key}, end + 1, nil
	}
	return token{kind: tokenIndex, index: n}, end + 1, nil
}

// --- Evaluator ---

type engine struct {
	ctx        context.Context
	maxDepth   int
	strictKeys bool
}

func (e *engine) evaluate(node interface{}, tokens []token, currentPath string) ([]Result, error) {
	if len(tokens) == 0 {
		return []Result{{Path: currentPath, Value: node}}, nil
	}

	select {
	case <-e.ctx.Done():
		return nil, &Error{Code: ErrCancelled, Message: "context cancelled", Cause: e.ctx.Err()}
	default:
	}

	tok := tokens[0]
	rest := tokens[1:]

	switch tok.kind {
	case tokenRoot:
		return e.evaluate(node, rest, "$")

	case tokenChild:
		obj, ok := node.(map[string]interface{})
		if !ok {
			if e.strictKeys {
				return nil, &Error{Code: ErrTypeMismatch, Message: fmt.Sprintf("expected object at %s, got %T", currentPath, node)}
			}
			return nil, nil
		}
		val, exists := obj[tok.key]
		if !exists {
			if e.strictKeys {
				return nil, &Error{Code: ErrKeyNotFound, Message: fmt.Sprintf("key '%s' not found at %s", tok.key, currentPath)}
			}
			return nil, nil
		}
		return e.evaluate(val, rest, currentPath+"."+tok.key)

	case tokenWildcard:
		return e.evalWildcard(node, rest, currentPath)

	case tokenIndex:
		arr, ok := node.([]interface{})
		if !ok {
			if e.strictKeys {
				return nil, &Error{Code: ErrTypeMismatch, Message: fmt.Sprintf("expected array at %s, got %T", currentPath, node)}
			}
			return nil, nil
		}
		idx := normalizeIndex(tok.index, len(arr))
		if idx < 0 || idx >= len(arr) {
			if e.strictKeys {
				return nil, &Error{Code: ErrIndexOutOfBounds, Message: fmt.Sprintf("index %d out of bounds at %s (length %d)", tok.index, currentPath, len(arr))}
			}
			return nil, nil
		}
		return e.evaluate(arr[idx], rest, fmt.Sprintf("%s[%d]", currentPath, idx))

	case tokenSlice:
		return e.evalSlice(node, tok.slice, rest, currentPath)

	case tokenUnion:
		return e.evalUnion(node, tok, rest, currentPath)

	case tokenRecursive:
		return e.evalRecursive(node, rest, currentPath, 0)

	case tokenFilter:
		return e.evalFilter(node, tok.filter, rest, currentPath)

	default:
		return nil, &Error{Code: ErrInvalidPath, Message: fmt.Sprintf("unknown token kind: %d", tok.kind)}
	}
}

func (e *engine) evalWildcard(node interface{}, rest []token, currentPath string) ([]Result, error) {
	var results []Result
	switch v := node.(type) {
	case map[string]interface{}:
		// sort keys for deterministic output
		keys := sortedKeys(v)
		for _, k := range keys {
			r, err := e.evaluate(v[k], rest, currentPath+"."+k)
			if err != nil {
				return nil, err
			}
			results = append(results, r...)
		}
	case []interface{}:
		for i, item := range v {
			r, err := e.evaluate(item, rest, fmt.Sprintf("%s[%d]", currentPath, i))
			if err != nil {
				return nil, err
			}
			results = append(results, r...)
		}
	}
	return results, nil
}

func (e *engine) evalSlice(node interface{}, slice [3]*int, rest []token, currentPath string) ([]Result, error) {
	arr, ok := node.([]interface{})
	if !ok {
		return nil, nil
	}
	n := len(arr)

	step := 1
	if slice[2] != nil {
		step = *slice[2]
		if step == 0 {
			return nil, &Error{Code: ErrInvalidPath, Message: "slice step cannot be zero"}
		}
	}

	var start, end int
	if step > 0 {
		start = 0
		end = n
	} else {
		start = n - 1
		end = -n - 1
	}

	if slice[0] != nil {
		start = normalizeIndex(*slice[0], n)
	}
	if slice[1] != nil {
		end = normalizeIndex(*slice[1], n)
	}

	var results []Result
	if step > 0 {
		for i := start; i < end && i < n; i += step {
			if i < 0 {
				continue
			}
			r, err := e.evaluate(arr[i], rest, fmt.Sprintf("%s[%d]", currentPath, i))
			if err != nil {
				return nil, err
			}
			results = append(results, r...)
		}
	} else {
		for i := start; i > end && i >= 0; i += step {
			if i >= n {
				continue
			}
			r, err := e.evaluate(arr[i], rest, fmt.Sprintf("%s[%d]", currentPath, i))
			if err != nil {
				return nil, err
			}
			results = append(results, r...)
		}
	}
	return results, nil
}

func (e *engine) evalUnion(node interface{}, tok token, rest []token, currentPath string) ([]Result, error) {
	var results []Result

	if len(tok.indices) > 0 {
		arr, ok := node.([]interface{})
		if !ok {
			return nil, nil
		}
		for _, idx := range tok.indices {
			i := normalizeIndex(idx, len(arr))
			if i < 0 || i >= len(arr) {
				continue
			}
			r, err := e.evaluate(arr[i], rest, fmt.Sprintf("%s[%d]", currentPath, i))
			if err != nil {
				return nil, err
			}
			results = append(results, r...)
		}
	} else {
		obj, ok := node.(map[string]interface{})
		if !ok {
			return nil, nil
		}
		for _, key := range tok.keys {
			val, exists := obj[key]
			if !exists {
				continue
			}
			r, err := e.evaluate(val, rest, currentPath+"."+key)
			if err != nil {
				return nil, err
			}
			results = append(results, r...)
		}
	}
	return results, nil
}

func (e *engine) evalRecursive(node interface{}, rest []token, currentPath string, depth int) ([]Result, error) {
	if e.maxDepth > 0 && depth > e.maxDepth {
		return nil, &Error{Code: ErrMaxDepthExceeded, Message: fmt.Sprintf("max depth %d exceeded", e.maxDepth)}
	}

	select {
	case <-e.ctx.Done():
		return nil, &Error{Code: ErrCancelled, Message: "context cancelled"}
	default:
	}

	var results []Result

	// Apply rest tokens to current node
	if len(rest) > 0 {
		r, err := e.evaluate(node, rest, currentPath)
		if err != nil {
			return nil, err
		}
		results = append(results, r...)
	} else {
		results = append(results, Result{Path: currentPath, Value: node})
	}

	// Recurse into children
	switch v := node.(type) {
	case map[string]interface{}:
		keys := sortedKeys(v)
		for _, k := range keys {
			r, err := e.evalRecursive(v[k], rest, currentPath+"."+k, depth+1)
			if err != nil {
				return nil, err
			}
			results = append(results, r...)
		}
	case []interface{}:
		for i, item := range v {
			r, err := e.evalRecursive(item, rest, fmt.Sprintf("%s[%d]", currentPath, i), depth+1)
			if err != nil {
				return nil, err
			}
			results = append(results, r...)
		}
	}

	return results, nil
}

func (e *engine) evalFilter(node interface{}, expr string, rest []token, currentPath string) ([]Result, error) {
	var results []Result

	evalItem := func(item interface{}, itemPath string) error {
		ok, err := evalFilterExpr(item, expr)
		if err != nil {
			return err
		}
		if ok {
			r, err := e.evaluate(item, rest, itemPath)
			if err != nil {
				return err
			}
			results = append(results, r...)
		}
		return nil
	}

	switch v := node.(type) {
	case []interface{}:
		for i, item := range v {
			if err := evalItem(item, fmt.Sprintf("%s[%d]", currentPath, i)); err != nil {
				return nil, err
			}
		}
	case map[string]interface{}:
		keys := sortedKeys(v)
		for _, k := range keys {
			if err := evalItem(v[k], currentPath+"."+k); err != nil {
				return nil, err
			}
		}
	}

	return results, nil
}

// --- Filter expression evaluator ---

// evalFilterExpr evaluates a filter expression like @.price < 30 against a node.
// Supports: comparison operators (<, >, <=, >=, ==, !=), existence (@.key),
// regex (@.key =~ /pattern/), and logical operators (&& and ||).
func evalFilterExpr(node interface{}, expr string) (bool, error) {
	expr = strings.TrimSpace(expr)

	// Logical OR (lowest precedence)
	if idx := findLogicalOp(expr, "||"); idx >= 0 {
		left, err := evalFilterExpr(node, expr[:idx])
		if err != nil {
			return false, err
		}
		if left {
			return true, nil
		}
		return evalFilterExpr(node, expr[idx+2:])
	}

	// Logical AND
	if idx := findLogicalOp(expr, "&&"); idx >= 0 {
		left, err := evalFilterExpr(node, expr[:idx])
		if err != nil {
			return false, err
		}
		if !left {
			return false, nil
		}
		return evalFilterExpr(node, expr[idx+2:])
	}

	// Parenthesized expression
	if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
		return evalFilterExpr(node, expr[1:len(expr)-1])
	}

	// Regex: @.key =~ /pattern/
	regexRE := regexp.MustCompile(`^(@[\w.\[\]'"*]+)\s*=~\s*/(.+)/([gimsuy]*)$`)
	if m := regexRE.FindStringSubmatch(expr); m != nil {
		lv, err := resolveFilterValue(node, m[1])
		if err != nil {
			return false, nil
		}
		s, ok := lv.(string)
		if !ok {
			return false, nil
		}
		re, err := regexp.Compile(m[2])
		if err != nil {
			return false, &Error{Code: ErrInvalidFilter, Message: fmt.Sprintf("invalid regex: %v", err)}
		}
		return re.MatchString(s), nil
	}

	// Comparison: lhs op rhs
	compRE := regexp.MustCompile(`^(.+?)\s*(==|!=|<=|>=|<|>)\s*(.+)$`)
	if m := compRE.FindStringSubmatch(expr); m != nil {
		lhs, op, rhs := strings.TrimSpace(m[1]), m[2], strings.TrimSpace(m[3])
		lv, lerr := resolveFilterValue(node, lhs)
		rv, rerr := resolveFilterValue(node, rhs)
		if lerr != nil || rerr != nil {
			return false, nil
		}
		return compareValues(lv, op, rv)
	}

	// Existence check: @.key
	if strings.HasPrefix(expr, "@") {
		val, err := resolveFilterValue(node, expr)
		return err == nil && val != nil, nil
	}

	return false, &Error{Code: ErrInvalidFilter, Message: fmt.Sprintf("cannot parse filter expression: %s", expr)}
}

// findLogicalOp finds the position of a logical operator outside parentheses.
func findLogicalOp(expr, op string) int {
	depth := 0
	for i := 0; i < len(expr)-len(op)+1; i++ {
		switch expr[i] {
		case '(':
			depth++
		case ')':
			depth--
		}
		if depth == 0 && expr[i:i+len(op)] == op {
			return i
		}
	}
	return -1
}

// resolveFilterValue resolves a filter operand, which may be a path (@.key) or a literal.
func resolveFilterValue(node interface{}, operand string) (interface{}, error) {
	operand = strings.TrimSpace(operand)

	if strings.HasPrefix(operand, "@") {
		// Path relative to current node
		subPath := "$" + operand[1:]
		e := &engine{maxDepth: 10, ctx: context.Background()}
		tokens, err := tokenize(subPath)
		if err != nil {
			return nil, err
		}
		results, err := e.evaluate(node, tokens, "$")
		if err != nil || len(results) == 0 {
			return nil, fmt.Errorf("not found")
		}
		return results[0].Value, nil
	}

	// String literal
	if (strings.HasPrefix(operand, "'") && strings.HasSuffix(operand, "'")) ||
		(strings.HasPrefix(operand, `"`) && strings.HasSuffix(operand, `"`)) {
		return operand[1 : len(operand)-1], nil
	}

	// Boolean
	if operand == "true" {
		return true, nil
	}
	if operand == "false" {
		return false, nil
	}
	if operand == "null" {
		return nil, nil
	}

	// Number
	if n, err := strconv.ParseFloat(operand, 64); err == nil {
		return n, nil
	}

	return nil, fmt.Errorf("cannot resolve operand: %s", operand)
}

func compareValues(lv interface{}, op string, rv interface{}) (bool, error) {
	// Normalize numbers to float64
	lf, lok := toFloat64(lv)
	rf, rok := toFloat64(rv)

	if lok && rok {
		switch op {
		case "==":
			return lf == rf, nil
		case "!=":
			return lf != rf, nil
		case "<":
			return lf < rf, nil
		case "<=":
			return lf <= rf, nil
		case ">":
			return lf > rf, nil
		case ">=":
			return lf >= rf, nil
		}
	}

	// String comparison
	ls := fmt.Sprintf("%v", lv)
	rs := fmt.Sprintf("%v", rv)
	switch op {
	case "==":
		return ls == rs, nil
	case "!=":
		return ls != rs, nil
	case "<":
		return ls < rs, nil
	case "<=":
		return ls <= rs, nil
	case ">":
		return ls > rs, nil
	case ">=":
		return ls >= rs, nil
	}

	return false, nil
}

func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return math.NaN(), false
}

func normalizeIndex(idx, length int) int {
	if idx < 0 {
		return length + idx
	}
	return idx
}

func sortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Simple insertion sort — small maps, no need for full sort import
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

// Ensure reflect is used (for potential future struct traversal)
var _ = reflect.TypeOf
