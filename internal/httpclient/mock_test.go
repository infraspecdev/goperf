package httpclient

import "net/http"

type MockHTTPDoer struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	if m.DoFunc != nil {
		return m.DoFunc(req)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
	}, nil
}
