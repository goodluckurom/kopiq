package context

import "kopiq/internal/domain"

// Provider snapshots the frontmost application place (D-11).
type Provider interface {
	// Capture returns application identity and optional deep fields.
	Capture(deep bool) (domain.Snapshot, error)
}
