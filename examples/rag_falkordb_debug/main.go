package main

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/smallnest/langgraphgo/rag/store"
)

func main() {
	ctx := context.Background()

	// Test direct Redis connection first
	fmt.Println("Testing direct Redis connection...")
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Ping Redis
	pong, err := client.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	fmt.Printf("Redis PING response: %s\n", pong)

	// Test FalkorDB module existence
	fmt.Println("\nTesting FalkorDB module...")
	// Try to check if GRAPH command exists
	res, err := client.Do(ctx, "COMMAND", "INFO", "GRAPH.QUERY").Result()
	if err != nil {
		log.Printf("FalkorDB module might not be loaded: %v", err)
	} else {
		fmt.Printf("GRAPH command info: %v\n", res)
	}

	// Test simple FalkorDB query
	fmt.Println("\nTesting simple FalkorDB query...")
	res, err = client.Do(ctx, "GRAPH.QUERY", "test", "RETURN 1", "--compact").Result()
	if err != nil {
		log.Printf("Failed to execute FalkorDB query: %v", err)
		return
	}

	fmt.Printf("FalkorDB response type: %T\n", res)
	fmt.Printf("FalkorDB response value: %v\n", res)

	// Check response structure
	if r, ok := res.([]any); ok {
		fmt.Printf("Response is []any with length: %d\n", len(r))
		for i, v := range r {
			fmt.Printf("  [%d] type: %T, value: %v\n", i, v, v)
			if innerSlice, ok := v.([]any); ok {
				fmt.Printf("      Inner slice length: %d\n", len(innerSlice))
				for j, innerV := range innerSlice {
					fmt.Printf("        [%d] type: %T, value: %v\n", j, innerV, innerV)
				}
			}
		}
	} else {
		fmt.Printf("Response is not []any, it's %s\n", reflect.TypeOf(res).String())
	}

	// Test creating a graph node
	fmt.Println("\nTesting node creation...")
	res2, err := client.Do(ctx, "GRAPH.QUERY", "test", "CREATE (n:Person {name: 'Test'})", "--compact").Result()
	if err != nil {
		log.Printf("Failed to create node: %v", err)
	} else {
		fmt.Printf("Node creation response: %v\n", res2)
	}

	// Test querying the created node
	fmt.Println("\nTesting node query...")
	res3, err := client.Do(ctx, "GRAPH.QUERY", "test", "MATCH (n:Person) RETURN n", "--compact").Result()
	if err != nil {
		log.Printf("Failed to query nodes: %v", err)
	} else {
		fmt.Printf("Node query response: %v\n", res3)
		if r, ok := res3.([]any); ok {
			fmt.Printf("Query response length: %d\n", len(r))
			for i, v := range r {
				fmt.Printf("  [%d] type: %T\n", i, v)
			}
		}
	}

	// Test using the FalkorDB Graph wrapper
	fmt.Println("\nTesting FalkorDB Graph wrapper...")
	g := store.NewGraph("test", client)

	queryResult, err := g.Query(ctx, "MATCH (n:Person) RETURN n")
	if err != nil {
		log.Printf("Failed to query with wrapper: %v", err)
	} else {
		fmt.Printf("Wrapper query result:\n")
		fmt.Printf("  Header: %v\n", queryResult.Header)
		fmt.Printf("  Results count: %d\n", len(queryResult.Results))
		fmt.Printf("  Statistics: %v\n", queryResult.Statistics)
	}

	// Test MERGE operation (what AddEntity uses)
	fmt.Println("\nTesting MERGE operation (like AddEntity)...")
	mergeResult, err := g.Query(ctx, "MERGE (n:Company {id: 'apple', name: 'Apple'}) RETURN n")
	if err != nil {
		log.Printf("Failed to execute MERGE: %v", err)
	} else {
		fmt.Printf("MERGE response:\n")
		fmt.Printf("  Header: %v\n", mergeResult.Header)
		fmt.Printf("  Results count: %d\n", len(mergeResult.Results))
		fmt.Printf("  Statistics: %v\n", mergeResult.Statistics)

		// Try to parse the result like the AddEntity does
		if len(mergeResult.Results) > 0 {
			fmt.Printf("  First result: %v\n", mergeResult.Results[0])
		}
	}

	// Test entity creation with the same format as the example
	fmt.Println("\nTesting entity creation like in the example...")

	// Test propsToString function
	testProps := map[string]any{
		"name":        "Apple Inc.",
		"type":        "ORGANIZATION",
		"description": "Technology company",
	}
	propsStr := propsToString(testProps)
	fmt.Printf("Props string: %s\n", propsStr)

	// Test the actual query format
	testQuery := fmt.Sprintf("MERGE (n:%s {id: '%s'}) SET n += %s", "ORGANIZATION", "apple", propsStr)
	fmt.Printf("Test query: %s\n", testQuery)

	// Test direct Redis call to see the raw response
	fmt.Println("Testing direct Redis call for the problematic query...")
	rawResponse, err := client.Do(ctx, "GRAPH.QUERY", "test", testQuery, "--compact").Result()
	if err != nil {
		log.Printf("Failed raw query: %v", err)
	} else {
		fmt.Printf("Raw response type: %T\n", rawResponse)
		fmt.Printf("Raw response: %v\n", rawResponse)
		if r, ok := rawResponse.([]any); ok {
			fmt.Printf("Raw response length: %d\n", len(r))
			for i, v := range r {
				fmt.Printf("  [%d] type: %T, value: %v\n", i, v, v)
			}
		}
	}

	entityResult, err := g.Query(ctx, testQuery)
	if err != nil {
		log.Printf("Failed to create entity like example: %v", err)
	} else {
		fmt.Printf("Entity creation response:\n")
		fmt.Printf("  Header: %v\n", entityResult.Header)
		fmt.Printf("  Results count: %d\n", len(entityResult.Results))
		fmt.Printf("  Statistics: %v\n", entityResult.Statistics)
		if len(entityResult.Results) > 0 {
			fmt.Printf("  First result type: %T\n", entityResult.Results[0])
		}
	}

	// Clean up
	client.Close()
}

// Helper function to test propsToString
func propsToString(m map[string]any) string {
	parts := []string{}
	for k, v := range m {
		var val any
		switch v := v.(type) {
		case []float32:
			// Convert to Cypher list: [v1, v2, ...]
			s := make([]string, len(v))
			for i, f := range v {
				s[i] = fmt.Sprintf("%f", f)
			}
			val = "[" + strings.Join(s, ",") + "]"
		default:
			val = quoteString(v)
		}
		parts = append(parts, fmt.Sprintf("%s: %v", k, val))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func quoteString(i any) any {
	switch x := i.(type) {
	case string:
		if len(x) == 0 {
			return "\"\""
		}
		if x[0] != '"' {
			x = "\"" + x
		}
		if x[len(x)-1] != '"' {
			x += "\""
		}
		return x
	default:
		return i
	}
}
