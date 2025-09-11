
package renderer

func resolveMainTemplateName(td *TemplateData, available map[string]bool) string {
	if td != nil && td.PageName != "" {
		n := "pages/" + td.PageName + ".gohtml"
		if available[n] { return n }
	}

	kind := KindPage
	if td != nil && td.Kind != "" { kind = td.Kind }

	switch kind {
	case KindHome:
		if available["pages/home.gohtml"] { return "pages/home.gohtml" }
		if available["pages/index.gohtml"] { return "pages/index.gohtml" }
	case KindBlogHome:
		if available["pages/bloghome.gohtml"] { return "pages/bloghome.gohtml" }
		if available["pages/index.gohtml"] { return "pages/index.gohtml" }
	case KindSingle:
		if available["pages/single.gohtml"] { return "pages/single.gohtml" }
		if available["pages/index.gohtml"] { return "pages/index.gohtml" }
	case KindPage, KindCustom:
		if available["pages/page.gohtml"] { return "pages/page.gohtml" }
		if available["pages/index.gohtml"] { return "pages/index.gohtml" }
	default:
		if available["pages/index.gohtml"] { return "pages/index.gohtml" }
	}
	return ""
}
