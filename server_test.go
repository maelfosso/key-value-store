package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSON(t *testing.T) {
	// in := map[string]string{"hello": "world"}
	// out := `{"hello":"world"}`

	header := http.Header{}
	headerKey := "Content-Type"
	headerValue := "application/json; charset=utf-8"
	header.Add(headerKey, headerValue)

	testCases := []struct {
		in     interface{}
		header http.Header
		out    string
	}{
		{map[string]string{"hello": "world"}, header, `{"hello":"world"}`},
		{map[string]string{"hello": "tables"}, header, `{"hello":"tables"}`},
		{make(chan bool), header, `{"error":"json: unsupported type: chan bool"}`},
	}

	for _, test := range testCases {
		recorder := httptest.NewRecorder()

		JSON(recorder, test.in)

		response := recorder.Result()
		defer response.Body.Close()

		got, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatalf("Error reading response body: %s", err)
		}

		if string(got) != test.out {
			t.Errorf("Got %s, expected %s", string(got), test.out)
		}

		if contentType := response.Header.Get(headerKey); contentType != headerValue {
			t.Errorf("Got %s, expected %s", contentType, headerValue)
		}
	}
}
