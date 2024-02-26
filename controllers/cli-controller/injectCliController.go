package clicontroller

import "github.com/go-chi/chi"

func InjectCliControllers(r *chi.Mux) {
	r.Post("/downloads/cli", downloadCli)
	r.Get("/downloads", getDownloads)
}
