package usrcontrollers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/akshay0074700747/projectandCompany_management_api-gateway/helpers"
	jwtvalidation "github.com/akshay0074700747/projectandCompany_management_api-gateway/jwtValidation"
	"github.com/akshay0074700747/projectandCompany_management_protofiles/pb/userpb"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Orders struct {
	OrderID        string `json:"OrderID"`
	UserID         string `json:"UserID"`
	AssetID        string `json:"AssetID"`
	SubscriptionID string `json:"SubscriptionID"`
	IsPayed        bool   `json:"IsPayed"`
}

type PaymentDetails struct {
	PaymentID  string    `json:"PaymentID"`
	OrderID    string    `json:"OrderID"`
	PaymentRef string    `json:"PaymentRef"`
	UpdatedAt  time.Time `json:"UpdatedAt" `
}

type Responce struct {
	Message    string `json:"Message"`
	StatusCode int    `json:"StatusCode"`
	Status     string `json:"Status"`
}

type SendMail struct {
	Email   string `json:"Email"`
	Message string `json:"Message"`
}

func (usr *UserCtl) signupUser(w http.ResponseWriter, r *http.Request) {
	if cookie, _ := r.Cookie("authentication"); cookie != nil {
		http.Error(w, "you are already logged in...", http.StatusConflict)
		return
	}

	var req userpb.SignupUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "cannot parse the signup req")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Otp == "" {

		otp := helpers.SelectRandomintBetweenRange(100000, 999999)

		if err := usr.Cache.CacheData(req.Email, []byte(strconv.Itoa(otp)), time.Minute*1, context.TODO()); err != nil {
			helpers.PrintErr(err, "error happened at saving otp")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		message := "Subject: OTP Verification\r\n" +
			"\r\n" +
			"Here is your otp,\r\n" + strconv.Itoa(otp) +
			"\r\n"

		value, err := json.Marshal(SendMail{
			Email:   req.Email,
			Message: message,
		})

		if err != nil {
			helpers.PrintErr(err, "error at marshaling")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = usr.Producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &usr.Topic, Partition: 0},
			Value:          value,
		}, usr.DeliveryChan)

		e := <-usr.DeliveryChan
		m := e.(*kafka.Message)
		if m.TopicPartition.Error != nil {
			helpers.PrintErr(err, "error at delivery Chan")
		} else {
			helpers.PrintMsg("message delivered")
		}

		var res Responce
		res.Message = "Otp Sent Successfully"
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

		fmt.Println(otp)

		return
	} else {
		var res []byte
		if err := usr.Cache.GetDataFromCache(req.Email, &res, context.TODO()); err != nil {
			helpers.PrintErr(err, "there is a problem withb getting data from cache")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		fmt.Println(len(res), "csdkvhcsj")

		var otp int
		if err := json.Unmarshal(res, &otp); err != nil {
			helpers.PrintErr(err, "there is a problem withb decoding")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		fmt.Println(otp, "------", req.Otp)

		if strconv.Itoa(otp) != req.Otp {
			http.Error(w, "the entered otp is not correct", http.StatusBadRequest)
			return
		}
	}

	fmt.Println(req.Otp, "kshbvkhf")
	fmt.Println(req.Password, "---javmhdsbkj")

	res, err := usr.Conn.SignupUser(r.Context(), &req)
	if err != nil {
		helpers.PrintErr(err, "there is a problem withb signupUser")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsondta, err := json.Marshal(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot parse to json on signup")
		return
	}

	cookieString, err := jwtvalidation.GenerateJwt(res.Id, false, []byte(usr.Secret))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot create jwt")
		return
	}

	cookie := &http.Cookie{
		Name:     "authentication",
		Value:    cookieString,
		Expires:  time.Now().Add(48 * time.Hour),
		Path:     "/",
		HttpOnly: true,
	}

	http.SetCookie(w, cookie)

	if companyCookie, _ := r.Cookie("companyCookie"); companyCookie != nil {
		companyCookie = &http.Cookie{
			Name:     "companyCookie",
			Value:    "",
			Expires:  time.Unix(0, 0),
			Path:     "/",
			HttpOnly: true,
		}

		http.SetCookie(w, companyCookie)
	}

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

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsondta)

}

func (usr *UserCtl) getRoles(w http.ResponseWriter, r *http.Request) {

	stream, err := usr.Conn.GetRoles(context.TODO(), &emptypb.Empty{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot getroles")
		return
	}

	var res []*userpb.Role
	for {
		role, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			helpers.PrintErr(err, "cannot recieve stream")
			return
		}
		res = append(res, role)
	}

	jsondta, err := json.Marshal(res)
	if err != nil {
		http.Error(w, "there is a problem with parsing to json", http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot parse to json on getroles")
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsondta)
}

func (usr *UserCtl) setStatus(w http.ResponseWriter, r *http.Request) {

	userID := r.Context().Value("userID").(string)

	var req userpb.StatusReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "cannot parse the statusreq req")
	}

	req.UserID = userID

	_, err := usr.Conn.SetStatus(context.TODO(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "error at SetStatus")
		return
	}

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write([]byte("status updated successfully..."))
}

