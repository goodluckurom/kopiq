package dock

// SetVisible shows or hides the app in the macOS Dock (live, no restart).
// On non-macOS platforms this is a no-op.
func SetVisible(show bool) {
	setVisible(show)
}
