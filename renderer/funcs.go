
package renderer

import (
	"bytes"
	"encoding/json"
	"html/template"
	"strings"
)

func baseFuncMap() template.FuncMap {
	return template.FuncMap{
		"upper":     strings.ToUpper,
		"lower":     strings.ToLower,
		"title":     strings.Title,
		"join":      strings.Join,
		"truncate":  truncate,
		"safeHTML":  func(s string) template.HTML { return template.HTML(s) },
		"json":      toJSON,
	}
}

func truncate(s string, n int) string {
	if n <= 0 || len(s) <= n { return s }
	if n <= 3 { return s[:n] }
	return s[:n-3] + "..."
}

func toJSON(v any) template.JS {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(true)
	_ = enc.Encode(v)
	out := strings.TrimSuffix(buf.String(), "\n")
	return template.JS(out)
}
