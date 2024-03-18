package clicontroller

import "github.com/go-chi/chi"

func (cli *CliCtl) InjectCliControllers(r *chi.Mux) {
	r.Get("/downloads", cli.getDownloads)
	r.Get("/download/cli", cli.downloadCli)
}
