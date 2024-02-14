package projectcontrollers

import (
	"encoding/json"
	"net/http"

	"github.com/akshay0074700747/projectandCompany_management_api-gateway/helpers"
	"github.com/akshay0074700747/projectandCompany_management_protofiles/pb/projectpb"
)

func (proj *ProjectCtl) createProject(w http.ResponseWriter, r *http.Request) {

	var req projectpb.CreateProjectReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "cannot decode json on createproject")
		http.Error(w, "error on parsing", http.StatusInternalServerError)
		return
	}

	res, err := proj.Conn.CreateProject(r.Context(), &req)
	if err != nil {
		helpers.PrintErr(err, "error on creating project")
		http.Error(w, "error on creating project", http.StatusInternalServerError)
		return
	}

	jsondta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error on marshaling to json on createproject")
		http.Error(w, "error on marshling to json", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsondta)
}
