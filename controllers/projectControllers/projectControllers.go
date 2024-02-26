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

func (proj *ProjectCtl) createProject(w http.ResponseWriter, r *http.Request) {

	projectID := r.Context().Value("projectID").(string)
	if projectID != "" {
		http.Error(w, "you are already logged into a project", http.StatusInternalServerError)
		return
	}

	var isCompanybased bool

	companyID := r.Context().Value("companyID").(string)
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

	ress,err := proj.Conn.LogintoProject(context.TODO(),&projectpb.LogintoProjectReq{
		ProjectUsername: res.ProjectUsername,
		MemberID: usrID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot invoke LogintoCompany")
		return
	}

	cookieString, err := jwtvalidation.GenerateJwtforProject(ress.ProjectID,ress.Role,ress.Permission, []byte(proj.Secret))
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

	res.ProjectID = r.Context().Value("ProjectID").(string)
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

	res.Deadline, err = time.Parse("2006-01-02 15:04:05", r.MultipartForm.Value["Deadline"][0])
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

	projectID := r.Context().Value("ProjectID").(string)
	userID := r.Context().Value("userID").(string)
	queryParams := r.URL.Query()
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

	res,err := proj.Conn.LogintoProject(context.TODO(),&loginReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot invoke LogintoCompany")
		return
	}

	cookieString, err := jwtvalidation.GenerateJwtforProject(res.ProjectID,res.Role,res.Permission, []byte(proj.Secret))
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
	
}

func (proj *ProjectCtl) getAssignedTasks(w http.ResponseWriter, r *http.Request) {
	
}

func (proj *ProjectCtl) getProgressofMembers(w http.ResponseWriter, r *http.Request) {
	
}

func (proj *ProjectCtl) getMemberProgress(w http.ResponseWriter, r *http.Request) {
	
}

func (proj *ProjectCtl) getProjectProgress(w http.ResponseWriter, r *http.Request) {
	
}

func (proj *ProjectCtl) markProgressofNonTechnical(w http.ResponseWriter, r *http.Request) {
	
}

func (proj *ProjectCtl) addTaskStatuses(w http.ResponseWriter, r *http.Request) {
	
}

func (proj *ProjectCtl) getLiveProjectsofCompany(w http.ResponseWriter, r *http.Request) {
	
}