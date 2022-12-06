package spur

type initPaths struct {
	RootPath    string
	FolderNames []string
	// ConfigPath string
}
type cookieConfig struct {
	name     string
	lifetime string
	persist  string
	secure   string
	domain   string
}
