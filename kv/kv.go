// Package kv defines the interface for per-path private key-value storage
// accessible to JavaScript running at a path via the hput global object.
//
// Each path's store is fully isolated — JS at /x cannot access /y's keys.
// All implementations must honour this isolation contract.
package kv

import "context"

// ListResult is returned by List.
type ListResult struct {
	Keys   []string
	Cursor string // opaque; pass back to List to get the next page. Empty means no more pages.
}

// ListOptions controls filtering and pagination for List.
type ListOptions struct {
	Prefix string // only return keys with this prefix; empty means all keys
	Limit  int    // max keys to return per call; 0 means no limit
	Cursor string // resume token from a previous ListResult; empty means start from beginning
}

// KV is the interface for per-path private key-value storage.
// All operations are scoped to a path — implementations must not allow
// one path to read or write another path's keys.
type KV interface {
	// Get retrieves the value stored at key within path's namespace.
	// Returns nil, nil if the key does not exist.
	Get(ctx context.Context, path, key string) ([]byte, error)

	// Put stores value at key within path's namespace.
	Put(ctx context.Context, path, key string, value []byte) error

	// Delete removes the key from path's namespace. No-op if key does not exist.
	Delete(ctx context.Context, path, key string) error

	// List returns keys in path's namespace, optionally filtered and paginated.
	List(ctx context.Context, path string, opts ListOptions) (ListResult, error)

	// Close releases any resources held by the store.
	Close() error
}
