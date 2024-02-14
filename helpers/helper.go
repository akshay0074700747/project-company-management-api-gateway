package helpers

import (
	"fmt"

	"google.golang.org/grpc"
)

func DialGrpc(addr string) (*grpc.ClientConn, error) {
	return grpc.Dial(addr, grpc.WithInsecure())
}

func PrintErr(err error, messge string) {
	fmt.Println(messge, err)
}

func PrintMsg(msg string)  {
	fmt.Println(msg)
}
