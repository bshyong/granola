// Package cache provides functionality for reading and parsing the Granola local cache file.
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// TranscriptSegment represents a single segment of speech in a transcript.
type TranscriptSegment struct {
	ID             string `json:"id"`
	DocumentID     string `json:"document_id"`
	StartTimestamp string `json:"start_timestamp"`
	EndTimestamp   string `json:"end_timestamp"`
	Text           string `json:"text"`
	Source         string `json:"source"` // "system" or "microphone"
	IsFinal        bool   `json:"is_final"`
}

// Document represents a meeting document from the cache.
type Document struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// CacheData contains the parsed cache data.
type CacheData struct {
	Documents   map[string]Document                `json:"documents"`
	Transcripts map[string][]TranscriptSegment     `json:"transcripts"`
}

// ReadCache reads and parses the Granola cache file.
func ReadCache(cachePath string) (*CacheData, error) {
	// Read cache file
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	// Parse outer JSON — the "cache" field may be a JSON string (v3) or a direct object (v6+)
	var raw struct {
		Cache json.RawMessage `json:"cache"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse cache JSON: %w", err)
	}

	// Determine format: if cache is a string, decode it; otherwise use directly
	var cacheBytes []byte
	var cacheStr string
	if err := json.Unmarshal(raw.Cache, &cacheStr); err == nil {
		// v3 format: cache is a JSON-encoded string
		cacheBytes = []byte(cacheStr)
	} else {
		// v6+ format: cache is a direct JSON object
		cacheBytes = raw.Cache
	}

	// Parse inner JSON (the actual cache data)
	var inner struct {
		State struct {
			Documents   map[string]json.RawMessage `json:"documents"`
			Transcripts map[string]json.RawMessage `json:"transcripts"`
		} `json:"state"`
	}

	if err := json.Unmarshal(cacheBytes, &inner); err != nil {
		return nil, fmt.Errorf("failed to parse cache state: %w", err)
	}

	// Parse documents
	documents := make(map[string]Document)
	for id, raw := range inner.State.Documents {
		var doc Document
		if err := json.Unmarshal(raw, &doc); err != nil {
			// Skip documents that fail to parse
			continue
		}
		doc.ID = id // Ensure ID is set
		documents[id] = doc
	}

	// Parse transcripts
	transcripts := make(map[string][]TranscriptSegment)
	for id, raw := range inner.State.Transcripts {
		var segments []TranscriptSegment
		if err := json.Unmarshal(raw, &segments); err != nil {
			// Skip transcripts that fail to parse
			continue
		}
		transcripts[id] = segments
	}

	return &CacheData{
		Documents:   documents,
		Transcripts: transcripts,
	}, nil
}

// GetDefaultCachePath returns the default cache file path for the current platform.
// It checks for cache files in descending version order and returns the first one found.
func GetDefaultCachePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// macOS path — check for cache files in descending version order
	granolaDir := filepath.Join(home, "Library", "Application Support", "Granola")
	for v := 10; v >= 3; v-- {
		path := filepath.Join(granolaDir, fmt.Sprintf("cache-v%d.json", v))
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Fallback to v3 if nothing found
	return filepath.Join(granolaDir, "cache-v3.json")
}
