package main

import (
	"io"
	"net/http"
	"testing"
)

func TestParseAZCredentials(t *testing.T) {
	goodCreds := AZCredentials{
		AccountName: "testaccount",
		AccountKey:  []byte("testkey"),
		Service:     "testservice",
	}

	tests := []struct {
		name          string
		accountName   string
		accountKey    string
		service       string
		expectedCreds *AZCredentials
		expectedError bool
	}{{
		name:          "properly base64 encoded account key",
		accountName:   "testaccount",
		accountKey:    "dGVzdGtleQ==",
		service:       "testservice",
		expectedCreds: &goodCreds,
		expectedError: false,
	}, {
		name:          "non-base64 encoded account key",
		accountName:   "testaccount",
		accountKey:    "T3=$stK3y@", // Contains characters not in (A-Z, a-z, 0-9, +, /), and the length of the string is not a multiple of four, and there's a misplaced padding character.
		service:       "testservice",
		expectedCreds: nil,
		expectedError: true,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds, err := parseAZCredentials(tt.accountName, tt.accountKey, tt.service)
			if creds != tt.expectedCreds {
				t.Errorf("got: %v, want: %v", creds, tt.expectedCreds)
			}

			if (tt.expectedError && err == nil) || (!tt.expectedError && err != nil) {
				t.Errorf("got: %v, want: %v", err, tt.expectedError)
			}
		})
	}
}

func TestComputeHMACSHA256(t *testing.T) {
	testCreds := AZCredentials{
		AccountKey: []byte("testkey"),
	}

	tests := []struct {
		name           string
		creds          *AZCredentials
		message        string
		expectedString string
		expectedError  bool
	}{{
		name:           "initial test",
		creds:          &testCreds,
		message:        "Hello, world!",
		expectedString: "JcFgXrIZPVbHKy1GWjQzi/KN+3hFaA/g2+Dn9JV7UAM=",
		expectedError:  false,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str, err := computeHMACSHA256(tt.creds, tt.message)

			if str != tt.expectedString {
				t.Errorf("got: %v, want: %v", str, tt.expectedString)
			}

			if (tt.expectedError && err == nil) || (!tt.expectedError && err != nil) {
				t.Errorf("got: %v, want: %v", err, tt.expectedError)
			}
		})
	}
}

func TestBuildStringToSign(t *testing.T) {
	buildReq := func(method, endpoint string, payload io.Reader, headers map[string]string) (*http.Request, error) {
		req, err := http.NewRequest(method, endpoint, payload)
		if err != nil {
			return nil, err
		}

		for key, value := range headers {
			req.Header.Set(key, value)
		}

		return req, nil
	}

	testCreds := AZCredentials{
		AccountName: "testaccount",
		AccountKey:  []byte("testkey"),
		Service:     "testservice",
	}

	goodReq, err := buildReq("GET", "https://localhost:3000/testpath?foo=bar&bar=baz", nil, map[string]string{
		"x-ms-header-a": "foo",
		"x-ms-header-b": "bar",
		"x-ms-header-0": "baz",
		"Content-Type":  "test/type",
	})
	if err != nil {
		t.Errorf("unexpected error: %w", err)
	}

	emptyReq, err := buildReq("GET", "https://localhost:3000", nil, map[string]string{})
	if err != nil {
		t.Errorf("unexpected error: %w", err)
	}

	tests := []struct {
		name           string
		creds          *AZCredentials
		req            *http.Request
		expectedString string
		expectedError  bool
	}{{
		name:           "normal request",
		creds:          &testCreds,
		req:            goodReq,
		expectedString: "GET\n\n\n\n\ntest/type\n\n\n\n\n\n\nx-ms-header-0:baz,\nx-ms-header-a:foo,\nx-ms-header-b:bar\n/testaccount/testpath\nbar:baz\nfoo:bar",
		expectedError:  false,
	}, {
		name:           "empty headers",
		creds:          &testCreds,
		req:            emptyReq,
		expectedString: "GET\n\n\n\n\n\n\n\n\n\n\n\n\n/testaccount/",
		expectedError:  false,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str, err := buildStringToSign(tt.creds, tt.req)
			if str != tt.expectedString {
				t.Errorf("got: %v, want: %v", str, tt.expectedString)
			}

			if (tt.expectedError && err == nil) || (!tt.expectedError && err != nil) {
				t.Errorf("got: %v, want: %v", err, tt.expectedError)
			}
		})
	}
}
