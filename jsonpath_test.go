package jsonpath_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/njchilds90/go-jsonpath"
)

// sampleJSON is a standard JSONPath test document.
var sampleJSON = []byte(`{
	"store": {
		"book": [
			{"category": "reference", "author": "Nigel Rees", "title": "Sayings of the Century", "price": 8.95},
			{"category": "fiction", "author": "Evelyn Waugh", "title": "Sword of Honour", "price": 12.99},
			{"category": "fiction", "author": "Herman Melville", "title": "Moby Dick", "isbn": "0-553-21311-3", "price": 8.99},
			{"category": "fiction", "author": "J. R. R. Tolkien", "title": "The Lord of the Rings", "isbn": "0-395-19395-8", "price": 22.99}
		],
		"bicycle": {
			"color": "red",
			"price": 19.95
		}
	},
	"expensive": 10
}`)

func TestQueryRoot(t *testing.T) {
	results, err := jsonpath.Query(sampleJSON, "$")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Path != "$" {
		t.Errorf("expected path '$', got %q", results[0].Path)
	}
}

func TestQueryChildKey(t *testing.T) {
	results, err := jsonpath.Query(sampleJSON, "$.expensive")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	v, ok := results[0].Value.(float64)
	if !ok || v != 10 {
		t.Errorf("expected 10, got %v", results[0].Value)
	}
}

func TestQueryNestedKey(t *testing.T) {
	results, err := jsonpath.Query(sampleJSON, "$.store.bicycle.color")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Value != "red" {
		t.Errorf("expected 'red', got %v", results[0].Value)
	}
}

func TestQueryArrayIndex(t *testing.T) {
	results, err := jsonpath.Query(sampleJSON, "$.store.book[0].title")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Value != "Sayings of the Century" {
		t.Errorf("unexpected value: %v", results[0].Value)
	}
}

func TestQueryNegativeIndex(t *testing.T) {
	results, err := jsonpath.Query(sampleJSON, "$.store.book[-1].title")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Value != "The Lord of the Rings" {
		t.Errorf("unexpected value: %v", results[0].Value)
	}
}

func TestQueryWildcardArray(t *testing.T) {
	results, err := jsonpath.Query(sampleJSON, "$.store.book[*].title")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}
}

func TestQueryWildcardObject(t *testing.T) {
	results, err := jsonpath.Query(sampleJSON, "$.store.*")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// store has 2 children: book and bicycle
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestQueryRecursiveDescent(t *testing.T) {
	results, err := jsonpath.Query(sampleJSON, "$..author")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 4 {
		t.Fatalf("expected 4 authors, got %d", len(results))
	}
}

func TestQueryRecursivePrice(t *testing.T) {
	results, err := jsonpath.Query(sampleJSON, "$..price")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 4 book prices + 1 bicycle price = 5
	if len(results) != 5 {
		t.Fatalf("expected 5 prices, got %d", len(results))
	}
}

func TestQuerySlice(t *testing.T) {
	results, err := jsonpath.Query(sampleJSON, "$.store.book[0:2].title")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestQuerySliceStep(t *testing.T) {
	results, err := jsonpath.Query(sampleJSON, "$.store.book[::2].title")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// books at index 0, 2
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestQueryFilterLessThan(t *testing.T) {
	results, err := jsonpath.Query(sampleJSON, "$.store.book[?(@.price < 10)].title")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// prices < 10: 8.95, 8.99
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d: %v", len(results), results)
	}
}

func TestQueryFilterEquals(t *testing.T) {
	results, err := jsonpath.Query(sampleJSON, `$.store.book[?(@.category == "fiction")].title`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 fiction books, got %d", len(results))
	}
}

func TestQueryFilterExistence(t *testing.T) {
	results, err := jsonpath.Query(sampleJSON, "$.store.book[?(@.isbn)].title")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 2 books have isbn
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestQueryUnionIndices(t *testing.T) {
	results, err := jsonpath.Query(sampleJSON, "$.store.book[0,3].title")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestQueryUnionKeys(t *testing.T) {
	data := []byte(`{"a": 1, "b": 2, "c": 3}`)
	results, err := jsonpath.Query(data, "$['a','b']")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestQueryBracketKey(t *testing.T) {
	data := []byte(`{"some-key": "value"}`)
	results, err := jsonpath.Query(data, "$['some-key']")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].Value != "value" {
		t.Errorf("unexpected results: %v", results)
	}
}

func TestFirst(t *testing.T) {
	result, err := jsonpath.First(sampleJSON, "$.store.bicycle.color")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Value != "red" {
		t.Errorf("expected 'red', got %v", result.Value)
	}
}

func TestFirstMissing(t *testing.T) {
	result, err := jsonpath.First(sampleJSON, "$.nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}

func TestValues(t *testing.T) {
	vals, err := jsonpath.Values(sampleJSON, "$.store.book[*].price")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vals) != 4 {
		t.Fatalf("expected 4 values, got %d", len(vals))
	}
}

func TestPaths(t *testing.T) {
	paths, err := jsonpath.Paths(sampleJSON, "$.store.book[*]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(paths) != 4 {
		t.Fatalf("expected 4 paths, got %d", len(paths))
	}
}

func TestExists(t *testing.T) {
	ok, err := jsonpath.Exists(sampleJSON, "$.store.bicycle")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected bicycle to exist")
	}

	ok, err = jsonpath.Exists(sampleJSON, "$.store.motorbike")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected motorbike to not exist")
	}
}

func TestCompile(t *testing.T) {
	cp, err := jsonpath.Compile("$.store.book[*].title")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cp.String() != "$.store.book[*].title" {
		t.Errorf("unexpected string: %s", cp.String())
	}

	results, err := cp.Query(sampleJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}
}

func TestMustCompilePanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid path")
		}
	}()
	jsonpath.MustCompile("invalid")
}

