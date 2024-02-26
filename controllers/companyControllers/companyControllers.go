package companycontrollers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/akshay0074700747/projectandCompany_management_api-gateway/helpers"
	jwtvalidation "github.com/akshay0074700747/projectandCompany_management_api-gateway/jwtValidation"
	"github.com/akshay0074700747/projectandCompany_management_protofiles/pb/companypb"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (comp *CompanyCtl) registerCompany(w http.ResponseWriter, r *http.Request) {

	companyID := r.Context().Value("companyID").(string)
	if companyID != "" {
		http.Error(w, "you are already logged into a company", http.StatusInternalServerError)
		return
	}

	usrID := r.Context().Value("userID").(string)

	var req companypb.RegisterCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "cannot decode json on registerCompany")
		http.Error(w, "error on parsing", http.StatusInternalServerError)
		return
	}

	req.OwnerID = usrID

	res, err := comp.Conn.RegisterCompany(r.Context(), &req)
	if err != nil {
		helpers.PrintErr(err, "error on registering Company")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ress,err := comp.Conn.LogintoCompany(context.TODO(),&companypb.LogintoCompanyReq{
		CompanyUsername: res.Companyusername,
		UserID: usrID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot invoke LogintoCompany")
		return
	}

	cookieString, err := jwtvalidation.GenerateJwtforCompany(ress.CompanyID,ress.Role,ress.Permission, []byte(comp.Secret))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot create jwt")
		return
	}

	cookie := &http.Cookie{
		Name:     "companyCookie",
		Value:    cookieString,
		Expires:  time.Now().Add(48 * time.Hour),
		Path:     "/",
		HttpOnly: true,
	}

	http.SetCookie(w, cookie)

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

func (comp *CompanyCtl) getCompanyTypes(w http.ResponseWriter, r *http.Request) {

	stream, err := comp.Conn.GetCompanyTypes(context.TODO(), &emptypb.Empty{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot getCompanyTypes")
		return
	}

	var res []*companypb.GetCompanyTypesRes
	for {
		compType, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			helpers.PrintErr(err, "cannot recieve stream")
			return
		}
		res = append(res, compType)
	}

	jsondta, err := json.Marshal(res)
	if err != nil {
		http.Error(w, "there is a problem with parsing to json", http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot parse to json on getCompanyTypes")
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsondta)
}

func (comp *CompanyCtl) getPermissions(w http.ResponseWriter, r *http.Request) {

	stream, err := comp.Conn.GetPermissions(context.TODO(), &emptypb.Empty{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot getPermissions")
		return
	}

	var res []*companypb.Permission
	for {
		compType, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			helpers.PrintErr(err, "cannot recieve stream")
			return
		}
		res = append(res, compType)
	}

	jsondta, err := json.Marshal(res)
	if err != nil {
		http.Error(w, "there is a problem with parsing to json", http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot parse to json on getPermissions")
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsondta)
}

func (comp *CompanyCtl) addEmployees(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	companyID := r.Context().Value("companyID").(string)

	var req companypb.AddEmployeeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "cannot decode json on addEmployees")
		http.Error(w, "error on parsing", http.StatusInternalServerError)
		return
	}

	req.CompanyID = companyID

	_, err := comp.Conn.AddEmployees(context.TODO(), &req)
	if err != nil {
		helpers.PrintErr(err, "error on addEmployees")
		http.Error(w, "error on addEmployees", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write([]byte("employee request send successfully..."))
}

func (comp *CompanyCtl) attachRolewithPermissions(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	companyID := r.Context().Value("companyID").(string)

	var req companypb.AttachRoleWithPermisssionsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "cannot decode json on addEmployees")
		http.Error(w, "error on parsing", http.StatusInternalServerError)
		return
	}

	req.CompanyID = companyID

	_, err := comp.Conn.AttachRoleWithPermisssions(context.TODO(), &req)
	if err != nil {
		helpers.PrintErr(err, "error on attachRolewithPermissions")
		http.Error(w, "error on attachRolewithPermissions", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write([]byte("attached role with the given permission successfully..."))
}

func (comp *CompanyCtl) getattachedRoleswithPermissions(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	companyID := r.Context().Value("companyID").(string)

	var req companypb.GetAttachedRoleswithPermissionsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "cannot decode json on getattachedRoleswithPermissions")
		http.Error(w, "error on parsing", http.StatusInternalServerError)
		return
	}

	req.CompanyID = companyID

	stream, err := comp.Conn.GetAttachedRoleswithPermissions(context.TODO(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot getattachedRoleswithPermissions")
		return
	}

	var res []*companypb.GetAttachedRoleswithPermissionsRes
	for {
		v, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			helpers.PrintErr(err, "cannot recieve stream")
			return
		}
		res = append(res, v)
	}

	jsondta, err := json.Marshal(res)
	if err != nil {
		http.Error(w, "there is a problem with parsing to json", http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot parse to json on getattachedRoleswithPermissions")
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsondta)
}

func (comp *CompanyCtl) addCompanyTypes(w http.ResponseWriter, r *http.Request) {

	var req companypb.AddCompanyTypeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot parse the AddCompanyTypeReq req")
		return
	}

	_, err := comp.Conn.AddCompanyTypes(context.TODO(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "error at addCompanyTypes")
		return
	}

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write([]byte("new Company Type added successfully..."))
}

func (comp *CompanyCtl) addPermissions(w http.ResponseWriter, r *http.Request) {

	var req companypb.AddPermissionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot parse the AddCompanyTypeReq req")
		return
	}

	_, err := comp.Conn.Permissions(context.TODO(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "error at addPermissions")
		return
	}

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write([]byte("new permission added successfully..."))
}

