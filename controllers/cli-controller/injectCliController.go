package clicontroller

import "github.com/go-chi/chi"

func InjectCliControllers(r *chi.Mux) {
	r.Get("/downloads", getDownloads)
	r.Get("/download/cli", downloadCli)
}
