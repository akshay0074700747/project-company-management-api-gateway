package injectdependency

import (
	authcontrollers "github.com/akshay0074700747/projectandCompany_management_api-gateway/controllers/authControllers"
	companycontrollers "github.com/akshay0074700747/projectandCompany_management_api-gateway/controllers/companyControllers"
	projectcontrollers "github.com/akshay0074700747/projectandCompany_management_api-gateway/controllers/projectControllers"
	usrcontrollers "github.com/akshay0074700747/projectandCompany_management_api-gateway/controllers/usrControllers"
	"github.com/akshay0074700747/projectandCompany_management_api-gateway/helpers"
	"github.com/go-chi/chi"
)

func Inject(r *chi.Mux) {
	//
	userConn, err := helpers.DialGrpc(":50001")
	helpers.PrintErr(err, "cannot connect to user-service")
	projectConn, err := helpers.DialGrpc(":50002")
	helpers.PrintErr(err, "cannot connect to project-service")
	companyConn, err := helpers.DialGrpc(":50003")
	helpers.PrintErr(err, "cannot connect to company-service")
	authConn, err := helpers.DialGrpc(":50004")
	helpers.PrintErr(err, "cannot connect to auth-service")
	//
	userCtl := usrcontrollers.NewUserserviceClient(userConn)
	projectCtl := projectcontrollers.NewProjectCtl(projectConn)
	companyCtl := companycontrollers.NewCompanyCtl(companyConn)
	authCtl := authcontrollers.NewAuthCtl(authConn)
	//
	userCtl.InjectUserControllers(r)
	projectCtl.InjectProjectControllers(r)
	companyCtl.InjectCompanyControllers(r)
	authCtl.InjectAuthControllers(r)

}
