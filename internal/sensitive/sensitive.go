package sensitive

import (
	"encoding/base64"
	"encoding/json"
	"regexp"
	"strings"

	"kopiq/internal/domain"
)

// Result of a sensitive check.
type Result struct {
	Suspected bool
	Reason    string
}

var (
	pemKey       = regexp.MustCompile(`PRIVATE KEY`)
	awsKey       = regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)
	githubPAT    = regexp.MustCompile(`\b(github_pat_[A-Za-z0-9_]{20,}|ghp_[A-Za-z0-9]{20,})\b`)
	slackTok     = regexp.MustCompile(`\bxox[bp]-[A-Za-z0-9-]{10,}\b`)
	openaiKey    = regexp.MustCompile(`\bsk-[A-Za-z0-9]{20,}\b`)
	connURI      = regexp.MustCompile(`(?i)\b(postgres|postgresql|mysql|mongodb|redis|amqp)://[^\s]+`)
)

// Check inspects canonical text for likely secrets.
func Check(item *domain.CanonicalItem) Result {
	if item.Kind != "text" || item.TextContent == "" {
		return Result{}
	}
	s := item.TextContent

	if pemKey.MatchString(s) {
		return Result{true, "pem"}
	}
	if awsKey.MatchString(s) {
		return Result{true, "aws_key"}
	}
	if githubPAT.MatchString(s) {
		return Result{true, "private_key_header"}
	}
	if slackTok.MatchString(s) {
		return Result{true, "private_key_header"}
	}
	if openaiKey.MatchString(s) {
		return Result{true, "private_key_header"}
	}
	if m := connURI.FindString(s); m != "" && hasPasswordUserinfo(m) {
		return Result{true, "connection_uri"}
	}
	if item.DetectedType == "jwt" || looksJWTWithExp(s) {
		return Result{true, "jwt_like"}
	}
	return Result{}
}

func hasPasswordUserinfo(uri string) bool {
	// scheme://user:pass@host
	i := strings.Index(uri, "://")
	if i < 0 {
		return false
	}
	rest := uri[i+3:]
	at := strings.Index(rest, "@")
	if at < 0 {
		return false
	}
	userinfo := rest[:at]
	return strings.Contains(userinfo, ":") && !strings.HasSuffix(userinfo, ":")
}

func looksJWTWithExp(s string) bool {
	parts := strings.Split(strings.TrimSpace(s), ".")
	if len(parts) != 3 || parts[2] == "" {
		return false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		payload, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return false
		}
	}
	var m map[string]any
	if err := json.Unmarshal(payload, &m); err != nil {
		return false
	}
	_, ok := m["exp"]
	return ok
}

// Apply marks the item sensitive according to policy ("expire" | "never").
// Returns false if the item must not be saved.
func Apply(item *domain.CanonicalItem, policy string, ttlSeconds int, nowMs int64) bool {
	res := Check(item)
	if !res.Suspected {
		return true
	}
	item.SensitiveReason = res.Reason
	if policy == "never" {
		return false
	}
	item.Sensitivity = "suspected"
	exp := nowMs + int64(ttlSeconds)*1000
	item.ExpiresAt = &exp
	meta := map[string]any{"sensitive_reason": res.Reason}
	b, _ := json.Marshal(meta)
	item.MetadataJSON = mergeMeta(item.MetadataJSON, b)
	return true
}

func mergeMeta(existing string, extra []byte) string {
	var a, b map[string]any
	_ = json.Unmarshal([]byte(existing), &a)
	_ = json.Unmarshal(extra, &b)
	if a == nil {
		a = map[string]any{}
	}
	for k, v := range b {
		a[k] = v
	}
	out, err := json.Marshal(a)
	if err != nil {
		return existing
	}
	return string(out)
}
