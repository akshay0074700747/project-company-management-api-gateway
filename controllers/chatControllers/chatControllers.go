package chatcontrollers

import (
	"fmt"
	"net/http"
)

func chatConteroller(w http.ResponseWriter, r *http.Request) {
	fmt.Println("=====================================here=======================================")
	http.Redirect(w, r, "http://localhost:50006/ws", http.StatusFound)
}
