package main

import (
	"net/http"

	injectdependency "github.com/akshay0074700747/projectandCompany_management_api-gateway/injectDependency"
	"github.com/go-chi/chi"
)

func main() {

	r := chi.NewRouter()

	injectdependency.Inject(r)

	http.ListenAndServe(":50000", r) 
}
