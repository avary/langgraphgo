package graph

import (
	"fmt"
	"reflect"

	"github.com/smallnest/langgraphgo/llms"
)

// MessageWithID is an interface that allows messages to have an ID for deduplication/upsert.
// Since langchaingo's MessageContent doesn't have an ID field, we can wrap it or use a custom struct.
// For now, we'll check if the message implements this interface or is a map with an "id" key.
type MessageWithID interface {
	GetID() string
	GetContent() llms.MessageContent
}

// AddMessages is a reducer designed for merging chat messages.
// It handles ID-based deduplication and upserts.
// If a new message has the same ID as an existing one, it replaces the existing one.
// Otherwise, it appends the new llms.
func AddMessages(current, new any) (any, error) {
	if current == nil {
		return new, nil
	}

	// Optimization: Fast path for []llms.MessageContent
	if currentSlice, ok := current.([]llms.MessageContent); ok {
		var newSlice []llms.MessageContent
		if ns, ok := new.([]llms.MessageContent); ok {
			newSlice = ns
		} else if n, ok := new.(llms.MessageContent); ok {
			newSlice = []llms.MessageContent{n}
		} else {
			// Fallback to reflection if new is not compatible
			goto ReflectionPath
		}

		// Since standard MessageContent doesn't support IDs in this implementation
		// (unless wrapped), and we are in the fast path for standard types,
		// we just append.
		return append(currentSlice, newSlice...), nil
	}

ReflectionPath:
	// We expect current to be a slice of messages
	currentVal := reflect.ValueOf(current)
	if currentVal.Kind() != reflect.Slice {
		return nil, fmt.Errorf("current value is not a slice")
	}

	// We expect new to be a slice of messages or a single message
	newVal := reflect.ValueOf(new)
	var newMessages []any

	if newVal.Kind() == reflect.Slice {
		for i := 0; i < newVal.Len(); i++ {
			newMessages = append(newMessages, newVal.Index(i).Interface())
		}
	} else {
		newMessages = append(newMessages, new)
	}

	// Convert current slice to a list of interfaces for manipulation
	result := make([]any, 0, currentVal.Len()+len(newMessages))
	for i := 0; i < currentVal.Len(); i++ {
		result = append(result, currentVal.Index(i).Interface())
	}

	// Index existing messages by ID if possible
	// Since standard MessageContent doesn't have ID, we only support ID logic
	// if the user uses a custom struct or map wrapper.
	// For standard MessageContent, we just append.

	// Map ID to index in result
	idToIndex := make(map[string]int)
	for i, msg := range result {
		if id := getMessageID(msg); id != "" {
			idToIndex[id] = i
		}
	}

	for _, msg := range newMessages {
		id := getMessageID(msg)
		if id != "" {
			if idx, exists := idToIndex[id]; exists {
				// Update existing message
				result[idx] = msg
			} else {
				// Append new message
				result = append(result, msg)
				idToIndex[id] = len(result) - 1
			}
		} else {
			// No ID, just append
			result = append(result, msg)
		}
	}

	// Convert back to the original slice type if possible, or []any
	// If current was []llms.MessageContent, we try to return that.
	// But if we mixed types (e.g. wrapped messages), we might need to return []any
	// or fail if types are incompatible.

	// For simplicity in this implementation, if the original type was []llms.MessageContent,
	// and we are just appending standard messages, we return that type.
	// If we are doing advanced ID stuff, we assume the user is using a compatible slice type.

	targetType := currentVal.Type()
	finalSlice := reflect.MakeSlice(targetType, 0, len(result))

	for _, item := range result {
		val := reflect.ValueOf(item)
		if val.Type().AssignableTo(targetType.Elem()) {
			finalSlice = reflect.Append(finalSlice, val)
		} else {
			// Try to convert? Or error?
			// If we can't put it back in the slice, we have a problem.
			return nil, fmt.Errorf("cannot append item of type %T to slice of %s", item, targetType.Elem())
		}
	}

	return finalSlice.Interface(), nil
}

// getMessageID tries to extract an ID from a message object.
func getMessageID(msg any) string {
	// 1. Check if it implements MessageWithID
	if m, ok := msg.(MessageWithID); ok {
		return m.GetID()
	}

	// 2. Check if it's a map with an "id" key
	if m, ok := msg.(map[string]any); ok {
		if id, ok := m["id"].(string); ok {
			return id
		}
	}

	// 3. Check specific struct fields via reflection (slow but flexible)
	val := reflect.ValueOf(msg)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() == reflect.Struct {
		field := val.FieldByName("ID")
		if field.IsValid() && field.Kind() == reflect.String {
			return field.String()
		}
	}

	return ""
}
