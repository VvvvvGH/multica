package handler

import (
	"net/http"
	"strings"
	"testing"

	"github.com/multica-ai/multica/server/internal/testutil"
)

func TestPasswordAuthentication(t *testing.T) {
	h := *testHandler
	h.cfg.PasswordAuthEnabled = true
	email := "password-auth-test@example.com"
	uid := dbfx.User(t, "Password Test", email)
	dbfx.Cleanup(t, "DELETE FROM user_password WHERE user_id = $1", uid)
	password := "test-only-password-2026"
	request := func(method, path string, body any) *http.Request {
		r := newRequest(method, path, body)
		r.Header.Set("X-User-ID", uid)
		return r
	}

	testutil.Call(t, h.SetPassword, request("PUT", "/api/me/password", map[string]string{"password": "short"})).Want(400)
	testutil.Call(t, h.SetPassword, request("PUT", "/api/me/password", map[string]string{"password": strings.Repeat("a", 73)})).Want(400)
	testutil.Call(t, h.SetPassword, request("PUT", "/api/me/password", map[string]string{"password": password})).Want(204)
	status := testutil.Call(t, h.GetPasswordStatus, request("GET", "/api/me/password", nil)).Want(200).Map()
	if status["configured"] != true {
		t.Fatalf("password should be configured: %v", status)
	}
	var hash string
	if err := testPool.QueryRow(t.Context(), "SELECT password_hash FROM user_password WHERE user_id = $1", uid).Scan(&hash); err != nil {
		t.Fatal(err)
	}
	if hash == password || !strings.HasPrefix(hash, "$2") {
		t.Fatal("password was not stored as a bcrypt hash")
	}

	for _, input := range []map[string]string{
		{"email": email, "password": "wrong-password"},
		{"email": "missing-password-user@example.com", "password": password},
	} {
		resp := testutil.Call(t, h.PasswordLogin, testutil.JSONRequest("POST", "/auth/password", input)).Want(401)
		if resp.Map()["error"] != "invalid email or password" || len(resp.Result().Cookies()) != 0 {
			t.Fatal("rejected login revealed account state or created cookies")
		}
	}
	login := testutil.Call(t, h.PasswordLogin, testutil.JSONRequest("POST", "/auth/password", map[string]string{"email": "  PASSWORD-AUTH-TEST@EXAMPLE.COM ", "password": password})).Want(200)
	if login.Map()["token"] == "" || len(login.Result().Cookies()) == 0 {
		t.Fatal("password login did not establish a normal session")
	}
	testutil.Call(t, h.SetPassword, request("PUT", "/api/me/password", map[string]string{"password": "replacement-password"})).Want(403)
	testutil.Call(t, h.SetPassword, request("PUT", "/api/me/password", map[string]string{"current_password": password, "password": "replacement-password"})).Want(204)
	testutil.Call(t, h.PasswordLogin, testutil.JSONRequest("POST", "/auth/password", map[string]string{"email": email, "password": password})).Want(401)
	testutil.Call(t, h.PasswordLogin, testutil.JSONRequest("POST", "/auth/password", map[string]string{"email": email, "password": "replacement-password"})).Want(200)
}

func TestPasswordAuthenticationDisabled(t *testing.T) {
	h := *testHandler
	h.cfg.PasswordAuthEnabled = false
	testutil.Call(t, h.PasswordLogin, testutil.JSONRequest("POST", "/auth/password", map[string]string{"email": handlerTestEmail, "password": "test-password"})).Want(404)
	testutil.Call(t, h.SetPassword, newRequest("PUT", "/api/me/password", map[string]string{"password": "test-password"})).Want(404)
}
