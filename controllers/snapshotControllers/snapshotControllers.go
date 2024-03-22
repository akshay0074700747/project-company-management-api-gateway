package snapshotcontrollers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/IBM/sarama"
	"github.com/akshay0074700747/projectandCompany_management_api-gateway/helpers"
)

type SnapMsg struct {
	Email     string `json:"email"`
	ProjectID string `json:"project_id"`
	Snapshots []Snap `json:"snapshots"`
	Key       string `json:"key"`
	IsStaged  bool   `json:"is_staged"`
}

type Snap struct {
	File        []byte `json:"file"`
	Filename    string `json:"file_name"`
	Key         string `json:"key"`
	Description string `json:"description"`
	Progress    int    `json:"progress"`
}

func (snap *SnapshotCtl) pushSnapshots(w http.ResponseWriter, r *http.Request) {

	fmt.Println("pusshingggggg....")

	var snapmsg SnapMsg

	err := r.ParseMultipartForm(r.ContentLength)
	if err != nil {
		helpers.PrintErr(err, "error happened at parsing form")
		http.Error(w, "failed to parse multipart form data", http.StatusInternalServerError)
		return
	}

	snapmsg.Email = r.MultipartForm.Value["email"][0]
	snapmsg.ProjectID = r.MultipartForm.Value["project_id"][0]
	snapmsg.Key = r.MultipartForm.Value["key"][0]
	staged := r.MultipartForm.Value["isStaged"][0]
	if staged == "true" {
		snapmsg.IsStaged = true
	}
	files := r.MultipartForm.File["files"]
	keys := r.MultipartForm.Value["keys"]
	descriptions := r.MultipartForm.Value["descriptions"]
	progresses := r.MultipartForm.Value["progresses"]

	for i, file := range files {

		zipFile, err := file.Open()
		if err != nil {
			helpers.PrintErr(err, "error occured at opening the snapshot file ")
			http.Error(w, err.Error(), http.StatusBadRequest)
			continue
		}
		defer zipFile.Close()

		pro, _ := strconv.Atoi(progresses[i])
		buffer := new(bytes.Buffer)
		_, err = buffer.ReadFrom(zipFile)
		if err != nil {
			helpers.PrintErr(err, "error at reading from zipfile")
			return
		}

		snapmsg.Snapshots = append(snapmsg.Snapshots, Snap{
			File:        buffer.Bytes(),
			Filename:    file.Filename,
			Key:         keys[i],
			Description: descriptions[i],
			Progress:    pro,
		})
	}

	msgBytes, err := json.Marshal(snapmsg)
	if err != nil {
		helpers.PrintErr(err, "error encoding to json")
		return
	}

	msg := &sarama.ProducerMessage{
		Topic:     snap.Topic,
		Partition: 0,
		Value:     sarama.ByteEncoder(msgBytes),
	}
	_, offset, err := snap.Producer.SendMessage(msg)
	if err != nil {
		helpers.PrintErr(err, "error sending message to Kafka")
		return
	}
	fmt.Println(offset, " completed...")
	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write([]byte("pushed successfully..."))

}

func (snap *SnapshotCtl) pullSnapshot(w http.ResponseWriter, r *http.Request) {

	url := fmt.Sprintf("http://snapshot-service:50005/snapshots/pull?commitID=%s", r.URL.Query().Get("commitID"))

	http.Redirect(w, r, url, http.StatusFound)

}
