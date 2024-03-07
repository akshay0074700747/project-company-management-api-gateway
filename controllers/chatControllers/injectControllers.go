package chatcontrollers

import "github.com/go-chi/chi"

func InjectChatControllers(r *chi.Mux) {
	r.Get("/ws", chatConteroller)
}
