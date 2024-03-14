package projectcontrollers

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
	"github.com/akshay0074700747/projectandCompany_management_protofiles/pb/projectpb"
	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type TaskDta struct {
	UserID      string    `json:"user_id"`
	ProjectID   string    `json:"project_id"`
	Task        string    `json:"task"`
	Description string    `json:"description"`
	File        []byte    `json:"file"`
	Stages      int       `json:"stages"`
	Deadline    time.Time `json:"deadline"`
}

type Responce struct {
	Message    string `json:"Message"`
	StatusCode int    `json:"StatusCode"`
	Status     string `json:"Status"`
}

func (proj *ProjectCtl) createProject(w http.ResponseWriter, r *http.Request) {

	var projectID string
	if r.Context().Value("projectBool") != nil {
		projectID = r.Context().Value("projectID").(string)
	}
	if projectID != "" {
		http.Error(w, "you are already logged into a project", http.StatusInternalServerError)
		return
	}

	var isCompanybased bool

	var companyID string
	if r.Context().Value("companyID") != nil {
		companyID = r.Context().Value("companyID").(string)
	}
	if companyID != "" {

		permission := r.Context().Value("companyPermission").(string)
		if permission != "ROOT" && permission != "SEMI-ROOT" {
			http.Error(w, "you dont have the neccessary permissions to do this operation", http.StatusInternalServerError)
			return
		}
		isCompanybased = true
	}

	usrID := r.Context().Value("userID").(string)

	var req projectpb.CreateProjectReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "cannot decode json on createproject")
		http.Error(w, "error on parsing", http.StatusInternalServerError)
		return
	}

	req.OwnerID = usrID

	if isCompanybased {
		req.IsCompanybased = true
		req.ComapanyId = companyID
	}

	res, err := proj.Conn.CreateProject(r.Context(), &req)
	if err != nil {
		helpers.PrintErr(err, "error on creating project")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ress, err := proj.Conn.LogintoProject(context.TODO(), &projectpb.LogintoProjectReq{
		ProjectUsername: res.ProjectUsername,
		MemberID:        usrID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot invoke LogintoProject")
		return
	}

	fmt.Println(ress)

	cookieString, err := jwtvalidation.GenerateJwtforProject(ress.ProjectID, ress.Role, ress.Permission, []byte(proj.Secret))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot create jwt")
		return
	}

	cookie := &http.Cookie{
		Name:     "projectCookie",
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

func (project *ProjectCtl) addMembers(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("projectPermission").(string)

	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the permission to do this operation", http.StatusConflict)
		return
	}

	projectID := r.Context().Value("projectID").(string)

	var req projectpb.AddMemberReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "cannot decode json on addMembers")
		http.Error(w, "error on parsing", http.StatusInternalServerError)
		return
	}

	req.ProjectID = projectID

	_, err := project.Conn.AddMembers(context.TODO(), &req)
	if err != nil {
		helpers.PrintErr(err, "error on addMembers")
		http.Error(w, "error on addMembers", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write([]byte("new member invite sent successfully..."))
}

func (project *ProjectCtl) projectInvites(w http.ResponseWriter, r *http.Request) {

	userID := r.Context().Value("userID").(string)

	var req projectpb.ProjectInvitesReq
	req.MemberID = userID

	stream, err := project.Conn.ProjectInvites(context.TODO(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot ProjectInvites")
		return
	}

	var res []*projectpb.ProjectInvitesRes
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
		helpers.PrintErr(err, "cannot parse to json on projectInvites")
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsondta)
}

func (project *ProjectCtl) acceptProjectInvite(w http.ResponseWriter, r *http.Request) {

	userID := r.Context().Value("userID").(string)

	var req projectpb.AcceptProjectInviteReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "cannot decode json on acceptProjectInvite")
		http.Error(w, "error on parsing", http.StatusInternalServerError)
		return
	}

	req.UserID = userID

	_, err := project.Conn.AcceptProjectInvite(context.TODO(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot acceptProjectInvite")
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write([]byte("accepted projectInvite successfully..."))
}

func (proj *ProjectCtl) projectDetails(w http.ResponseWriter, r *http.Request) {

	projectID := r.Context().Value("projectID").(string)

	res, err := proj.Conn.GetProjectDetailes(context.TODO(), &projectpb.GetProjectReq{
		ProjectID: projectID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "error at projectDetails")
		return
	}

	jsondta, err := json.Marshal(res)
	if err != nil {
		http.Error(w, "there is a problem with parsing to json", http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot parse to json on projectDetails")
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsondta)
}

func (project *ProjectCtl) getProjectMembers(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("projectPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the permission to do this operation", http.StatusConflict)
		return
	}

	projectID := r.Context().Value("projectID").(string)

	stream, err := project.Conn.GetProjectMembers(context.TODO(), &projectpb.GetProjectReq{
		ProjectID: projectID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot getProjectMembers")
		return
	}

	var res []*projectpb.GetProjectMembersRes
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
		helpers.PrintErr(err, "cannot parse to json on getProjectMembers")
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsondta)
}

func (project *ProjectCtl) assignTasks(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("projectPermission").(string)
	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the permission to do this operation", http.StatusConflict)
		return
	}

	var res TaskDta

	res.ProjectID = r.Context().Value("projectID").(string)
	queryParams := r.URL.Query()
	res.UserID = queryParams.Get("memberID")

	err := r.ParseMultipartForm(r.ContentLength)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "there is a problem with parsing the form")
		return
	}

	res.Task = r.MultipartForm.Value["Task"][0]
	res.Description = r.MultipartForm.Value["Description"][0]
	res.Stages, err = strconv.Atoi(r.MultipartForm.Value["Stages"][0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "there is a problem with converting from string to integer")
		return
	}

	res.Deadline, err = time.Parse("2006-01-02", r.MultipartForm.Value["Deadline"][0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "there is a problem with converting from string to datetime")
		return
	}

	file := r.MultipartForm.File["File"][0]

	txtFile, err := file.Open()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "there is a problem with parsing to txt file")
		return
	}

	buffer := new(bytes.Buffer)
	_, err = buffer.ReadFrom(txtFile)
	if err != nil {
		helpers.PrintErr(err, "error at reading from txtfile")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	res.File = buffer.Bytes()

	taskBytes, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error encoding to json")
		return
	}

	_, err = project.Conn.IsMemberAccepted(context.TODO(), &projectpb.IsMemberAcceptedReq{
		UserID:    res.UserID,
		ProjectID: res.ProjectID,
	})
	if err != nil {
		helpers.PrintErr(err, "error at reading from txtfile")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = project.TaskAssignator.Producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &project.TaskAssignator.Topic, Partition: 0},
		Value:          taskBytes,
	}, project.TaskAssignator.DeliveryChan)

	e := <-project.TaskAssignator.DeliveryChan
	m := e.(*kafka.Message)
	if m.TopicPartition.Error != nil {
		helpers.PrintErr(err, "error at delivery Chan")
	} else {
		helpers.PrintMsg("message delivered")
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write([]byte("Tasks assigned Successfully..."))
}

