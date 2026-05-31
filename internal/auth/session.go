package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

const (
	SessionCookie = "sid"
	sessionTTL    = 24 * time.Hour
)

type SessionData struct {
	UserID    int
	FullName  string
	Role      string
	GroupID   *int
	GroupName string
	CreatedAt time.Time
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*SessionData
}

func NewSessionStore() *SessionStore {
	s := &SessionStore{sessions: make(map[string]*SessionData)}
	go func() {
		ticker := time.NewTicker(time.Hour)
		for range ticker.C {
			s.cleanup()
		}
	}()
	return s
}

func (s *SessionStore) Create(data *SessionData) string {
	id := randomHex(32)
	data.CreatedAt = time.Now()
	s.mu.Lock()
	s.sessions[id] = data
	s.mu.Unlock()
	return id
}

func (s *SessionStore) Get(id string) *SessionData {
	s.mu.RLock()
	d, ok := s.sessions[id]
	s.mu.RUnlock()
	if !ok || time.Since(d.CreatedAt) > sessionTTL {
		return nil
	}
	return d
}

func (s *SessionStore) Delete(id string) {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}

func (s *SessionStore) cleanup() {
	s.mu.Lock()
	for k, v := range s.sessions {
		if time.Since(v.CreatedAt) > sessionTTL {
			delete(s.sessions, k)
		}
	}
	s.mu.Unlock()
}

func (s *SessionStore) GetSession(r *http.Request) *SessionData {
	c, err := r.Cookie(SessionCookie)
	if err != nil {
		return nil
	}
	return s.Get(c.Value)
}

func (s *SessionStore) SetCookie(w http.ResponseWriter, sid string) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    sid,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionTTL.Seconds()),
	})
}

func (s *SessionStore) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:   SessionCookie,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}

func WriteJSON(w http.ResponseWriter, sess *SessionData) {
	type resp struct {
		UserID    int    `json:"user_id"`
		FullName  string `json:"full_name"`
		Role      string `json:"role"`
		GroupID   *int   `json:"group_id"`
		GroupName string `json:"group_name"`
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp{
		UserID:    sess.UserID,
		FullName:  sess.FullName,
		Role:      sess.Role,
		GroupID:   sess.GroupID,
		GroupName: sess.GroupName,
	})
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
