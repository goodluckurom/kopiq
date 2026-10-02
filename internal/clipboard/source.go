package clipboard

import (
	"kopiq/internal/domain"
)

// Source is the platform clipboard boundary (D-08).
type Source interface {
	// Changes returns a channel of generation tokens. Closed when Stop is called.
	Changes() <-chan domain.Generation
	// Read loads every representation for that generation.
	Read(gen domain.Generation) (domain.RawPayload, error)
	// Write places text on the clipboard and returns the new self-owned generation.
	Write(text string) (domain.Generation, error)
	// WriteImage places PNG bytes on the clipboard and returns the new generation.
	WriteImage(png []byte) (domain.Generation, error)
	// CurrentGeneration returns the OS counter without reading payload.
	CurrentGeneration() (domain.Generation, error)
	// Start begins watching. Baseline generation is recorded and not emitted.
	Start() error
	// Stop ends watching.
	Stop() error
	// SupportsBackgroundWatch reports whether background clipboard events work.
	SupportsBackgroundWatch() bool
}

// Ring is a small in-memory set of recent generation tokens.
type Ring struct {
	items []domain.Generation
	cap   int
}

// NewRing creates a ring with the given capacity.
func NewRing(capacity int) *Ring {
	return &Ring{cap: capacity}
}

func (r *Ring) Add(g domain.Generation) {
	for _, x := range r.items {
		if x == g {
			return
		}
	}
	r.items = append(r.items, g)
	if len(r.items) > r.cap {
		r.items = r.items[len(r.items)-r.cap:]
	}
}

func (r *Ring) Contains(g domain.Generation) bool {
	for _, x := range r.items {
		if x == g {
			return true
		}
	}
	return false
}
