package helpers

import (
	"fmt"
	"math/rand"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
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

func SelectRandomintBetweenRange(min, max int) int {

	rand.New(rand.NewSource(time.Now().UnixNano()))

	return rand.Intn(max-min+1) + min

}

func Serialize(m protoreflect.ProtoMessage) ([]byte, error) {

	serialized, err := proto.Marshal(m)
	if err != nil {
		return nil, err
	}
	return serialized, nil
}