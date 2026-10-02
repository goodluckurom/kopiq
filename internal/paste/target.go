package paste

import "fmt"

// Target is the application that should receive paste-back.
type Target struct {
	ProcessID       int
	ApplicationID   string
	ApplicationName string
}

// Result describes what PasteBack did.
type Result struct {
	// Pasted is true when we activated the target and synthesized paste.
	Pasted bool
	// CopiedOnly is true when we only wrote the clipboard (manual paste needed).
	CopiedOnly bool
	// Message is a short UI hint when Pasted is false.
	Message string
}

func (t Target) Valid() bool {
	return t.ProcessID > 0 || t.ApplicationID != ""
}

func (t Target) String() string {
	if t.ApplicationName != "" {
		return t.ApplicationName
	}
	if t.ApplicationID != "" {
		return t.ApplicationID
	}
	return fmt.Sprintf("pid:%d", t.ProcessID)
}
