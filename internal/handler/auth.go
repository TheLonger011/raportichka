package handler

import (
	"encoding/json"
	"net/http"

	"github.com/TheLonger011/raportichka/internal/auth"
	"github.com/TheLonger011/raportichka/internal/domain"
)

func (h *Handler) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	sess := h.sessions.GetSession(r)
	if sess != nil {
		if sess.Role == "student" {
			http.Redirect(w, r, "/student", http.StatusFound)
		} else {
			http.Redirect(w, r, "/", http.StatusFound)
		}
		return
	}
	http.ServeFile(w, r, "pages/login/index.html")
}

func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(auth.SessionCookie)
	if err == nil {
		h.sessions.Delete(c.Value)
	}
	h.sessions.ClearCookie(w)
	http.Redirect(w, r, "/login", http.StatusFound)
}

func (h *Handler) handleAPILogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Role     string `json:"role"`
		FullName string `json:"full_name"`
		Password string `json:"password"`
		GroupID  *int   `json:"group_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	user, err := h.storage.Login(req.FullName, req.Password, domain.Role(req.Role), req.GroupID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusUnauthorized)
		return
	}

	sess := &auth.SessionData{
		UserID:    user.ID,
		FullName:  user.FullName,
		Role:      string(user.Role),
		GroupID:   user.GroupID,
		GroupName: user.GroupName,
	}
	sid := h.sessions.Create(sess)
	h.sessions.SetCookie(w, sid)

	redirect := "/"
	if user.Role == domain.RoleStudent {
		redirect = "/student"
	}
	jsonResp(w, map[string]string{"redirect": redirect})
}

func (h *Handler) handleAPIMe(w http.ResponseWriter, r *http.Request) {
	sess := h.sessions.GetSession(r)
	if sess == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	auth.WriteJSON(w, sess)
}
