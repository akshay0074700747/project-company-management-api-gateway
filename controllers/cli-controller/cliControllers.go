package clicontroller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/akshay0074700747/projectandCompany_management_api-gateway/helpers"
	"github.com/akshay0074700747/projectandCompany_management_api-gateway/rediss"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/joho/godotenv"
)

type CliCtl struct {
	Client     *s3.S3
	BucketName string
	Cache      *rediss.Cache
}

func NewCliCtl(cache *rediss.Cache) *CliCtl {
	if err := godotenv.Load(".env"); err != nil {
		helpers.PrintErr(err, "the secret cannot be retrieved...")
	}
	accessKey := os.Getenv("accesskey")
	secretAccessKey := os.Getenv("secretaccess")

	creds := credentials.NewStaticCredentials(accessKey, secretAccessKey, "")

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("ap-south-1"),
		Credentials: creds,
	})
	if err != nil {
		fmt.Println("Error creating session:", err)
	}

	svc := s3.New(sess)

	return &CliCtl{
		Client:     svc,
		BucketName: "project-and-company-management-cli-utitlity-tracker",
		Cache:      cache,
	}
}

type GetDownloads struct {
	Key  string `json:"Key"`
	Size string `json:"Size"`
}

var (
	converter float32
)

func (cli *CliCtl) getDownloads(w http.ResponseWriter, r *http.Request) {

	input := &s3.ListObjectsInput{
		Bucket: aws.String(cli.BucketName),
	}

	var result []GetDownloads

	if err := cli.Client.ListObjectsPages(input, func(page *s3.ListObjectsOutput, lastPage bool) bool {
		converter = 1048576
		for _, object := range page.Contents {
			sizeMB := float32(*object.Size) / converter
			result = append(result, GetDownloads{
				Key:  *object.Key,
				Size: fmt.Sprintf("%f MB", sizeMB),
			})

		}
		return true
	}); err != nil {
		fmt.Println("Error listing objects:", err)
		return
	}

	jsondta, err := json.Marshal(result)
	if err != nil {
		http.Error(w, "there is a problem with parsing to json", http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot parse to json on getdownloads")
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsondta)
}

func (cli *CliCtl) downloadCli(w http.ResponseWriter, r *http.Request) {

	objKey := r.URL.Query().Get("objectKey")

	fmt.Println(objKey)

	if objKey == "" {
		http.Error(w, "the object key cannot be empty", http.StatusInternalServerError)
		return
	}

	var res []byte
	if err := cli.Cache.GetDataFromCache(objKey, &res, context.TODO()); err != nil {
		input := &s3.GetObjectInput{
			Bucket: aws.String(cli.BucketName),
			Key:    aws.String(objKey),
		}

		result, err := cli.Client.GetObject(input)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			helpers.PrintErr(err, "error at getting the object")
			return
		}

		fmt.Println(*result.ContentLength, " ----------size")

		val := fmt.Sprintf("attachment; filename=%s", objKey)

		w.Header().Set("Content-Disposition", val)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", *result.ContentLength))

		if _, err = io.Copy(w, result.Body); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			helpers.PrintErr(err, "cannot copy from reader to wirter")
			return
		}

		go func() {

			defer result.Body.Close()

			var bytRes = make([]byte, int(*result.ContentLength))

			_, err := result.Body.Read(bytRes)
			if err != nil && err != io.EOF {
				helpers.PrintErr(err, "eror happened at reading from result")
			}

			// bytRes, err := io.ReadAll(result.Body)
			// if err != nil {
			// 	helpers.PrintErr(err, "eror happened at reading from result")
			// }

			fmt.Println(len(bytRes), "--sjdbhakhjcb")

			if err = cli.Cache.CacheData(objKey, bytRes, time.Hour*24, context.TODO()); err != nil {
				helpers.PrintErr(err, "cannot cache")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			fmt.Println("successfully cached...")

		}()

	} else {

		reader := bytes.NewReader(res)

		val := fmt.Sprintf("attachment; filename=%s", objKey)

		w.Header().Set("Content-Disposition", val)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", reader.Size()))

		if _, err = io.Copy(w, reader); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			helpers.PrintErr(err, "cannot copy from reader to wirter")
			return
		}

		fmt.Println("successfully got from cache...")
	}

	// input := &s3.GetObjectInput{
	// 	Bucket: aws.String(cli.BucketName),
	// 	Key:    aws.String(objKey),
	// }

	// result, err := cli.Client.GetObject(input)
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	helpers.PrintErr(err, "error at getting the object")
	// 	return
	// }

	// defer result.Body.Close()

	// fmt.Println(*result.ContentLength, " ----------size")

	// val := fmt.Sprintf("attachment; filename=%s", objKey)

	// w.Header().Set("Content-Disposition", val)
	// w.Header().Set("Content-Type", "application/octet-stream")
	// w.Header().Set("Content-Length", fmt.Sprintf("%d", *result.ContentLength))

	// if _, err = io.Copy(w, result.Body); err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	helpers.PrintErr(err, "cannot copy from reader to wirter")
	// 	return
	// }

}
