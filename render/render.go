package render

import (
	"fmt"
	"github.com/CloudyKit/jet/v6"
	"html/template"
	"net/http"
	"strings"
)

type Render struct {
	Renderer   string
	RootPath   string
	Secure     bool
	Port       string
	ServerName string
	JetViews   *jet.Set
}
type TemplateData struct {
	IsAuthenticated bool
	IntMap          map[string]int
	StringMap       map[string]string
	FloatMap        map[string]float32
	Data            map[string]any
	CSRFToken       string
	Port            string
	ServerName      string
	Secure          bool
}

func (s *Render) Page(w http.ResponseWriter, r *http.Request, page string, variables, data any) error {
	switch strings.ToLower(s.Renderer) {
	case "go":
		return s.GoPage(w, r, page, variables, data)

	case "html":
	default:

	}

	return nil
}

// GoPage Render the Go Page from the template
func (s *Render) GoPage(w http.ResponseWriter, r *http.Request, page string, variables, data any) error {
	tmpl, err := template.ParseFiles(fmt.Sprintf("%s/views/%s.gohtml", s.RootPath, page))
	if err != nil {
		return err
	}

	td := &TemplateData{}
	if data != nil {
		td = data.(*TemplateData)
	}
	tmpl.Execute(w, &td)
	return nil
}

// JetPage Render the Jet Page from the template
func (s *Render) JetPage(w http.ResponseWriter, r *http.Request, page string, variables, data any) error {
	var vars jet.VarMap
	if variables == nil {
		vars = make(jet.VarMap)
	} else {
		vars = variables.(jet.VarMap)
	}
	td := &TemplateData{}
	if data != nil {
		td = data.(*TemplateData)
	}
	t, err := s.JetViews.GetTemplate(fmt.Sprintf("%s.jet", page))
	if err != nil {
		return err
	}
	if err := t.Execute(w, vars, td); err != nil {
		return err
	}
	return nil
}
