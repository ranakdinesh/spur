package renderer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type templateSet struct {
	tmpl      *template.Template
	hash      string
	available map[string]bool
}

type templateSetCache struct {
	cfg     Config
	byTheme map[string]*templateSet
}

func newTemplateSetCache(cfg Config) *templateSetCache {
	return &templateSetCache{cfg: cfg, byTheme: map[string]*templateSet{}}
}

func (c *templateSetCache) Invalidate(theme string) {
	if theme == "" {
		c.byTheme = map[string]*templateSet{}
		return
	}
	delete(c.byTheme, theme)
}

func (c *templateSetCache) GetOrBuild(theme string) (*templateSet, error) {
	if ts, ok := c.byTheme[theme]; ok {
		return ts, nil
	}
	ts, err := c.build(theme)
	if err != nil {
		return nil, err
	}
	c.byTheme[theme] = ts
	return ts, nil
}

func (c *templateSetCache) build(theme string) (*templateSet, error) {
	root := c.cfg.TemplateRoot
	sysRoot := filepath.Join(root, "system")
	thmRoot := filepath.Join(root, "themes", theme)

	winners := map[string]string{}

	mergeCategory := func(cat string) {
		sysDir := filepath.Join(sysRoot, cat)
		filepath.WalkDir(sysDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d == nil || d.IsDir() {
				return nil
			}
			if filepath.Ext(path) != ".gohtml" {
				return nil
			}
			rel, err := filepath.Rel(sysDir, path)
			if err != nil {
				return nil
			}
			canon := filepath.ToSlash(filepath.Join(cat, rel))
			winners[canon] = path
			return nil
		})
		thmDir := filepath.Join(thmRoot, cat)
		filepath.WalkDir(thmDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d == nil || d.IsDir() {
				return nil
			}
			if filepath.Ext(path) != ".gohtml" {
				return nil
			}
			rel, err := filepath.Rel(thmDir, path)
			if err != nil {
				return nil
			}
			canon := filepath.ToSlash(filepath.Join(cat, rel))
			winners[canon] = path
			return nil
		})
	}
	for _, cat := range []string{"layouts", "pages", "components", "functions"} {
		mergeCategory(cat)
	}

	available := map[string]bool{}
	for k := range winners {
		available[filepath.ToSlash(k)] = true
	}

	if len(winners) == 0 {
		return &templateSet{tmpl: template.New("empty").Funcs(baseFuncMap()).Option("missingkey=error"), hash: "", available: available}, nil
	}

	h := sha256.New()
	set := template.New("root").Funcs(baseFuncMap()).Option("missingkey=error")

	keys := make([]string, 0, len(winners))
	for k := range winners {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, canon := range keys {
		full := winners[canon]
		b, err := os.ReadFile(full)
		if err != nil {
			return nil, fmt.Errorf("read template %s: %w", canon, err)
		}
		_, _ = h.Write([]byte(canon))
		_, _ = h.Write(b)
		if _, err := set.New(canon).Parse(string(b)); err != nil {
			if te, ok := err.(*template.Error); ok && c.cfg.Logger != nil {
				c.cfg.Logger.Printf("Template parse error in %s: %s", full, te.Error())

			}
			return nil, fmt.Errorf("parse template %s: %w", canon, annotateTemplateError(err, canon))
		}
	}

	return &templateSet{tmpl: set, hash: hex.EncodeToString(h.Sum(nil)), available: available}, nil
}

func annotateTemplateError(err error, canon string) error {
	msg := err.Error()
	if strings.Contains(msg, "no such template") {
		return fmt.Errorf("%s,(hint: include by canonical names like layouts/base.gohtml or components/header.gohtml)", msg)
	}
	return err
}
