package companycontrollers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/akshay0074700747/projectandCompany_management_api-gateway/helpers"
	jwtvalidation "github.com/akshay0074700747/projectandCompany_management_api-gateway/jwtValidation"
	"github.com/akshay0074700747/projectandCompany_management_protofiles/pb/companypb"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/go-redis/redis/v8"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Responce struct {
	Message    string `json:"Message"`
	StatusCode int    `json:"StatusCode"`
	Status     string `json:"Status"`
}

func (comp *CompanyCtl) CompanyMiddleware(next http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		companyID := r.Context().Value("companyID").(string)
		userID := r.Context().Value("userID").(string)

		fmt.Println(companyID, "  ---hsdavmchakdlkhkdsj")
		fmt.Println(userID, "  ---usr")

		var res []byte
		if err := comp.Cache.GetDataFromCache(companyID+"_"+userID, &res, context.TODO()); err != nil {
			if err == redis.Nil {
				check, err := comp.Conn.GetUserStat(context.TODO(), &companypb.GetUserStatReq{
					CompanyID: companyID,
					UserID:    userID,
				})
				if err != nil {
					helpers.PrintErr(err, "cannot GetUserStat")
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				fmt.Println(check.IsAcceptable, "---aceppt")

				resss, err := json.Marshal(check.IsAcceptable)
				if err != nil {
					helpers.PrintErr(err, "cannot encode")
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if err = comp.Cache.CacheData(companyID+"_"+userID, resss, time.Hour*48, context.TODO()); err != nil {
					helpers.PrintErr(err, "cannot cache")
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if !check.IsAcceptable {
					http.Error(w, "you have no authority to access this route", http.StatusInternalServerError)
					return
				}

			} else {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		} else {
			var ress bool
			if err := json.Unmarshal(res, &ress); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			if !ress {
				http.Error(w, "you are no longer a member of the company", http.StatusBadRequest)
				return
			}
		}
		next(w, r)
	}
}

func (comp *CompanyCtl) registerCompany(w http.ResponseWriter, r *http.Request) {

	var companyID string
	if r.Context().Value("companyBool") != nil {
		companyID = r.Context().Value("companyID").(string)
	}
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
		if err.Error() == "rpc error: code = Unknown desc = the user is not payed" {
			var res Responce
			res.Message = "You must Subscribe to Create a Company"
			res.Status = "Failed"
			res.StatusCode = http.StatusBadRequest

			jsonDta, err := json.Marshal(res)
			if err != nil {
				helpers.PrintErr(err, "error happenedat marshaling to json")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusBadRequest)

			w.Header().Set("Content-Type", "application/json")

			w.Write(jsonDta)

			return
		}
		helpers.PrintErr(err, "error on registering Company")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ress, err := comp.Conn.LogintoCompany(context.TODO(), &companypb.LogintoCompanyReq{
		CompanyUsername: res.Companyusername,
		UserID:          usrID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot invoke LogintoCompany")
		return
	}

	cookieString, err := jwtvalidation.GenerateJwtforCompany(ress.CompanyID, ress.Role, ress.Permission, []byte(comp.Secret))
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "Added Employee Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusCreated

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
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

	var res Responce
	res.Message = "Binded Role with Permission Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusCreated

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) getattachedRoleswithPermissions(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	companyID := r.Context().Value("companyID").(string)

	stream, err := comp.Conn.GetAttachedRoleswithPermissions(context.TODO(), &companypb.GetAttachedRoleswithPermissionsReq{
		CompanyID: companyID,
	})
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

	var res Responce
	res.Message = "Added Company Type Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusCreated

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
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

	var res Responce
	res.Message = "Added Permission Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusCreated

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) companyDetails(w http.ResponseWriter, r *http.Request) {

	companyID := r.Context().Value("companyID").(string)

	res, err := comp.Conn.GetCompanyDetails(context.TODO(), &companypb.GetCompanyReq{
		CompanyID: companyID,
	})
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

	stream, err := comp.Conn.GetCompanyEmployees(context.TODO(), &companypb.GetCompanyReq{
		CompanyID: companyID,
	})
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

	res, err := comp.Conn.LogintoCompany(context.TODO(), &loginReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot invoke LogintoCompany")
		return
	}

	fmt.Println(res)

	cookieString, err := jwtvalidation.GenerateJwtforCompany(res.CompanyID, res.Role, res.Permission, []byte(comp.Secret))
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

	var ress Responce
	ress.Message = "Logged into the Company Successfully"
	ress.Status = "Success"
	ress.StatusCode = http.StatusOK

	jsonDta, err := json.Marshal(ress)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) addCompanyMemberStatuses(w http.ResponseWriter, r *http.Request) {

	var req companypb.MemberStatusReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot decode from json")
		return
	}

	_, err := comp.Conn.AddMemberStatus(context.TODO(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot AddMemberStatus")
		return
	}

	var res Responce
	res.Message = "Added member status Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusCreated

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) salaryIncrementofEmployees(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	companyID := r.Context().Value("companyID").(string)

	var req companypb.SalaryIncrementofEmployeeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot decode json")
		return
	}

	req.CompanyID = companyID

	_, err := comp.Conn.SalaryIncrementofEmployee(context.TODO(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot SalaryIncrementofEmployee")
		return
	}

	var res Responce
	res.Message = "Incremented Salary of Employee Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusCreated

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) salaryIncrementofRole(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	companyID := r.Context().Value("companyID").(string)

	var req companypb.SalaryIncrementofRoleReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot decode json")
		return
	}

	req.CompanyID = companyID

	_, err := comp.Conn.SalaryIncrementofRole(context.TODO(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot SalaryIncrementofRole")
		return
	}

	var res Responce
	res.Message = "Incremented Salary of Role Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusCreated

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) getAverageSalaryPerRole(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	companyID := r.Context().Value("companyID").(string)

	stream, err := comp.Conn.GetAverageSalaryperRole(context.TODO(), &companypb.GetAverageSalaryperRoleReq{
		CompanyID: companyID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot GetAverageSalaryperRole")
		return
	}

	var res []*companypb.GetAverageSalaryperRoleRes
	for {
		data, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			helpers.PrintErr(err, "cannot recieve stream")
			return
		}
		res = append(res, data)
	}

	jsonDta, err := json.Marshal(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot marshl to json")
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) getSalaryLeaderboard(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	companyID := r.Context().Value("companyID").(string)

	stream, err := comp.Conn.GetSalaryLeaderboard(context.TODO(), &companypb.GetSalaryLeaderboardReq{
		CompanyID: companyID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot get stream")
		return
	}

	var res []*companypb.GetSalaryLeaderboardRes
	for {
		data, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			helpers.PrintErr(err, "cannot recieve fromstream")
			return
		}
		res = append(res, data)
	}

	jsonDta, err := json.Marshal(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot encode to json")
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) raiseProblem(w http.ResponseWriter, r *http.Request) {

	companyID := r.Context().Value("companyID").(string)
	userID := r.Context().Value("userID").(string)

	if companyID == "" || userID == "" {
		http.Error(w, "companyID and userID is empty", http.StatusInternalServerError)
		return
	}

	var req companypb.RaiseProblemReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot decode from json")
		return
	}

	req.CompanyID = companyID
	req.UserID = userID

	_, err := comp.Conn.RaiseProblem(context.TODO(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot RaiseProblem")
		return
	}

	var res Responce
	res.Message = "Raised Problem Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusCreated

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) getProbelmsinaCompany(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	companyID := r.Context().Value("companyID").(string)

	stream, err := comp.Conn.GetProblems(context.TODO(), &companypb.GetProblemsReq{
		CompanyID: companyID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot GetProblems")
		return
	}

	var res []*companypb.GetProblemsRes

	for {
		data, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			helpers.PrintErr(err, "cannot recieve from stream")
			return
		}
		res = append(res, data)
	}

	jsonDta, err := json.Marshal(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot marshal to json")
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) getProfileViewsinaCompany(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	companyID := r.Context().Value("companyID").(string)

	fromTime, err := time.Parse("2006-01-02 15:04:05", r.URL.Query().Get("From"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	toTime, err := time.Parse("2006-01-02 15:04:05", r.URL.Query().Get("To"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	res, err := comp.Conn.GetProfileViews(context.TODO(), &companypb.GetProfileViewsReq{
		CompanyID: companyID,
		From:      timestamppb.New(fromTime),
		To:        timestamppb.New(toTime),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot GetProfileViews")
		return
	}

	jsonDta, err := json.Marshal(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot marshel to json")
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) getOnlineCompanyVisitors(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	companyID := r.Context().Value("companyID").(string)

	fromTime, err := time.Parse("2006-01-02 15:04:05", r.URL.Query().Get("From"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	toTime, err := time.Parse("2006-01-02 15:04:05", r.URL.Query().Get("To"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	stream, err := comp.Conn.GetVisitors(context.TODO(), &companypb.GetVisitorsReq{
		CompanyID: companyID,
		From:      timestamppb.New(fromTime),
		To:        timestamppb.New(toTime),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot GetVisitors")
		return
	}

	var res []*companypb.GetVisitorsRes

	for {
		data, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			helpers.PrintErr(err, "cannot recieve from stream")
			return
		}
		res = append(res, data)
	}

	jsonDta, err := json.Marshal(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot marshel to json")
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) logoutFromCompany(w http.ResponseWriter, r *http.Request) {

	if cookie, _ := r.Cookie("companyCookie"); cookie == nil {
		http.Error(w, "allready not logged into a company", http.StatusBadRequest)
		return
	}

	cookie := &http.Cookie{
		Name:     "companyCookie",
		Value:    "",
		Expires:  time.Unix(0, 0),
		Path:     "/",
		HttpOnly: true,
	}

	http.SetCookie(w, cookie)

	if projectCookie, _ := r.Cookie("projectCookie"); projectCookie != nil {
		projectCookie = &http.Cookie{
			Name:     "projectCookie",
			Value:    "",
			Expires:  time.Unix(0, 0),
			Path:     "/",
			HttpOnly: true,
		}

		http.SetCookie(w, projectCookie)
	}

	var res Responce
	res.Message = "Logged out of the Company Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusOK

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

type Address struct {
	StreetName string `json:"StreetName"`
	StreetNo   int32  `json:"StreetNo"`
	PinNo      int32  `json:"PinNo"`
	District   string `json:"District"`
	State      string `json:"State"`
	Nation     string `json:"Nation"`
}

type JobApplications struct {
	ApplicationID      string  `json:"ApplicationID"`
	Name               string  `json:"Name"`
	Email              string  `json:"Email"`
	Mobile             string  `json:"Mobile"`
	AddressofApplicant Address `json:"AddressofApplicant"`
	HighestEducation   string  `json:"HighestEducation"`
	Nationality        string  `json:"Nationality"`
	Experiance         uint32  `json:"Experiance"`
	CurrentCTC         float32 `json:"CurrentCTC"`
	Resume             []byte  `json:"Resume"`
	FileName           string  `json:"FileName"`
	JobID              string  `json:"JobID"`
}

func (comp *CompanyCtl) applyforJob(w http.ResponseWriter, r *http.Request) {

	var res JobApplications

	err := r.ParseMultipartForm(r.ContentLength)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "there is a problem with parsing the form")
		return
	}

	exp, err := strconv.Atoi(r.MultipartForm.Value["Experiance"][0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "there is a problem with parsing")
		return
	}

	ctc, err := strconv.Atoi(r.MultipartForm.Value["CurrentCTC"][0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "there is a problem with parsing")
		return
	}

	pin, err := strconv.Atoi(r.MultipartForm.Value["PinNo"][0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "there is a problem with parsing")
		return
	}

	streetNo, err := strconv.Atoi(r.MultipartForm.Value["StreetNo"][0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "there is a problem with parsing")
		return
	}

	res.Name = r.MultipartForm.Value["Name"][0]
	res.Email = r.MultipartForm.Value["Email"][0]
	res.Nationality = r.MultipartForm.Value["Nationality"][0]
	res.Mobile = r.MultipartForm.Value["Mobile"][0]
	res.HighestEducation = r.MultipartForm.Value["HighestEducation"][0]
	res.Experiance = uint32(exp)
	res.CurrentCTC = float32(ctc)
	res.AddressofApplicant.District = r.MultipartForm.Value["District"][0]
	res.AddressofApplicant.Nation = r.MultipartForm.Value["Nation"][0]
	res.AddressofApplicant.PinNo = int32(pin)
	res.AddressofApplicant.State = r.MultipartForm.Value["State"][0]
	res.AddressofApplicant.StreetName = r.MultipartForm.Value["Street"][0]
	res.AddressofApplicant.StreetNo = int32(streetNo)
	res.JobID = r.MultipartForm.Value["JobID"][0]

	file := r.MultipartForm.File["File"][0]

	txtFile, err := file.Open()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "there is a problem with parsing file")
		return
	}

	res.FileName = file.Filename

	buffer := new(bytes.Buffer)
	_, err = buffer.ReadFrom(txtFile)
	if err != nil {
		helpers.PrintErr(err, "error at reading from txtfile")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	res.Resume = buffer.Bytes()

	taskBytes, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error encoding to json")
		return
	}

	err = comp.JobApplier.Producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &comp.JobApplier.Topic, Partition: 0},
		Value:          taskBytes,
	}, comp.JobApplier.DeliveryChan)

	e := <-comp.JobApplier.DeliveryChan
	m := e.(*kafka.Message)
	if m.TopicPartition.Error != nil {
		helpers.PrintErr(err, "error at delivery Chan")
	} else {
		helpers.PrintMsg("message delivered")
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write([]byte("Applied fro job Successfully..."))
}

func (comp *CompanyCtl) addCompanyClients(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	var req companypb.AddClientReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot decode from json")
		return
	}

	req.CompanyID = r.Context().Value("companyID").(string)

	if _, err := comp.Conn.AddClient(context.TODO(), &req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot AddClient")
		return
	}

	var res Responce
	res.Message = "Added Client to the Company Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusCreated

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) associateClientwithProject(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	var req companypb.AssociateClientWithProjectReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.CompanyID = r.Context().Value("companyID").(string)

	if _, err := comp.Conn.AssociateClientWithProject(context.TODO(), &req); err != nil {
		helpers.PrintErr(err, "error happened at AssociateClientWithProject")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "Added Client to the Company Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusCreated

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) getPastProjects(w http.ResponseWriter, r *http.Request) {

	stream, err := comp.Conn.GetPastProjects(context.TODO(), &companypb.GetProjectsReq{
		CompanyID: r.Context().Value("companyID").(string),
	})
	if err != nil {
		helpers.PrintErr(err, "error happenedat GetExtensionRequests")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res []*companypb.GetProjectsRes

	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			helpers.PrintErr(err, "error happened st recieving from stream")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		res = append(res, msg)
	}

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat parsing to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) getClients(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	stream, err := comp.Conn.GetClients(context.TODO(), &companypb.GetClientsReq{
		CompanyID: r.Context().Value("companyID").(string),
	})
	if err != nil {
		helpers.PrintErr(err, "error happenedat GetExtensionRequests")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res []*companypb.GetClientsRes

	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			helpers.PrintErr(err, "error happened st recieving from stream")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		res = append(res, msg)
	}

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat parsing to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) revenueGenerated(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	stream, err := comp.Conn.GetRevenueGenerated(context.TODO(), &companypb.GetRevenueGeneratedReq{
		CompanyID: r.Context().Value("companyID").(string),
	})
	if err != nil {
		helpers.PrintErr(err, "error happenedat GetExtensionRequests")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res []*companypb.GetRevenueGeneratedRes

	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			helpers.PrintErr(err, "error happened st recieving from stream")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		res = append(res, msg)
	}

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat parsing to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) updateRevenueStatus(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	var req companypb.UpdateRevenueStatusReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.CompanyID = r.Context().Value("companyID").(string)

	_, err := comp.Conn.UpdateRevenueStatus(context.TODO(), &req)
	if err != nil {
		helpers.PrintErr(err, "error happened at UpdateRevenueStatus")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "Updated Revenue Status Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusCreated

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) attachCompanyPolicies(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	var req companypb.AttachCompanyPoliciesReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.CompanyID = r.Context().Value("companyID").(string)

	_, err := comp.Conn.AttachCompanyPolicies(context.TODO(), &companypb.AttachCompanyPoliciesReq{})
	if err != nil {
		helpers.PrintErr(err, "error happened at UpdateRevenueStatus")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "Attached Company Policies Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusCreated

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)

}

func (comp *CompanyCtl) updatePaymentStatus(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	var req companypb.UpdatePaymentStatusofEmployeeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.CompanyID = r.Context().Value("companyID").(string)

	_, err := comp.Conn.UpdatePaymentStatusofEmployee(context.TODO(), &req)
	if err != nil {
		helpers.PrintErr(err, "error happened at UpdateRevenueStatus")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "Updated Payment Status Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusCreated

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)

}

func (comp *CompanyCtl) assignProblem(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	var req companypb.AssignProblemReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.CompanyID = r.Context().Value("companyID").(string)

	_, err := comp.Conn.AssignProblem(context.TODO(), &req)
	if err != nil {
		helpers.PrintErr(err, "error happened at UpdateRevenueStatus")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "Assigned Problem to Employee Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusCreated

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) resolveProblem(w http.ResponseWriter, r *http.Request) {

	var req companypb.ResolveProblemReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.CompanyID = r.Context().Value("companyID").(string)
	req.ResolverID = r.Context().Value("resolverID").(string)

	_, err := comp.Conn.ResolveProblem(context.TODO(), &req)
	if err != nil {
		helpers.PrintErr(err, "error happened at ResolveProblem")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "Resolved Problem Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusCreated

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) applyforLeave(w http.ResponseWriter, r *http.Request) {

	var req companypb.ApplyForLeaveReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.CompanyID = r.Context().Value("companyID").(string)
	req.EmployeeID = r.Context().Value("userID").(string)
	_, err := comp.Conn.ApplyForLeave(context.TODO(), &req)
	if err != nil {
		helpers.PrintErr(err, "error happened at ApplyForLeave")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "Applied for Leave Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusCreated

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) getLeaveRequests(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	stream, err := comp.Conn.GetEmployeeLeaveRequests(context.TODO(), &companypb.GetEmployeeLeaveRequestsReq{
		CompanyID: r.Context().Value("companyID").(string),
	})
	if err != nil {
		helpers.PrintErr(err, "error happenedat GetEmployeeLeaveRequests")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res []*companypb.GetEmployeeLeaveRequestsRes

	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			helpers.PrintErr(err, "error happened st recieving from stream")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		res = append(res, msg)
	}

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat parsing to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) decideEmployeeLeave(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	var req companypb.DecideEmployeeLeaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err := comp.Conn.DecideEmployeeLeave(context.TODO(), &req)
	if err != nil {
		helpers.PrintErr(err, "error happened at DecideEmployeeLeave")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "Successfull"
	res.Status = "Success"
	res.StatusCode = http.StatusCreated

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) getLeaves(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	stream, err := comp.Conn.GetLeaves(context.TODO(), &companypb.GetLeavesReq{
		CompanyID: r.Context().Value("companyID").(string),
		From:      r.URL.Query().Get("From"),
		To:        r.URL.Query().Get("To"),
	})
	if err != nil {
		helpers.PrintErr(err, "error happenedat GetLeaves")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res []*companypb.GetLeavesRes

	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			helpers.PrintErr(err, "error happened st recieving from stream")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		res = append(res, msg)
	}

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat parsing to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) postJobs(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	var req companypb.PostJobsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.CompanyID = r.Context().Value("companyID").(string)

	_, err := comp.Conn.PostJobs(context.TODO(), &req)
	if err != nil {
		helpers.PrintErr(err, "error happened at PostJobs")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "Posted Job Successfully Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusCreated

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) getJobsofCompany(w http.ResponseWriter, r *http.Request) {

	stream, err := comp.Conn.GetJobsofCompany(context.TODO(), &companypb.GetJobsofCompanyReq{
		CompanyID: r.Context().Value("companyID").(string),
	})
	if err != nil {
		helpers.PrintErr(err, "error happened at GetJobsofCompany")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res []*companypb.GetJobsofCompanyRes

	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			helpers.PrintErr(err, "error happened st recieving from stream")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		res = append(res, msg)
	}

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat parsing to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) getJobApplications(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	stream, err := comp.Conn.GetJobApplications(context.TODO(), &companypb.GetJobApplicationsReq{
		CompanyID: r.Context().Value("companyID").(string),
		JobID:     r.URL.Query().Get("jobID"),
	})
	if err != nil {
		helpers.PrintErr(err, "error happenedat GetJobApplications")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res []*companypb.GetJobApplicationsRes

	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			helpers.PrintErr(err, "error happened st recieving from stream")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		res = append(res, msg)
	}

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat parsing to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) shortlistApplications(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	var req companypb.ShortlistApplicationsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.CompanyID = r.Context().Value("companyID").(string)

	_, err := comp.Conn.ShortlistApplications(context.TODO(), &req)
	if err != nil {
		helpers.PrintErr(err, "error happened at ShortlistApplications")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "Shortlisted Application Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusCreated

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) scheduleInterview(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	var req companypb.ScheduleInterviewReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.CompanyID = r.Context().Value("companyID").(string)

	_, err := comp.Conn.ScheduleInterview(context.TODO(), &req)
	if err != nil {
		helpers.PrintErr(err, "error happened at ScheduleInterview")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "Scheduled Interview Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusCreated

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) sheduledInterviews(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	stream, err := comp.Conn.GetScheduledInterviews(context.TODO(), &companypb.GetScheduledInterviewsReq{
		CompanyID: r.Context().Value("companyID").(string),
	})
	if err != nil {
		helpers.PrintErr(err, "error happenedat GetScheduledInterviews")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res []*companypb.GetScheduledInterviewsRes

	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			helpers.PrintErr(err, "error happened st recieving from stream")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		res = append(res, msg)
	}

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat parsing to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) getDetailsofApplicationByID(w http.ResponseWriter, r *http.Request) {

	res, err := comp.Conn.GetDetailsofApplicationByID(context.TODO(), &companypb.GetDetailsofApplicationByIDReq{
		ApplicationID: r.URL.Query().Get("ApplicationID"),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "error at GetDetailsofApplicationByID")
		return
	}

	jsondta, err := json.Marshal(res)
	if err != nil {
		http.Error(w, "there is a problem with parsing to json", http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot parse to json on showUserDetails")
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsondta)
}

func (comp *CompanyCtl) getScheduledInterviewsofUser(w http.ResponseWriter, r *http.Request) {

	stream, err := comp.Conn.GetScheduledInterviewsofUser(context.TODO(), &companypb.GetScheduledInterviewsofUserReq{
		UserID: r.Context().Value("userID").(string),
	})
	if err != nil {
		helpers.PrintErr(err, "error happenedat GetScheduledInterviewsofUser")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res []*companypb.GetScheduledInterviewsofUserRes

	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			helpers.PrintErr(err, "error happened st recieving from stream")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		res = append(res, msg)
	}

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat parsing to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) rescheduleInterview(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	var req companypb.RescheduleInterviewReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err := comp.Conn.RescheduleInterview(context.TODO(), &req)
	if err != nil {
		helpers.PrintErr(err, "error happened at RescheduleInterview")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "Rescheduled Interview Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusCreated

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) getShortlistedApplications(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	stream, err := comp.Conn.GetShortlistedApplications(context.TODO(), &companypb.GetShortlistedApplicationsReq{
		CompanyID: r.Context().Value("companyID").(string),
		JobID:     r.URL.Query().Get("JobID"),
	})
	if err != nil {
		helpers.PrintErr(err, "error happenedat GetShortlistedApplications")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res []*companypb.GetShortlistedApplicationsRes

	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			helpers.PrintErr(err, "error happened st recieving from stream")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		res = append(res, msg)
	}

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat parsing to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) getJobs(w http.ResponseWriter, r *http.Request) {

	stream, err := comp.Conn.GetJobs(context.TODO(), &companypb.GetJobsReq{
		CompanyID: r.URL.Query().Get("CompanyID"),
		Role:      r.URL.Query().Get("Role"),
	})
	if err != nil {
		helpers.PrintErr(err, "error happenedat GetJobs")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res []*companypb.GetJobsRes

	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			helpers.PrintErr(err, "error happened st recieving from stream")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		res = append(res, msg)
	}

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat parsing to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) getAllapplicationsofUser(w http.ResponseWriter, r *http.Request) {

	stream, err := comp.Conn.GetAllJobApplicationsofUser(context.TODO(), &companypb.GetAllJobApplicationsofUserReq{
		UserID: r.Context().Value("userID").(string),
	})
	if err != nil {
		helpers.PrintErr(err, "error happened at GetAllJobApplicationsofUser")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res []*companypb.GetAllJobApplicationsofUserRes

	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			helpers.PrintErr(err, "error happened st recieving from stream")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		res = append(res, msg)
	}

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat parsing to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) getAssignedProblems(w http.ResponseWriter, r *http.Request) {

	stream, err := comp.Conn.GetAssignedProblems(context.TODO(), &companypb.GetAssignedProblemsReq{
		UserID:    r.Context().Value("userID").(string),
		CompanyID: r.Context().Value("companyID").(string),
	})
	if err != nil {
		helpers.PrintErr(err, "error happened at GetAssignedProblems")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res []*companypb.GetAssignedProblemsRes

	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			helpers.PrintErr(err, "error happened at recieving from stream")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		res = append(res, msg)
	}

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat parsing to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) dropCompany(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	var req companypb.DropCompanyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat parsing to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err := comp.Conn.DropCompany(context.TODO(), &req)
	if err != nil {
		helpers.PrintErr(err, "error happenedat DropCompany")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "Dropped Company Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusOK

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) editDetais(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	var req companypb.EditCompanyDetailsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat parsing to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.CompanyID = r.Context().Value("companyID").(string)

	_, err := comp.Conn.EditCompanyDetails(context.TODO(), &req)
	if err != nil {
		helpers.PrintErr(err, "error happenedat EditCompanyDetails")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "Edited Company Detials Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusOK

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) deleteCompanyEmployee(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	var req companypb.TerminateEmployeeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat parsing to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.CompanyID = r.Context().Value("companyID").(string)

	_, err := comp.Conn.TerminateEmployee(context.TODO(), &req)
	if err != nil {
		helpers.PrintErr(err, "error happenedat TerminateEmployee")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "Terminated Employee Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusOK

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) editCompanyEmployees(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	var req companypb.EditCompanyEmployeesReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat parsing to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.CompanyID = r.Context().Value("companyID").(string)

	_, err := comp.Conn.EditCompanyEmployees(context.TODO(), &req)
	if err != nil {
		helpers.PrintErr(err, "error happenedat EditCompanyEmployees")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "Edited Company employee Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusOK

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) deleteProblem(w http.ResponseWriter, r *http.Request) {

	var req companypb.DeleteProblemReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat parsing to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err := comp.Conn.DeleteProblem(context.TODO(), &req)
	if err != nil {
		helpers.PrintErr(err, "error happenedat DeleteProblem")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "deleted problem Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusOK

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) editproblems(w http.ResponseWriter, r *http.Request) {

	var req companypb.EditProblemReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat parsing to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err := comp.Conn.EditProblem(context.TODO(), &req)
	if err != nil {
		helpers.PrintErr(err, "error happenedat EditProblem")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "edited problem Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusOK

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) dropClient(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	var req companypb.DropClientFromCompanyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat parsing to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.CompanyID = r.Context().Value("companyID").(string)

	_, err := comp.Conn.DropClientFromCompany(context.TODO(), &req)
	if err != nil {
		helpers.PrintErr(err, "error happenedat DropClientFromCompany")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "dropped Company Client Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusOK

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) editCompanyPolicies(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	var req companypb.UpdateCompanyPoliciesReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat parsing to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.CompanyID = r.Context().Value("companyID").(string)

	_, err := comp.Conn.UpdateCompanyPolicies(context.TODO(), &req)
	if err != nil {
		helpers.PrintErr(err, "error happenedat UpdateCompanyPolicies")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "updated company policies Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusOK

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) deleteJob(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	var req companypb.DeleteJobReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat parsing to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err := comp.Conn.DeleteJob(context.TODO(), &req)
	if err != nil {
		helpers.PrintErr(err, "error happenedat DeleteJob")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "deleted job Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusOK

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (comp *CompanyCtl) updateJob(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
		return
	}

	var req companypb.UpdateJobReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat parsing to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err := comp.Conn.UpdateJob(context.TODO(), &req)
	if err != nil {
		helpers.PrintErr(err, "error happenedat UpdateJob")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "updated job Successfully"
	res.Status = "Success"
	res.StatusCode = http.StatusOK

	jsonDta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}
