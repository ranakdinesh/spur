
package renderer

import "html/template"

type PageKind string
const (
	KindHome     PageKind = "home"
	KindPage     PageKind = "page"
	KindSingle   PageKind = "single"
	KindBlogHome PageKind = "bloghome"
	KindCustom   PageKind = "custom"
)

type TemplateData struct {
	Theme string
	Kind PageKind
	PageName string

	IsAuthenticated bool
	IntMap map[string]int
	StringMap map[string]string
	FloatMap map[string]float32
	Data map[string]any

	CSRFToken string
	Port string
	ServerName string
	Secure bool
	Error string
	Flash string

	Title string
	Description string
	CanonicalURL string
	OGImage string
	Keywords []string
	SchemaJSONLD template.JS
	MetaTags map[string]string

	CSPNonce string
	Blocks map[string]string

	Page *PageModel
	Post *PostModel
	List *ListModel

	Forms []FormInstance
}

type PageModel struct{ Title, Slug string }
type PostModel struct{ Title, Slug string }
type ListModel struct{ Items []any; Pagination any }

type FormInstance struct {
	Key, Action, Method string
	Fields []FormField
	Errors map[string]string
	Values map[string]string
	Hints  map[string]string
}

type FormField struct {
	Name, Type, Label, Placeholder, Pattern, Min, Max string
	Required bool
	Options []string
}
