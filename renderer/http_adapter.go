package renderer

import (
	"context"
	"net/http"
)

// in renderer (e.g., http_adapter.go)
type HTTPOption func(*httpOpts)

func WithStatus(code int) HTTPOption       { return func(o *httpOpts) { o.Status = code } }
func WithContentType(ct string) HTTPOption { return func(o *httpOpts) { o.ContentType = ct } }
func WithHeader(k, v string) HTTPOption {
	return func(o *httpOpts) {
		if o.Extra == nil {
			o.Extra = map[string][]string{}
		}
		o.Extra[k] = append(o.Extra[k], v)
	}
}
func WithoutETag() HTTPOption { return func(o *httpOpts) { o.DisableETag = true } }

type httpOpts struct {
	Status      int
	ContentType string
	Extra       map[string][]string
	DisableETag bool
}

// New convenience method for handlers:
func (e *engine) RenderHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request, td *TemplateData, opts ...HTTPOption) error {
	o := httpOpts{Status: http.StatusOK, ContentType: "text/html; charset=utf-8"}
	for _, fn := range opts {
		fn(&o)
	}

	html, etag, err := e.Render(ctx, td) // reuse the existing Render
	if err != nil {
		return err
	}

	// ETag / 304 handling
	if !o.DisableETag && etag != "" {
		if inm := r.Header.Get("If-None-Match"); inm != "" && inm == etag {
			w.Header().Set("ETag", etag)
			w.WriteHeader(http.StatusNotModified)
			return nil
		}
		w.Header().Set("ETag", etag)
	}

	if o.ContentType != "" {
		w.Header().Set("Content-Type", o.ContentType)
	}
	for k, vs := range o.Extra {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(o.Status)
	_, err = w.Write(html)
	return err
}