func (usr *UserCtl) searchAvlMembers(w http.ResponseWriter, r *http.Request) {

	queries := r.URL.Query()
	roleID, err := strconv.Atoi(queries.Get("roleID"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		helpers.PrintErr(err, "error at SearchforMembers")
		return
	}

	var req userpb.SearchReq
	req.RoleID = uint32(roleID)

	stream, err := usr.Conn.SearchforMembers(context.TODO(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "error at SearchforMembers")
		return
	}

	var res []*userpb.SearchRes

	for {
		search, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			helpers.PrintErr(err, "cannot recieve stream")
			return
		}
		res = append(res, search)
	}

	jsondta, err := json.Marshal(res)
	if err != nil {
		http.Error(w, "there is a problem with parsing to json", http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot parse to json on searchAvlMembers")
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsondta)
}

func (usr *UserCtl) addRoles(w http.ResponseWriter, r *http.Request) {

	var req userpb.AddRoleReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "cannot parse the AddRoleReq req")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err := usr.Conn.AddRoles(context.TODO(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "error at addRoles")
		return
	}

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write([]byte("new Role added successfully..."))
}

func (usr *UserCtl) showUserDetails(w http.ResponseWriter, r *http.Request) {

	userID := r.Context().Value("userID").(string)

	res, err := usr.Conn.GetUserDetails(context.TODO(), &userpb.GetUserDetailsReq{
		UserID: userID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "error at showUserDetails")
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

func (usr *UserCtl) editStatus(w http.ResponseWriter, r *http.Request) {

	var req userpb.EditStatusReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "cannot parse the AddRoleReq req")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.UserID = r.Context().Value("userID").(string)

	_, err := usr.Conn.EditStatus(context.TODO(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "error at EditStatus")
		return
	}

	var res Responce
	res.Message = "Edited status Successfully"
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

func (usr *UserCtl) updateDetails(w http.ResponseWriter, r *http.Request) {

	var req userpb.UpdateUserDetailsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "cannot parse the EditStatusReq req")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.UserID = r.Context().Value("userID").(string)

	_, err := usr.Conn.UpdateUserDetails(context.TODO(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "error at UpdateUserDetails")
		return
	}

	var res Responce
	res.Message = "Edited user details Successfully"
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

func (usr *UserCtl) getSubscriptionPlans(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "http://localhost:50007/subscription/plan", http.StatusFound)
}

func (usr *UserCtl) addSubscription(w http.ResponseWriter, r *http.Request) {
	// http.Redirect(w, r, "http://localhost:50007/subscription/plan/add", http.StatusFound)
	client := &http.Client{}
	req, err := http.NewRequest("POST", "http://localhost:50007/subscription/plan/add", r.Body)
	if err != nil {
		helpers.PrintErr(err, "eroror happenend at proxying the request")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.Header = r.Header

	_, err = client.Do(req)
	if err != nil {
		helpers.PrintErr(err, "eroror happenend at proxying the request")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "Added subscription Plan Successfully"
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

func (usr *UserCtl) subscribe(w http.ResponseWriter, r *http.Request) {
	// http.Redirect(w, r, "http://localhost:50007/subscription/plan/subscribe", http.StatusFound)
	client := &http.Client{}

	userID := r.Context().Value("userID").(string)

	req, err := http.NewRequest("POST", "http://localhost:50007/subscription/plan/subscribe", r.Body)
	if err != nil {
		helpers.PrintErr(err, "eroror happenend at proxying the request")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.Header = r.Header
	req.Header.Set("X-User-ID", userID)

	res, err := client.Do(req)
	if err != nil {
		helpers.PrintErr(err, "eroror happenend at proxying the request")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var orders Orders
	if err := json.NewDecoder(res.Body).Decode(&orders); err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonDta, err := json.Marshal(orders)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}

func (usr *UserCtl) getSubscriptions(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "http://localhost:50007/subscriptions", http.StatusFound)
}

func (usr *UserCtl) pay(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "http://localhost:50007/subscription/plan/subscribe/order/pay", http.StatusFound)
	// client := &http.Client{}
	// req, err := http.NewRequest("POST", "http://localhost:50007/subscription/plan/subscribe/order/pay", r.Body)
	// if err != nil {
	// 	helpers.PrintErr(err, "eroror happenend at proxying the request")
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }

	// req.Header = r.Header

	// _, err = client.Do(req)
	// if err != nil {
	// 	helpers.PrintErr(err, "eroror happenend at proxying the request")
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }

	// var res Responce
	// res.Message = "Added subscription Plan Successfully"
	// res.Status = "Success"
	// res.StatusCode = http.StatusOK

	// jsonDta, err := json.Marshal(res)
	// if err != nil {
	// 	helpers.PrintErr(err, "error happenedat marshaling to json")
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }

	// w.WriteHeader(http.StatusOK)

	// w.Header().Set("Content-Type", "application/json")

	// w.Write(jsonDta)
}

func (usr *UserCtl) verifyPayment(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "http://localhost:50007/verify/payment", http.StatusFound)
}

func (usr *UserCtl) verifiedPayment(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "http://localhost:50007/payment/verified", http.StatusFound)
}

func (usr *UserCtl) getAllPayments(w http.ResponseWriter, r *http.Request) {
	// http.Redirect(w, r, "http://localhost:50007/payments", http.StatusFound)
	client := &http.Client{}

	userID := r.Context().Value("userID").(string)

	req, err := http.NewRequest("GET", "http://localhost:50007/payments", r.Body)
	if err != nil {
		helpers.PrintErr(err, "eroror happenend at proxying the request")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.Header = r.Header
	req.Header.Set("X-User-ID", userID)

	res, err := client.Do(req)
	if err != nil {
		helpers.PrintErr(err, "eroror happenend at proxying the request")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var payments []PaymentDetails
	if err := json.NewDecoder(res.Body).Decode(&payments); err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonDta, err := json.Marshal(payments)
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonDta)
}
