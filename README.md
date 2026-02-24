# go-jsonpath

[![Go Reference](https://pkg.go.dev/badge/github.com/njchilds90/go-jsonpath.svg)](https://pkg.go.dev/github.com/njchilds90/go-jsonpath)
[![CI](https://github.com/njchilds90/go-jsonpath/actions/workflows/ci.yml/badge.svg)](https://github.com/njchilds90/go-jsonpath/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/njchilds90/go-jsonpath)](https://goreportcard.com/report/github.com/njchilds90/go-jsonpath)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A complete, spec-compliant JSONPath query engine for Go. Zero dependencies. Deterministic. AI-agent ready.

## Why?

JSONPath is the XPath of JSON — a query language for extracting, filtering, and navigating nested JSON without marshalling it into rigid structs. Python has `jsonpath-ng`. JavaScript has `jsonpath-plus`. Go had no complete, well-maintained, idiomatic implementation — until now.

## Install
```bash
go get github.com/njchilds90/go-jsonpath
```

## Quick Start
```go
import "github.com/njchilds90/go-jsonpath"

data := []byte(`{"store":{"book":[{"title":"Go Programming","price":29.99},{"title":"Clean Code","price":34.99}]}}`)

// Get all book titles
results, err := jsonpath.Query(data, "$.store.book[*].title")

// Get first match
result, err := jsonpath.First(data, "$.store.book[0].title")

// Just values
vals, err := jsonpath.Values(data, "$.store.book[*].price")

// Check existence
ok, err := jsonpath.Exists(data, "$.store.book[0].isbn")

// Pre-compile for repeated use (faster)
cp := jsonpath.MustCompile("$.store.book[*].price")
results, err = cp.Query(data)
```

## Supported Syntax

| Expression | Meaning |
|---|---|
| `$` | Root element |
| `.key` | Child key |
| `['key']` | Child key (bracket notation) |
| `[*]` | All children (wildcard) |
| `[0]` | Array index |
| `[-1]` | Last element (negative index) |
| `[0:2]` | Slice (start:end) |
| `[::2]` | Slice with step |
| `..key` | Recursive descent |
| `[?(@.price < 10)]` | Filter expression |
| `[0,2]` | Union of indices |
| `['a','b']` | Union of keys |

## Filter Expressions
```go
// Comparison operators
jsonpath.Query(data, "$.book[?(@.price < 10)]")
jsonpath.Query(data, "$.book[?(@.price >= 20)]")
jsonpath.Query(data, `$.book[?(@.category == "fiction")]`)
jsonpath.Query(data, `$.book[?(@.title != "Draft")]`)

// Existence check
jsonpath.Query(data, "$.book[?(@.isbn)]")

// Logical AND / OR
jsonpath.Query(data, "$.book[?(@.price > 5 && @.price < 15)]")
jsonpath.Query(data, "$.book[?(@.price < 9 || @.price > 20)]")

// Regex
jsonpath.Query(data, "$.book[?(@.title =~ /Go/)]")
```

## Context Support
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

results, err := jsonpath.QueryContext(ctx, data, "$..price")
```

## Options
```go
// Strict mode: return errors for missing keys instead of empty results
results, err := jsonpath.Query(data, "$.missing", jsonpath.WithAllowMissingKeys(true))

// Limit recursive descent depth (default: 100)
results, err := jsonpath.Query(data, "$..key", jsonpath.WithMaxDepth(20))
```

## Structured Errors
```go
results, err := jsonpath.Query(data, "$.key")
if err != nil {
    if jsonpath.IsPathError(err) { /* malformed expression */ }
    if jsonpath.IsJSONError(err) { /* bad JSON input */ }
    if jsonpath.IsNotFound(err)  { /* strict mode: missing key */ }
    if jsonpath.IsCancelled(err) { /* context cancelled */ }
}
```

## AI Agent Design

This library is designed for safe use in AI agent pipelines:

- **Deterministic** — identical inputs always produce identical outputs in the same order
- **No global state** — safe for concurrent use
- **Structured errors** — programmatically distinguishable error codes
- **Context support** — cancel long-running queries
- **Pure functions** — no side effects, no mutation of input data
- **Composable** — `CompiledPath` can be reused across many documents

## Performance

Pre-compile paths for best performance:
```go
cp := jsonpath.MustCompile("$..price")
// Reuse cp across thousands of documents
for _, doc := range docs {
    results, _ := cp.Query(doc)
}
```

## License

MIT