func (project *ProjectCtl) downloadTask(w http.ResponseWriter, r *http.Request) {

	// permission := r.Context().Value("projectPermission").(string)
	// if permission != "BASIC" && permission != "SEMI-ROOT" {
	// 	http.Error(w, "you dont have the permission to do this operation", http.StatusConflict)
	// 	return
	// }

	// projectID := r.Context().Value("projectID").(string)
	// userID := r.Context().Value("userID").(string)
	queryParams := r.URL.Query()
	projectID := queryParams.Get("projectID")
	userID := queryParams.Get("userID")
	taskID := queryParams.Get("taskID")

	res, err := project.Conn.DownloadTask(context.TODO(), &projectpb.DownloadTaskReq{
		UserID:     userID,
		ProjectID:  projectID,
		TaskFileID: taskID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot DownloadTask")
		return
	}

	reader := bytes.NewReader(res.File)

	w.Header().Set("Content-Disposition", "attachment; filename=task.txt")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", reader.Size()))

	if _, err = io.Copy(w, reader); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot copy from reader to wirter")
		return
	}
}

func (proj *ProjectCtl) LogintoProject(w http.ResponseWriter, r *http.Request) {

	var loginReq projectpb.LogintoProjectReq
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot parse the LogintoProjectReq req")
		return
	}

	loginReq.MemberID = r.Context().Value("userID").(string)

	res, err := proj.Conn.LogintoProject(context.TODO(), &loginReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot invoke LogintoCompany")
		return
	}

	cookieString, err := jwtvalidation.GenerateJwtforProject(res.ProjectID, res.Role, res.Permission, []byte(proj.Secret))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot create jwt")
		return
	}

	cookie := &http.Cookie{
		Name:     "projectCookie",
		Value:    cookieString,
		Expires:  time.Now().Add(48 * time.Hour),
		Path:     "/",
		HttpOnly: true,
	}

	http.SetCookie(w, cookie)

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write([]byte("logged into project"))
}

