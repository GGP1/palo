package rest_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GGP1/palo/pkg/http/rest"
)

// Checkboxes
const (
	succeed = "\u2713"
	failed  = "\u2717"
)

// Fix
func TestRouting(t *testing.T) {
	srv := httptest.NewServer(rest.NewRouter())
	defer srv.Close()

	t.Log("Given the need to test the router.")
	{
		t.Logf("\tTest 0: When checking GET request.")
		{
			res, err := http.Get("http://localhost:4000/")
			if err != nil {
				t.Errorf("\t%s\tShould return a response: %v", failed, err)
			}
			t.Logf("\t%s\tShould return a response.", succeed)

			if res.StatusCode != http.StatusOK {
				t.Errorf("\t%s\tShould be status OK: got %v", failed, res.StatusCode)
			}
			t.Logf("\t%s\tShould be status OK.", succeed)
		}
	}

}