func (comp *CompanyCtl) companyDetails(w http.ResponseWriter, r *http.Request) {

	companyID := r.Context().Value("companyID").(string)

	var req companypb.GetCompanyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot parse the GetCompanyReq req")
		return
	}

	req.CompanyID = companyID

	res, err := comp.Conn.GetCompanyDetails(context.TODO(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "error at GetCompanyDetails")
		return
	}

	jsondta, err := json.Marshal(res)
	if err != nil {
		http.Error(w, "there is a problem with parsing to json", http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot parse to json on companyDetails")
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsondta)
}

func (comp *CompanyCtl) getCompanyEmployees(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	companyID := r.Context().Value("companyID").(string)

	var req companypb.GetCompanyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot parse the GetCompanyReq req")
		return
	}

	req.CompanyID = companyID

	stream, err := comp.Conn.GetCompanyEmployees(context.TODO(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot getCompanyEmployees")
		return
	}

	var res []*companypb.GetCompanyEmployeesRes
	for {
		v, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			helpers.PrintErr(err, "cannot recieve stream")
			return
		}
		res = append(res, v)
	}

	jsondta, err := json.Marshal(res)
	if err != nil {
		http.Error(w, "there is a problem with parsing to json", http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot parse to json on getCompanyEmployees")
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsondta)
}

func (comp *CompanyCtl) LogintoCompany(w http.ResponseWriter, r *http.Request) {
	
	var loginReq companypb.LogintoCompanyReq
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot parse the LogintoCompanyReq req")
		return
	}

	loginReq.UserID = r.Context().Value("userID").(string)

	res,err := comp.Conn.LogintoCompany(context.TODO(),&loginReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot invoke LogintoCompany")
		return
	}

	cookieString, err := jwtvalidation.GenerateJwtforCompany(res.CompanyID,res.Role,res.Permission, []byte(comp.Secret))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot create jwt")
		return
	}

	cookie := &http.Cookie{
		Name:     "companyCookie",
		Value:    cookieString,
		Expires:  time.Now().Add(48 * time.Hour),
		Path:     "/",
		HttpOnly: true,
	}

	http.SetCookie(w, cookie)

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write([]byte("logged into company"))
}

func (comp *CompanyCtl) addCompanyMemberStatuses(w http.ResponseWriter, r *http.Request) {
	
}

func (comp *CompanyCtl) salaryIncrementofEmployees(w http.ResponseWriter, r *http.Request) {
	
}

func (comp CompanyCtl) salaryIncrementofRole(w http.ResponseWriter, r *http.Request) {
	
}

func (comp *CompanyCtl) getAverageSalaryPerRole(w http.ResponseWriter, r *http.Request) {
	
}

func (comp *CompanyCtl) getSalaryLeaderboard(w http.ResponseWriter, r *http.Request) {
	
}

func (comp *CompanyCtl) raiseProblem(w http.ResponseWriter, r *http.Request) {
	
}

func (comp *CompanyCtl) getProbelmsinaCompany(w http.ResponseWriter, r *http.Request) {
	
}

func (comp *CompanyCtl) getProfileViewsinaCompany(w http.ResponseWriter, r *http.Request) {
	
}

func (comp *CompanyCtl) getOnlineCompanyVisitors(w http.ResponseWriter, r *http.Request) {
	
}