func TestQueryValueContext_Cancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := jsonpath.QueryContext(ctx, sampleJSON, "$..price")
	if err == nil {
		t.Error("expected error for cancelled context")
	}
	if !jsonpath.IsCancelled(err) {
		t.Errorf("expected IsCancelled, got: %v", err)
	}
}

func TestQueryWithTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results, err := jsonpath.QueryContext(ctx, sampleJSON, "$..price")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 5 {
		t.Fatalf("expected 5 results, got %d", len(results))
	}
}

func TestQueryValue(t *testing.T) {
	var doc interface{}
	if err := json.Unmarshal(sampleJSON, &doc); err != nil {
		t.Fatal(err)
	}

	results, err := jsonpath.QueryValue(doc, "$.expensive")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].Value.(float64) != 10 {
		t.Errorf("unexpected results: %v", results)
	}
}

func TestErrorTypes(t *testing.T) {
	_, err := jsonpath.Query(sampleJSON, "invalid")
	if !jsonpath.IsPathError(err) {
		t.Errorf("expected path error, got: %v", err)
	}

	_, err = jsonpath.Query([]byte("not json"), "$")
	if !jsonpath.IsJSONError(err) {
		t.Errorf("expected json error, got: %v", err)
	}
}

func TestStrictMode(t *testing.T) {
	_, err := jsonpath.Query(sampleJSON, "$.nonexistent", jsonpath.WithAllowMissingKeys(true))
	if err == nil {
		t.Error("expected error in strict mode for missing key")
	}
	if !jsonpath.IsNotFound(err) {
		t.Errorf("expected IsNotFound, got: %v", err)
	}
}

func TestResultMarshalJSON(t *testing.T) {
	results, err := jsonpath.Query(sampleJSON, "$.expensive")
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(results[0])
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	if len(b) == 0 {
		t.Error("expected non-empty JSON")
	}
}

func TestFilterLogicalAnd(t *testing.T) {
	results, err := jsonpath.Query(sampleJSON, `$.store.book[?(@.price > 8 && @.price < 10)].title`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 8.95 and 8.99 both qualify
	if len(results) != 2 {
		t.Fatalf("expected 2, got %d: %v", len(results), results)
	}
}

func TestFilterLogicalOr(t *testing.T) {
	results, err := jsonpath.Query(sampleJSON, `$.store.book[?(@.price < 9 || @.price > 20)].title`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 8.95, 8.99, 22.99
	if len(results) != 3 {
		t.Fatalf("expected 3, got %d: %v", len(results), results)
	}
}

func TestMaxDepth(t *testing.T) {
	// Deeply nested JSON
	data := []byte(`{"a":{"b":{"c":{"d":{"e":"deep"}}}}}`)
	_, err := jsonpath.Query(data, "$..e", jsonpath.WithMaxDepth(2))
	if err == nil {
		t.Error("expected max depth error")
	}
}

func TestDeterministicOutput(t *testing.T) {
	// Run same query multiple times and verify consistent ordering
	data := []byte(`{"z":1,"a":2,"m":3}`)
	for i := 0; i < 10; i++ {
		results, err := jsonpath.Query(data, "$.*")
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 3 {
			t.Fatalf("expected 3, got %d", len(results))
		}
		// Keys sorted: a, m, z → values 2, 3, 1
		if results[0].Path != "$.a" || results[1].Path != "$.m" || results[2].Path != "$.z" {
			t.Errorf("non-deterministic output on run %d: %v", i, results)
		}
	}
}

func TestEmptyArray(t *testing.T) {
	data := []byte(`{"items":[]}`)
	results, err := jsonpath.Query(data, "$.items[*]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestNullValue(t *testing.T) {
	data := []byte(`{"key":null}`)
	results, err := jsonpath.Query(data, "$.key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Value != nil {
		t.Errorf("expected nil value, got %v", results[0].Value)
	}
}

func BenchmarkQuery(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = jsonpath.Query(sampleJSON, "$..price")
	}
}

func BenchmarkCompileAndQuery(b *testing.B) {
	cp := jsonpath.MustCompile("$..price")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cp.Query(sampleJSON)
	}
}

func BenchmarkFilter(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = jsonpath.Query(sampleJSON, "$.store.book[?(@.price < 10)].title")
	}
}
