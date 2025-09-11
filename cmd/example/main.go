package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ranakdinesh/spur/renderer"
	"github.com/ranakdinesh/spur/spur"
)

type App struct {
	Eng renderer.Engine
}

func New(rn renderer.Engine) *App {
	return &App{Eng: rn}
}

func (a *App) Home(w http.ResponseWriter, r *http.Request) {
	td := renderer.TemplateData{
		Theme: "default",
		Kind:  renderer.KindPage,
		Title: "About Us",
		StringMap: map[string]string{
			"Hero":    "<h2>Modern Go + WP-Style rendering</h2>",
			"content": "<p>This is the Home Page</p>",
		},
	}

	if err := a.Eng.RenderHTTP(r.Context(), w, r, &td); err != nil {
		if te, ok := err.(*template.Error); ok {
			log.Printf("TEMPLATE EXEC: name=%s line=%d err=%s", te.Name, te.Line, te.Error())
		} else {
			log.Printf("RenderHTTP: %v", err)
		}
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}
}
func main() {

	ctx := context.Background()
	s, err := spur.New(ctx, nil)
	if err != nil {
		panic(err)
	}
	ren := renderer.LoadConfigFromEnv()
	ren.Logger = log.New(os.Stderr, "renderer", log.LstdFlags|log.Lshortfile)

	eng, err := renderer.New(ren)
	if err != nil {
		log.Fatalf("could not create renderer: %v", err)
	}

	ap := New(eng)
	// HTTP route group (parent app)
	if s.HTTP != nil {
		s.HTTP.MountGroup("/api", func(r chi.Router) {
			r.Get("/", ap.Home)
		})
	}

	// start everything
	if err := s.Run(context.Background()); err != nil {
		s.Log.Logger.Error().Err(err).Msg("spur stopped with error")
	}

	// give some time to flush logs, exporters, etc.
	time.Sleep(200 * time.Millisecond)
}
