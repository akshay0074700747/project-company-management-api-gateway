package chatcontrollers

import (
	"fmt"
	"net/http"
)

func chatConteroller(w http.ResponseWriter, r *http.Request) {

	fmt.Println("======here=======websocket========")

	http.Redirect(w, r, "http://chat-service.default.svc.cluster.local:50006/ws", http.StatusFound)
}
