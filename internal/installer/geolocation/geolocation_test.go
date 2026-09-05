package geolocation

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDetect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"city":"Berlin","country":"Germany"}`))
	}))
	defer server.Close()

	got, err := detect(context.Background(), server.Client(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if got.City != "Berlin" || got.Country != "Germany" {
		t.Fatalf("unexpected location: %+v", got)
	}
}

func TestDetectRejectsIncompleteLocation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"city":"","country":"Germany"}`))
	}))
	defer server.Close()

	if _, err := detect(context.Background(), server.Client(), server.URL); err == nil {
		t.Fatal("accepted incomplete location")
	}
}
