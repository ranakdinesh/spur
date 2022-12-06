package spur

import "net/http"

func (s *Spur) SesssionLoad(next http.Handler) http.Handler {
	return s.Session.LoadAndSave(next)
}
