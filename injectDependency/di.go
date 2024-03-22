package injectdependency

import (
	"os"

	authcontrollers "github.com/akshay0074700747/projectandCompany_management_api-gateway/controllers/authControllers"
	chatcontrollers "github.com/akshay0074700747/projectandCompany_management_api-gateway/controllers/chatControllers"
	clicontroller "github.com/akshay0074700747/projectandCompany_management_api-gateway/controllers/cli-controller"
	companycontrollers "github.com/akshay0074700747/projectandCompany_management_api-gateway/controllers/companyControllers"
	projectcontrollers "github.com/akshay0074700747/projectandCompany_management_api-gateway/controllers/projectControllers"
	snapshotcontrollers "github.com/akshay0074700747/projectandCompany_management_api-gateway/controllers/snapshotControllers"
	usrcontrollers "github.com/akshay0074700747/projectandCompany_management_api-gateway/controllers/usrControllers"
	"github.com/akshay0074700747/projectandCompany_management_api-gateway/helpers"
	"github.com/akshay0074700747/projectandCompany_management_api-gateway/rediss"
	"github.com/go-chi/chi"
	"github.com/joho/godotenv"
)

func Inject(r *chi.Mux) {

	if err := godotenv.Load(".env"); err != nil {
		helpers.PrintErr(err, "the secret cannot be retrieved...")
	}
	secret := os.Getenv("secret")

	//
	userConn, err := helpers.DialGrpc("user-service:50001")
	if err != nil {
		helpers.PrintErr(err, "cannot connect to user-service")
	}
	projectConn, err := helpers.DialGrpc("project-service:50002")
	if err != nil {
		helpers.PrintErr(err, "cannot connect to project-service")
	}
	companyConn, err := helpers.DialGrpc("company-service:50003")
	if err != nil {
		helpers.PrintErr(err, "cannot connect to company-service")
	}
	authConn, err := helpers.DialGrpc("auth-service:50004")
	if err != nil {
		helpers.PrintErr(err, "cannot connect to auth-service")
	}
	//
	cache := rediss.NewCache(rediss.NewRedis())
	//
	userCtl := usrcontrollers.NewUserserviceClient(userConn, secret, cache, "Emailsender")
	taskAssignator := projectcontrollers.NewTaskProducer("taskTopic")
	projectCtl := projectcontrollers.NewProjectCtl(projectConn, taskAssignator, secret, cache)
	jobAssignator := companycontrollers.NewJobProducer("jobTopic")
	companyCtl := companycontrollers.NewCompanyCtl(companyConn, secret, jobAssignator, cache)
	authCtl := authcontrollers.NewAuthCtl(authConn, secret)
	cliCtl := clicontroller.NewCliCtl(cache)
	//
	userCtl.InjectUserControllers(r)
	projectCtl.InjectProjectControllers(r)
	companyCtl.InjectCompanyControllers(r)
	authCtl.InjectAuthControllers(r)
	cliCtl.InjectCliControllers(r)
	snapShotCtl := snapshotcontrollers.NewSnapshotCtl("SnapshotTopic")
	snapShotCtl.InjectSnapshotControllers(r)
	chatcontrollers.InjectChatControllers(r)
}
