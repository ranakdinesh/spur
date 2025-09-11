package renderer

import (
	"crypto/sha1"
	"encoding/hex"
	"sort"
)

// canFullCache decides if a final HTML can be shared-cached (full_anon).
// Keep this conservative: avoid caching anything that looks user/session-specific.
func canFullCache(td *TemplateData) bool {
	if td == nil {
		return true
	}
	if td.IsAuthenticated {
		return false
	}
	if td.Error != "" || td.Flash != "" {
		return false
	}
	// Pages containing forms typically embed per-request CSRF tokens;
	// treat them as non-cacheable at the full-page layer.
	if len(td.Forms) > 0 {
		return false
	}
	return true
}

// makeFullCacheKey builds a stable cache key from theme, the main template
// name, the parsed-set content hash, and a hash of "public" (non-sensitive)
// bits of TemplateData.
func makeFullCacheKey(theme, main, thash string, td *TemplateData) string {
	return theme + "|" + main + "|" + thash + "|" + publicDataHash(td)
}

// publicDataHash hashes only public/stable fields so the cache key changes
// when visible content changes, but NOT for per-user/session fields.
func publicDataHash(td *TemplateData) string {
	if td == nil {
		return ""
	}
	h := sha1.New()
	add := func(s string) {
		if s != "" {
			_, _ = h.Write([]byte(s))
		}
	}

	// SEO / visible metadata
	add(td.Title)
	add(td.Description)
	add(td.CanonicalURL)
	add(td.OGImage)

	if len(td.Keywords) > 0 {
		kws := append([]string(nil), td.Keywords...)
		sort.Strings(kws)
		for _, k := range kws {
			add(k)
		}
	}

	if len(td.MetaTags) > 0 {
		keys := make([]string, 0, len(td.MetaTags))
		for k := range td.MetaTags {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			add(k)
			add(td.MetaTags[k])
		}
	}

	// Page content blocks
	if len(td.Blocks) > 0 {
		keys := make([]string, 0, len(td.Blocks))
		for k := range td.Blocks {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			add(k)
			add(td.Blocks[k])
		}
	}

	// Optional lightweight models (titles/slugs only)
	if td.Page != nil {
		add(td.Page.Title)
		add(td.Page.Slug)
	}
	if td.Post != nil {
		add(td.Post.Title)
		add(td.Post.Slug)
	}

	// IMPORTANT: we intentionally DO NOT include:
	// - CSRFToken, Flash, Error
	// - IsAuthenticated
	// - arbitrary Data maps that may contain user/session state
	// so full_anon caching remains safe.

	return hex.EncodeToString(h.Sum(nil))
}
