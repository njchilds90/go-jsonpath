package jsonpath_test

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/njchilds90/go-jsonpath"
)

func ExampleQuery() {
	data := []byte(`{"store":{"book":[{"title":"Go Programming","price":29.99},{"title":"Clean Code","price":34.99}]}}`)

	results, err := jsonpath.Query(data, "$.store.book[*].title")
	if err != nil {
		log.Fatal(err)
	}
	for _, r := range results {
		fmt.Println(r.Value)
	}
	// Output:
	// Go Programming
	// Clean Code
}

func ExampleFirst() {
	data := []byte(`{"user":{"name":"Alice","role":"admin"}}`)

	result, err := jsonpath.First(data, "$.user.name")
	if err != nil {
		log.Fatal(err)
	}
	if result != nil {
		fmt.Println(result.Value)
	}
	// Output:
	// Alice
}

func ExampleExists() {
	data := []byte(`{"feature":{"enabled":true}}`)

	ok, err := jsonpath.Exists(data, "$.feature.enabled")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(ok)
	// Output:
	// true
}

func ExampleCompile() {
	cp := jsonpath.MustCompile("$.store.book[*].price")

	doc1 := []byte(`{"store":{"book":[{"price":9.99},{"price":14.99}]}}`)
	doc2 := []byte(`{"store":{"book":[{"price":4.99}]}}`)

	for _, doc := range [][]byte{doc1, doc2} {
		vals, _ := cp.Query(doc)
		for _, v := range vals {
			fmt.Println(v.Value)
		}
	}
	// Output:
	// 9.99
	// 14.99
	// 4.99
}

func ExampleQuery_filter() {
	data := []byte(`{"products":[{"name":"Widget","price":5.00},{"name":"Gadget","price":25.00},{"name":"Doohickey","price":8.50}]}`)

	results, err := jsonpath.Query(data, "$.products[?(@.price < 10)].name")
	if err != nil {
		log.Fatal(err)
	}
	for _, r := range results {
		fmt.Println(r.Value)
	}
	// Output:
	// Widget
	// Doohickey
}

func ExampleValues() {
	data := []byte(`{"scores":[10,20,30,40]}`)

	vals, err := jsonpath.Values(data, "$.scores[*]")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(vals))
	// Output:
	// 4
}

func ExampleQuery_recursiveDescent() {
	data := []byte(`{"a":{"price":1},"b":{"c":{"price":2}}}`)

	results, err := jsonpath.Query(data, "$..price")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(results))
	// Output:
	// 2
}

func ExampleResult_MarshalJSON() {
	data := []byte(`{"key":"value"}`)
	results, _ := jsonpath.Query(data, "$.key")
	b, _ := json.Marshal(results[0])
	fmt.Println(string(b))
	// Output:
	// {"path":"$.key","value":"value"}
}
