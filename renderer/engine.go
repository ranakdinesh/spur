package renderer

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"html/template"
	"net/http"
)

type Engine interface {
	Render(ctx context.Context, td *TemplateData) ([]byte, string, error)
	Invalidate(ctx context.Context, key string) error
	Close(ctx context.Context) error
	RenderHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request, td *TemplateData, opts ...HTTPOption) error
}

type engine struct {
	cfg    Config
	tcache *templateSetCache
	fcache *fullPageCache
	csrf   *csrfProvider
}

func New(cfg Config) (Engine, error) {
	cfg = cfg.withDefaults()
	tc := newTemplateSetCache(cfg)

	var fc *fullPageCache
	if cfg.Cache.Mode == CacheFullAnon || cfg.Cache.Mode == CacheFull {
		fc = newFullPageCache(cfg.Cache.MaxEntries, cfg.Cache.TTL)
	}

	cs := newCSRFProvider(cfg.Security.CSRFSecret)

	return &engine{cfg: cfg, tcache: tc, fcache: fc, csrf: cs}, nil
}

func (e *engine) Close(ctx context.Context) error { return nil }

func (e *engine) Invalidate(ctx context.Context, key string) error {
	e.tcache.Invalidate(key)
	if e.fcache != nil {
		e.fcache.InvalidatePrefix(key)
	}
	return nil
}

func (e *engine) Render(ctx context.Context, td *TemplateData) ([]byte, string, error) {
	if td == nil {
		td = &TemplateData{}
	}
	if td.CSRFToken == "" {
		td.CSRFToken = e.csrf.Token()
	}
	theme := td.Theme
	if theme == "" {
		theme = e.cfg.DefaultTheme
	}

	setEntry, err := e.tcache.GetOrBuild(theme)
	if err != nil {
		return nil, "", err
	}

	mainName := resolveMainTemplateName(td, setEntry.available)
	if mainName == "" {
		if setEntry.available["pages/index.gohtml"] {
			mainName = "pages/index.gohtml"
		} else {
			return []byte{}, "", nil
		}
	}

	useFull := e.cfg.Cache.Mode == CacheFull || (e.cfg.Cache.Mode == CacheFullAnon && canFullCache(td))
	var cacheKey string
	if useFull && e.fcache != nil {
		cacheKey = makeFullCacheKey(theme, mainName, setEntry.hash, td)
		if entry, ok := e.fcache.Get(cacheKey); ok {
			return entry.html, entry.etag, nil
		}
	}

	var buf bytes.Buffer
	if err := setEntry.tmpl.ExecuteTemplate(&buf, mainName, td); err != nil {
		if te, ok := err.(*template.Error); ok && e.cfg.Logger != nil {
			e.cfg.Logger.Printf("TEMPLATE EXEC: name =%s  line=%d main=%s err=%s", te.Name, te.Line, mainName, te.Error())
		}

		return nil, "", err
	}
	out := buf.Bytes()

	h := sha1.Sum(append(out, []byte(setEntry.hash)...))
	etag := hex.EncodeToString(h[:])

	if useFull && e.fcache != nil {
		e.fcache.Put(cacheKey, out, etag)
	}

	return out, etag, nil
}
