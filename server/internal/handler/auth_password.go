package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/multica-ai/multica/server/internal/auth"
	"golang.org/x/crypto/bcrypt"
)

const passwordCost = 12

// A missing credential still performs a bcrypt comparison before rejecting it.
const dummyPasswordHash = "$2a$12$R9h/cIPz0gi.URNNX3kh2OPST9/PgBkqquzi.Ss7KIUgO2t0jWMUW"

func (h *Handler) PasswordLogin(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.PasswordAuthEnabled {
		http.NotFound(w, r)
		return
	}
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	user, userErr := h.Queries.GetUserByEmail(r.Context(), strings.ToLower(strings.TrimSpace(req.Email)))
	if userErr != nil && !errors.Is(userErr, pgx.ErrNoRows) {
		writeError(w, http.StatusInternalServerError, "failed to load account")
		return
	}
	hash := dummyPasswordHash
	credentialFound := false
	if userErr == nil {
		err := h.DB.QueryRow(r.Context(), "SELECT password_hash FROM user_password WHERE user_id = $1", user.ID).Scan(&hash)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusInternalServerError, "failed to load credential")
			return
		}
		credentialFound = err == nil
	}
	comparison := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password))
	if comparison != nil || !credentialFound || auth.IsTemporarilyDisabledUser(uuidToString(user.ID), user.Email) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	token, err := h.issueJWT(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}
	if err := auth.SetAuthCookies(w, token); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, LoginResponse{Token: token, User: h.userToResponse(user)})
}

func (h *Handler) GetPasswordStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	configured := false
	if h.cfg.PasswordAuthEnabled {
		if err := h.DB.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM user_password WHERE user_id = $1)", parseUUID(userID)).Scan(&configured); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load password status")
			return
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]bool{"enabled": h.cfg.PasswordAuthEnabled, "configured": configured})
}

func (h *Handler) SetPassword(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.PasswordAuthEnabled {
		http.NotFound(w, r)
		return
	}
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	var req struct {
		Password        string `json:"password"`
		CurrentPassword string `json:"current_password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.Password) < 8 || len(req.Password) > 72 {
		writeError(w, http.StatusBadRequest, "password must contain 8 to 72 bytes")
		return
	}
	uid := parseUUID(userID)
	// Resolve the account before writing a credential; task tokens are also
	// rejected by RequireHumanActor at the routing boundary.
	if _, err := h.Queries.GetUser(r.Context(), uid); err != nil {
		writeError(w, http.StatusUnauthorized, "account not found")
		return
	}
	var oldHash string
	err := h.DB.QueryRow(r.Context(), "SELECT password_hash FROM user_password WHERE user_id = $1", uid).Scan(&oldHash)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusInternalServerError, "failed to load credential")
		return
	}
	if err == nil && bcrypt.CompareHashAndPassword([]byte(oldHash), []byte(req.CurrentPassword)) != nil {
		writeError(w, http.StatusForbidden, "current password is incorrect")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), passwordCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}
	tag, err := h.DB.Exec(r.Context(), `INSERT INTO user_password (user_id, password_hash)
		VALUES ($1, $2) ON CONFLICT (user_id) DO UPDATE SET password_hash = EXCLUDED.password_hash, updated_at = now()
		WHERE user_password.password_hash = $3`, uid, string(hash), oldHash)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save password")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusConflict, "password changed; reload and try again")
		return
	}
	slog.Info("account password updated", "user_id", userID)
	w.WriteHeader(http.StatusNoContent)
}
