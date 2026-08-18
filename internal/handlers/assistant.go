package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// AssistantHandlers renders the AI operations assistant web page.
type AssistantHandlers struct {
	*App
}

func (h *AssistantHandlers) Routes(r chi.Router) {
	r.Get("/", h.Index)
}

// GET /assistant — chat UI page.
func (h *AssistantHandlers) Index(w http.ResponseWriter, r *http.Request) {
	session, _ := h.getUserFromContext(r)

	h.renderPage(w, r, "assistant.html", PageData{
		Title: "AI Assistant",
		User:  session,
	})
}
