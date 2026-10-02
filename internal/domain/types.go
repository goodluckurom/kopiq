package domain

// ClipboardItem is the list/detail DTO exposed to the React UI via Wails.
type ClipboardItem struct {
	ID             int64        `json:"id"`
	ContentHash    string       `json:"contentHash"`
	Kind           string       `json:"kind"` // text | image | files
	DetectedType   string       `json:"detectedType"`
	Language       string       `json:"language,omitempty"`
	Title          string       `json:"title,omitempty"`
	TextContent    string       `json:"textContent,omitempty"`
	TextPreview    string       `json:"textPreview"`
	MimeType       string       `json:"mimeType,omitempty"`
	BlobPath       string       `json:"blobPath,omitempty"`
	ThumbnailPath  string       `json:"thumbnailPath,omitempty"`
	ByteSize       int64        `json:"byteSize"`
	Width          int          `json:"width,omitempty"`
	Height         int          `json:"height,omitempty"`
	ImageKept      bool         `json:"imageKept"`
	IsPinned       bool         `json:"isPinned"`
	Sensitivity    string       `json:"sensitivity"` // none | suspected
	CopyCount      int          `json:"copyCount"`
	MetadataJSON   string       `json:"metadataJson,omitempty"`
	CreatedAt      int64        `json:"createdAt"`
	LastCopiedAt   int64        `json:"lastCopiedAt"`
	ExpiresAt      *int64       `json:"expiresAt,omitempty"`
	LastContext    *CopyContext `json:"context,omitempty"`
	Masked         bool         `json:"masked,omitempty"`
}

// CopyContext is the place a copy happened.
type CopyContext struct {
	ID              int64  `json:"id"`
	ApplicationName string `json:"applicationName"`
	ApplicationID   string `json:"applicationId"`
	IconCacheKey    string `json:"iconCacheKey,omitempty"`
	ProcessID       int    `json:"processId,omitempty"`
	WindowTitle     string `json:"windowTitle,omitempty"`
	DocumentTitle   string `json:"documentTitle,omitempty"`
	URL             string `json:"url,omitempty"`
	Domain          string `json:"domain,omitempty"`
	FileName        string `json:"fileName,omitempty"`
	FilePath        string `json:"filePath,omitempty"`
	PlaceKey        string `json:"placeKey,omitempty"`
	Deep            bool   `json:"deep"`
	CapturedAt      int64  `json:"capturedAt"`
}

// CopyEvent links an item to a context at a point in time.
type CopyEvent struct {
	ID        int64 `json:"id"`
	ItemID    int64 `json:"itemId"`
	ContextID int64 `json:"contextId"`
	CopiedAt  int64 `json:"copiedAt"`
}

// HistoryFilter narrows HistoryService.List.
type HistoryFilter struct {
	Kind         string `json:"kind,omitempty"`
	DetectedType string `json:"detectedType,omitempty"`
	AppID        string `json:"appId,omitempty"`
	PinnedOnly   bool   `json:"pinnedOnly,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	Offset       int    `json:"offset,omitempty"`
}

// HistoryChangedEvent is emitted on history.changed.
type HistoryChangedEvent struct {
	ItemID int64  `json:"itemId"`
	Action string `json:"action"` // insert | update | delete
}

// AppCount is a source-app tally for the sidebar.
type AppCount struct {
	Name         string `json:"name"`
	AppID        string `json:"appId"`
	Count        int    `json:"count"`
	IconCacheKey string `json:"iconCacheKey,omitempty"`
}

// HistoryCounts powers sidebar category and source chips.
type HistoryCounts struct {
	Total       int            `json:"total"`
	Pinned      int            `json:"pinned"`
	Snippets    int            `json:"snippets"`
	Collections int            `json:"collections"`
	Stacks      int            `json:"stacks"`
	ByCategory  map[string]int `json:"byCategory"`
	ByApp       []AppCount     `json:"byApp"`
}

// Collection is a named folder of clipboard items (KQ-017).
type Collection struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"createdAt"`
	SortOrder int    `json:"sortOrder"`
	ItemCount int    `json:"itemCount"`
}

// Snippet is a saved body with optional aliases (KQ-017).
type Snippet struct {
	ID           int64    `json:"id"`
	SourceItemID *int64   `json:"sourceItemId,omitempty"`
	Title        string   `json:"title"`
	Body         string   `json:"body"`
	Language     string   `json:"language,omitempty"`
	CreatedAt    int64    `json:"createdAt"`
	UpdatedAt    int64    `json:"updatedAt"`
	Aliases      []string `json:"aliases,omitempty"`
}

// Stack is a saved ordered paste sequence (KQ-018).
type Stack struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"createdAt"`
	StepCount int    `json:"stepCount"`
}

// StackStepRow is one persisted stack step.
type StackStepRow struct {
	Position   int    `json:"position"`
	ItemID     *int64 `json:"itemId,omitempty"`
	InlineText string `json:"inlineText"`
	Label      string `json:"label"`
}

// CaptureStatus reports monitor state to the UI.
type CaptureStatus struct {
	Enabled        bool   `json:"enabled"`
	Paused         bool   `json:"paused"`
	PauseUntil     int64  `json:"pauseUntil,omitempty"` // 0 = not paused; -1 = until resume
	RemainingMs    int64  `json:"remainingMs,omitempty"`
	BackgroundWatch bool  `json:"backgroundWatch"`
	Message        string `json:"message,omitempty"`
}

// Generation is an opaque clipboard generation token from the OS.
type Generation int64

// RawPayload is every representation read for one generation.
type RawPayload struct {
	Generation Generation
	Text       string
	HTML       string
	RTF        string
	ImageBytes []byte
	ImageType  string // png, tiff, jpeg, bmp
	FileURLs   []string
	OSTypes    []string
}

// CanonicalItem is the normalized payload before persistence.
type CanonicalItem struct {
	Kind            string
	Title           string
	TextContent     string
	ImageBytes      []byte
	MimeType        string
	Width           int
	Height          int
	ImageKept       bool
	ByteSize        int64
	ContentHash     string
	BlobPath        string
	ThumbnailPath   string
	MetadataJSON    string
	DetectedType    string
	Language        string
	Sensitivity     string // none | suspected
	SensitiveReason string
	ExpiresAt       *int64
	TextPreview     string
}

// Snapshot is a frontmost-app context capture.
type Snapshot struct {
	ApplicationName string
	ApplicationID   string
	IconCacheKey    string
	ProcessID       int
	WindowTitle     string
	DocumentTitle   string
	URL             string
	Domain          string
	FileName        string
	FilePath        string
	Deep            bool
	CapturedAt      int64
}

// PlaceKey derives the grouping key for a snapshot.
func (s Snapshot) PlaceKey() string {
	if s.Domain != "" {
		return s.Domain
	}
	if s.FilePath != "" {
		return s.FilePath
	}
	if s.WindowTitle != "" {
		return s.WindowTitle
	}
	return ""
}

// ToCopyContext converts a snapshot into a storeable context (id unset).
func (s Snapshot) ToCopyContext() CopyContext {
	return CopyContext{
		ApplicationName: s.ApplicationName,
		ApplicationID:   s.ApplicationID,
		IconCacheKey:    s.IconCacheKey,
		ProcessID:       s.ProcessID,
		WindowTitle:     s.WindowTitle,
		DocumentTitle:   s.DocumentTitle,
		URL:             s.URL,
		Domain:          s.Domain,
		FileName:        s.FileName,
		FilePath:        s.FilePath,
		PlaceKey:        s.PlaceKey(),
		Deep:            s.Deep,
		CapturedAt:      s.CapturedAt,
	}
}
