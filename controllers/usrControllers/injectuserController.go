package usrcontrollers

import (
	"github.com/akshay0074700747/projectandCompany_management_protofiles/pb/userpb"
	"github.com/go-chi/chi"
	"google.golang.org/grpc"
)

type UserCtl struct{
	Conn userpb.UserServiceClient
}

func NewUserserviceClient(conn *grpc.ClientConn) *UserCtl  {
	
	return &UserCtl{
		Conn: userpb.NewUserServiceClient(conn),
	}
}

func (usr *UserCtl) InjectUserControllers(r *chi.Mux) {
	r.Post("/signup",usr.signupUser)
}
