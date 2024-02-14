package companycontrollers

import (
	"encoding/json"
	"net/http"

	"github.com/akshay0074700747/projectandCompany_management_api-gateway/helpers"
	"github.com/akshay0074700747/projectandCompany_management_protofiles/pb/companypb"
)

func (comp *CompanyCtl) registerCompany(w http.ResponseWriter, r *http.Request) {

	var req companypb.RegisterCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "cannot decode json on registerCompany")
		http.Error(w, "error on parsing", http.StatusInternalServerError)
		return
	}

	res, err := comp.Conn.RegisterCompany(r.Context(), &req)
	if err != nil {
		helpers.PrintErr(err, "error on registering Company")
		http.Error(w, "error on registering Company", http.StatusInternalServerError)
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
