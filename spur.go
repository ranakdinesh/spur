package spur

import (
	"fmt"
	"github.com/CloudyKit/jet/v6"
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/ranakdinesh/spur/render"
	"github.com/ranakdinesh/spur/session"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

//const version = "1.0.0"

type Spur struct {
	AppName  string
	Debug    bool
	Version  string
	Routes   *chi.Mux
	Render   *render.Render
	JetViews *jet.Set
	Errorlog *log.Logger
	Infolog  *log.Logger
	Session  *scs.SessionManager
	RootPath string
	config   config
}
type config struct {
	port        string
	renderer    string
	cookie      cookieConfig
	SessionType string
}

func (s *Spur) New(rootPath string) error {
	pathConfig := initPaths{
		RootPath:    rootPath,
		FolderNames: []string{"adapter", "cmd", "config", "handlers", "migrations", "views", "public", "logs", "tmp", "Model", "", "utils", "views"},
	}
	err := s.Init(pathConfig)

	if err != nil {
		return err
	}

	err = s.checkDotEnv(rootPath)
	if err != nil {
		return err
	}
	//Read the .env file
	err = godotenv.Load(fmt.Sprintf("%s/.env", rootPath))
	if err != nil {
		return err
	}

	//Create loggers of the application
	infoLog, errorLog := s.CreateLoggers()
	s.Infolog = infoLog
	s.Errorlog = errorLog
	s.Debug, _ = strconv.ParseBool(os.Getenv("DEBUG"))
	s.Version = os.Getenv("VERSION")
	s.RootPath = rootPath
	s.config = config{
		port:     os.Getenv("PORT"),
		renderer: os.Getenv("RENDERER"),
		cookie: cookieConfig{
			name:     os.Getenv("COOKIE_NAME"),
			lifetime: os.Getenv("COOKIE_LIFETIME"),
			persist:  os.Getenv("COOKIE_PERSIST"),
			secure:   os.Getenv("COOKIE_SECURE"),
			domain:   os.Getenv("COOKIE_DOMAIN"),
		},
		SessionType: os.Getenv("SESSION_TYPE"),
	}
	s.Routes = s.routes().(*chi.Mux)
	s.Render = s.createRenderer()
	sess := session.Session{
		SessionType:    s.config.SessionType,
		CookieName:     s.config.cookie.name,
		CookieLifeTime: s.config.cookie.lifetime,
		CookiePersist:  s.config.cookie.persist,
		CookieSecure:   s.config.cookie.secure,
		CookieDomain:   s.config.cookie.domain,
	}
	s.Session = sess.InitSession()
	// Setting Jet Views
	var views = jet.NewSet(
		jet.NewOSFileSystemLoader(fmt.Sprintf("%s/views", rootPath)),
	)
	s.JetViews = views

	return nil
}

func (s *Spur) Init(p initPaths) error {
	root := p.RootPath
	//creating Folders if they do not exist
	for _, path := range p.FolderNames {
		err := s.CreateDirIfNotExist(root + "/" + path)
		if err != nil {
			return err
		}
	}
	return nil
}
func (s *Spur) checkDotEnv(rootPath string) error {

	err := s.CreateFileIfNotExists(fmt.Sprintf("%s/.env", rootPath))
	if err != nil {
		return err
	}

	return nil
}
func (s *Spur) CreateLoggers() (*log.Logger, *log.Logger) {
	var infoLog, errorLog *log.Logger
	infoLog = log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog = log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)
	return infoLog, errorLog
}

//ListenAndServe function ListenAndServe will start the server

func (s *Spur) ListenAndServe() {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", os.Getenv("PORT")),
		ErrorLog:     s.Errorlog,
		Handler:      s.routes(),
		IdleTimeout:  30 * time.Second,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 600 * time.Second,
	}
	s.Infolog.Printf("Starting server on port %s", os.Getenv("PORT"))
	err := srv.ListenAndServe()
	if err != nil {
		s.Errorlog.Fatal(err)
	}

}
func (s *Spur) createRenderer() *render.Render {
	renderer := render.Render{
		Renderer: s.config.renderer,
		RootPath: s.RootPath,
		Port:     s.config.port,
		JetViews: s.JetViews,
	}

	return &renderer
}
