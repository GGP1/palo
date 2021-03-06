package cookie

import (
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GGP1/adak/internal/crypt"

	"github.com/stretchr/testify/assert"
)

func TestDelete(t *testing.T) {
	w := httptest.NewRecorder()

	value := "adak"
	name := "test-delete"
	http.SetCookie(w, &http.Cookie{
		Name:  name,
		Value: value,
		Path:  "/",
	})

	Delete(w, name)

	assert.NotEqual(t, 0, len(w.Result().Cookies()))
}

func TestGet(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)

	expected := "adak"
	ciphertext, err := crypt.Encrypt([]byte(expected))
	assert.NoError(t, err)

	name := "test-get"
	r.AddCookie(&http.Cookie{
		Name:  name,
		Value: hex.EncodeToString(ciphertext),
		Path:  "/",
	})

	got, err := Get(r, name)
	assert.NoError(t, err)

	assert.Equal(t, expected, got.Value)
}

func TestGetValue(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)

	expected := "adak"
	ciphertext, err := crypt.Encrypt([]byte(expected))
	assert.NoError(t, err)

	name := "test-get"
	r.AddCookie(&http.Cookie{
		Name:  name,
		Value: hex.EncodeToString(ciphertext),
		Path:  "/",
	})

	got, err := GetValue(r, name)
	assert.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestGetErrors(t *testing.T) {
	t.Run("Cookie isn't set", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		_, err := Get(r, "invalid")
		assert.Error(t, err)
	})

	t.Run("Invalid hex value", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		r.AddCookie(&http.Cookie{
			Name:  "test",
			Value: "fail",
			Path:  "/",
		})

		_, err := Get(r, "test")
		assert.Error(t, err)

	})
}

func TestIsSet(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)

	t.Run("Not set", func(t *testing.T) {
		if IsSet(r, "not-set") {
			t.Error("Expected false, got true")
		}
	})

	t.Run("Set", func(t *testing.T) {
		r.AddCookie(&http.Cookie{
			Name:  "set",
			Value: "test",
			Path:  "/",
		})

		if !IsSet(r, "set") {
			t.Error("Expected true, got false")
		}
	})
}

func TestSet(t *testing.T) {
	w := httptest.NewRecorder()

	value := "adak"
	name := "test-set"

	err := Set(w, name, value, "/", 0)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(w.Result().Cookies()))
}
