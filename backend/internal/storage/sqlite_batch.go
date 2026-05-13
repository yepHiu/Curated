package storage

import "strings"

const sqliteInClauseBatchSize = 500

func forEachInClauseBatch[T any](values []T, fn func([]T) error) error {
	for start := 0; start < len(values); start += sqliteInClauseBatchSize {
		end := start + sqliteInClauseBatchSize
		if end > len(values) {
			end = len(values)
		}
		if err := fn(values[start:end]); err != nil {
			return err
		}
	}
	return nil
}

func inClausePlaceholders(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", count), ",")
}

func inClauseArgs[T any](values []T) []any {
	args := make([]any, len(values))
	for i, value := range values {
		args[i] = value
	}
	return args
}