func (proj *ProjectCtl) addMemberStatus(w http.ResponseWriter, r *http.Request) {

	var req projectpb.MemberStatusReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot parse the MemberStatusReq req")
		return
	}

	_, err := proj.Conn.AddMemberStatus(context.Background(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot AddMemberStatus")
		return
	}

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write([]byte("project status added successfully"))
}

func (proj *ProjectCtl) getAssignedTasks(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("projectPermission").(string)

	fmt.Println(permission)

	if permission != "BASIC" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the permission to do this operation", http.StatusConflict)
		return
	}

	projectID := r.Context().Value("projectID").(string)
	userID := r.Context().Value("userID").(string)

	res, err := proj.Conn.GetAssignedTask(context.TODO(), &projectpb.GetAssignedTaskReq{
		UserID:    userID,
		ProjectID: projectID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot GetAssignedTask")
		return
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

func (proj *ProjectCtl) getProgressofMembers(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("projectPermission").(string)

	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the permission to do this operation", http.StatusConflict)
		return
	}

	projectID := r.Context().Value("projectID").(string)

	stream, err := proj.Conn.GetProgressofMembers(context.TODO(), &projectpb.GetProgressofMembersReq{
		ProjectID: projectID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot GetProgressofMembers")
		return
	}

	var res []*projectpb.GetProgressofMembersRes
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

func (proj *ProjectCtl) getMemberProgress(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("projectPermission").(string)

	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the permission to do this operation", http.StatusConflict)
		return
	}

	projectID := r.Context().Value("projectID").(string)
	memberID := r.URL.Query().Get("memberID")

	res, err := proj.Conn.GetProgressofMember(context.TODO(), &projectpb.GetProgressofMemberReq{
		ProjectID: projectID,
		MemberID:  memberID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot GetProgressofMember")
		return
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

func (proj *ProjectCtl) getProjectProgress(w http.ResponseWriter, r *http.Request) {

	projectID := r.Context().Value("projectID").(string)

	res, err := proj.Conn.GetProjectProgress(context.TODO(), &projectpb.GetProjectProgressReq{
		ProjectID: projectID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot GetProjectProgress")
		return
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

func (proj *ProjectCtl) markProgressofNonTechnical(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("projectPermission").(string)

	if permission != "BASIC" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the permission to do this operation", http.StatusConflict)
		return
	}

	projectID := r.Context().Value("projectID").(string)
	userID := r.Context().Value("userID").(string)

	var req projectpb.MarkProgressofNonTechnicalReq

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannotdecode from json")
		return
	}

	req.ProjectID = projectID
	req.MemberID = userID

	_, err := proj.Conn.MarkProgressofNonTechnical(context.TODO(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot MarkProgressofNonTechnical")
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write([]byte("progress have been marked successfully"))
}

func (proj *ProjectCtl) addTaskStatuses(w http.ResponseWriter, r *http.Request) {

	var req projectpb.AddTaskStatusesReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot decode from json")
		return
	}

	_, err := proj.Conn.AddTaskStatuses(context.TODO(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot AddTaskStatuses")
		return
	}

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write([]byte("task status have been added successfully"))
}

func (proj *ProjectCtl) getLiveProjectsofCompany(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("companyPermission").(string)

	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the permission to do this operation", http.StatusConflict)
		return
	}

	companyID := r.Context().Value("companyID").(string)

	stream, err := proj.Conn.GetLiveProjects(context.TODO(), &projectpb.GetLiveProjectsReq{
		CompanyID: companyID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot GetLiveProjects")
		return
	}

	var res []*projectpb.GetLiveProjectsRes
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

	jsonData, err := json.Marshal(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot marshal to json")
		return
	}

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonData)
}

func (proj *ProjectCtl) logoutFromProject(w http.ResponseWriter, r *http.Request) {

	if cookie, _ := r.Cookie("projectCookie"); cookie == nil {
		http.Error(w, "allready not logged into a project", http.StatusBadRequest)
		return
	}

	cookie := &http.Cookie{
		Name:     "projectCookie",
		Value:    "",
		Expires:  time.Unix(0, 0),
		Path:     "/",
		HttpOnly: true,
	}

	http.SetCookie(w, cookie)

	w.WriteHeader(http.StatusOK)

	w.Write([]byte("Logged out Successfully..."))
}

func (proj *ProjectCtl) getCompletedMembers(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("projectPermission").(string)

	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the permission to do this operation", http.StatusConflict)
		return
	}

	projectID := r.Context().Value("projectID").(string)

	stream, err := proj.Conn.GetCompletedMembers(context.TODO(), &projectpb.GetCompletedMembersReq{
		ProjectID: projectID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "errror happened at GetCompletedMembers")
		return
	}

	var res []*projectpb.GetCompletedMembersRes

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

func (proj *ProjectCtl) getCriticalMembers(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("projectPermission").(string)

	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the permission to do this operation", http.StatusConflict)
		return
	}

	projectID := r.Context().Value("projectID").(string)

	stream, err := proj.Conn.GetCriticalMembers(context.TODO(), &projectpb.GetCriticalMembersReq{
		ProjectID: projectID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "errror happened at GetCompletedMembers")
		return
	}

	var res []*projectpb.GetCriticalMembersRes

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

func (proj *ProjectCtl) raiseIssue(w http.ResponseWriter, r *http.Request) {

	var req projectpb.RaiseIssueReq

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat parsing from json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.ProjectID = r.Context().Value("projectID").(string)
	req.RaiserID = r.Context().Value("userID").(string)

	if _, err := proj.Conn.RaiseIssue(context.TODO(), &req); err != nil {
		helpers.PrintErr(err, "error happenedat raising the issue")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "Issue Raised Successfully"
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

func (proj *ProjectCtl) getIssuesofMember(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("projectPermission").(string)

	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the permission to do this operation", http.StatusConflict)
		return
	}

	res, err := proj.Conn.GetIssues(context.TODO(), &projectpb.GetIssuesReq{
		ProjectID: r.Context().Value("projectID").(string),
		MemberID:  r.URL.Query().Get("userID"),
	})
	if err != nil {
		helpers.PrintErr(err, "error happenedat GetIssues")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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

func (proj *ProjectCtl) getIssues(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("projectPermission").(string)

	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the permission to do this operation", http.StatusConflict)
		return
	}

	stream, err := proj.Conn.GetIssuesofProject(context.TODO(), &projectpb.GetIssuesofProjectReq{
		ProjectID: r.Context().Value("projectID").(string),
	})
	if err != nil {
		helpers.PrintErr(err, "error happenedat GetIssues")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res []*projectpb.GetIssuesofProjectRes

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

func (proj *ProjectCtl) rateTask(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("projectPermission").(string)

	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the permission to do this operation", http.StatusConflict)
		return
	}

	var req projectpb.RateTaskReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happened at decoding from json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.ProjectID = r.Context().Value("projectID").(string)

	if _, err := proj.Conn.RateTask(context.TODO(), &req); err != nil {
		helpers.PrintErr(err, "error happened at RateTask")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "Rated Task Successfully"
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

func (proj *ProjectCtl) feedbackforTask(w http.ResponseWriter, r *http.Request) {

	res, err := proj.Conn.GetfeedBackforTask(context.TODO(), &projectpb.GetfeedBackforTaskReq{
		ProjectID: r.Context().Value("projectID").(string),
		MemberID:  r.Context().Value("userID").(string),
	})
	if err != nil {
		helpers.PrintErr(err, "error happenedat marshaling to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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

func (proj *ProjectCtl) requestforDeadlineExtension(w http.ResponseWriter, r *http.Request) {

	var req projectpb.RequestforDeadlineExtensionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat marshaling from json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.MemberID = r.Context().Value("userID").(string)
	req.ProjectID = r.Context().Value("projectID").(string)

	if _, err := proj.Conn.RequestforDeadlineExtension(context.TODO(), &req); err != nil {
		helpers.PrintErr(err, "error happened at RequestforDeadlineExtension")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "Requested for Deadline Extension Successfully"
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

func (proj *ProjectCtl) getExtensionRequests(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("projectPermission").(string)

	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the permission to do this operation", http.StatusConflict)
		return
	}

	stream, err := proj.Conn.GetExtensionRequests(context.TODO(), &projectpb.GetExtensionRequestsReq{
		ProjectID: r.Context().Value("projectID").(string),
	})
	if err != nil {
		helpers.PrintErr(err, "error happenedat GetExtensionRequests")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res []*projectpb.GetExtensionRequestsRes

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

func (proj *ProjectCtl) grantExtension(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("projectPermission").(string)

	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the permission to do this operation", http.StatusConflict)
		return
	}

	var req projectpb.GrantExtensionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat marshaling from json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := proj.Conn.GrantExtension(context.TODO(), &req); err != nil {
		helpers.PrintErr(err, "error happened at GrantExtension")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "Granted Deadline Extension Successfully"
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

func (proj *ProjectCtl) verifyTaskCompletion(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("projectPermission").(string)

	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the permission to do this operation", http.StatusConflict)
		return
	}

	var req projectpb.VerifyTaskCompletionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat marshaling from json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.ProjectID = r.Context().Value("projectID").(string)

	if _, err := proj.Conn.VerifyTaskCompletion(context.TODO(), &req); err != nil {
		helpers.PrintErr(err, "error happened at VerifyTaskCompletion")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "Verified Task Completion Successfully"
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

func (proj *ProjectCtl) getverifiedTasks(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("projectPermission").(string)

	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the permission to do this operation", http.StatusConflict)
		return
	}

	stream, err := proj.Conn.GetVerifiedTasks(context.TODO(), &projectpb.GetVerifiedTasksReq{
		ProjectID: r.Context().Value("projectID").(string),
	})
	if err != nil {
		helpers.PrintErr(err, "error happenedat GetVerifiedTasks")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res []*projectpb.GetVerifiedTasksRes

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

func (proj *ProjectCtl) dropProject(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("projectPermission").(string)

	if permission != "ROOT" {
		http.Error(w, "you dont have the permission to do this operation", http.StatusConflict)
		return
	}

	var req projectpb.DropProjectReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat decoding to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err := proj.Conn.DropProject(context.TODO(), &req)
	if err != nil {
		helpers.PrintErr(err, "error happenedat DropProject")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "Dropped Project Successfully"
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

func (proj *ProjectCtl) terminateProjectmembers(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("projectPermission").(string)

	if permission != "ROOT" {
		http.Error(w, "you dont have the permission to do this operation", http.StatusConflict)
		return
	}

	var req projectpb.TerminateProjectMembersReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat decoding to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.ProjectID = r.Context().Value("projectID").(string)

	_, err := proj.Conn.TerminateProjectMembers(context.TODO(), &req)
	if err != nil {
		helpers.PrintErr(err, "error happenedat TerminateProjectMembers")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "Terminated Project member Successfully"
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

func (proj *ProjectCtl) updateProjectDetails(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("projectPermission").(string)

	if permission != "ROOT" {
		http.Error(w, "you dont have the permission to do this operation", http.StatusConflict)
		return
	}

	var req projectpb.EditProjectDetailsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat decoding to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.ProjectID = r.Context().Value("projectID").(string)

	_, err := proj.Conn.EditProjectDetails(context.TODO(), &req)
	if err != nil {
		helpers.PrintErr(err, "error happenedat EditProjectDetails")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "Update Project Details Successfully"
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

func (proj *ProjectCtl) EditMember(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("projectPermission").(string)

	if permission != "ROOT" {
		http.Error(w, "you dont have the permission to do this operation", http.StatusConflict)
		return
	}

	var req projectpb.EditMemberReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat decoding to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.ProjectID = r.Context().Value("projectID").(string)

	_, err := proj.Conn.EditMember(context.TODO(), &req)
	if err != nil {
		helpers.PrintErr(err, "error happenedat EditMember")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "Updated Project member Successfully"
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

func (proj *ProjectCtl) editFeedback(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("projectPermission").(string)

	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the permission to do this operation", http.StatusConflict)
		return
	}

	var req projectpb.EditFeedbackReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat decoding to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.ProjectID = r.Context().Value("projectID").(string)

	_, err := proj.Conn.EditFeedback(context.TODO(), &req)
	if err != nil {
		helpers.PrintErr(err, "error happenedat EditFeedback")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "Updated feedback Successfully"
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

func (proj *ProjectCtl) deleteFeedback(w http.ResponseWriter, r *http.Request) {

	permission := r.Context().Value("projectPermission").(string)

	if permission != "ROOT" && permission != "SEMI-ROOT" {
		http.Error(w, "you dont have the permission to do this operation", http.StatusConflict)
		return
	}

	var req projectpb.DeleteFeedbackReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "error happenedat decoding to json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.ProjectID = r.Context().Value("projectID").(string)

	_, err := proj.Conn.DeleteFeedback(context.TODO(), &req)
	if err != nil {
		helpers.PrintErr(err, "error happenedat DeleteFeedback")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res Responce
	res.Message = "Deleted feedback Successfully"
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
