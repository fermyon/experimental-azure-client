package azure

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

type AZCredentials struct {
	AccountName string
	AccountKey  []byte
	Service     string
}

func ParseAZCredentials(accountName, accountKey, service string) (*AZCredentials, error) {
	decodedKey, err := base64.StdEncoding.DecodeString(accountKey)
	if err != nil {
		return nil, fmt.Errorf("decode account key: %v", err)
	}
	return &AZCredentials{AccountName: accountName, AccountKey: decodedKey, Service: service}, nil
}

func ComputeHMACSHA256(c *AZCredentials, message string) (string, error) {
	h := hmac.New(sha256.New, c.AccountKey)
	_, err := h.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(h.Sum(nil)), err
}

func BuildStringToSign(c *AZCredentials, req *http.Request) (string, error) {
	// Returns a blank value if the header value doesn't exist
	getHeader := func(key string, headers http.Header) string {
		if headers == nil {
			return ""
		}
		if v, ok := headers[key]; ok {
			if len(v) > 0 {
				return v[0]
			}
		}
		return ""
	}

	headers := req.Header

	// Per the documentation, the Content-Length field must be an empty string if the content length of the request is zero.
	contentLength := getHeader("Content-Length", headers)
	if contentLength == "0" {
		contentLength = ""
	}

	canonicalizedResource, err := buildCanonicalizedResource(c, req.URL)
	if err != nil {
		return "", err
	}

	stringToSign := strings.Join([]string{
		req.Method,
		getHeader("Content-Encoding", headers),
		getHeader("Content-Language", headers),
		contentLength,
		getHeader("Content-MD5", headers),
		getHeader("Content-Type", headers),
		"", // Empty date because x-ms-date is expected
		getHeader("If-Modified-Since", headers),
		getHeader("If-Match", headers),
		getHeader("If-None-Match", headers),
		getHeader("If-Unmodified-Since", headers),
		getHeader("Range", headers),
		buildCanonicalizedHeader(headers),
		canonicalizedResource,
	}, "\n")

	return stringToSign, nil
}

// buildCanonicalizedHeader retrieves all headers which start with 'x-ms-' and creates a lexicographically sorted string:
// x-ms-header-a:foo,\nx-ms-header-b:bar,\nx-ms-header-0:baz\n
func buildCanonicalizedHeader(headers http.Header) string {
	cm := map[string][]string{}
	for k, v := range headers {
		headerName := strings.TrimSpace(strings.ToLower(k))
		if strings.HasPrefix(headerName, "x-ms-") {
			cm[headerName] = v
		}
	}
	if len(cm) == 0 {
		return ""
	}

	keys := make([]string, 0, len(cm))
	for key := range cm {
		keys = append(keys, key)
	}
	sort.Strings(keys) // Canonicalized headers must be in lexicographical order

	ch := bytes.NewBufferString("")
	for i, key := range keys {
		if i > 0 {
			ch.WriteRune('\n')
		}
		ch.WriteString(key)
		ch.WriteRune(':')
		ch.WriteString(strings.Join(cm[key], ","))
	}
	return ch.String()
}

func buildCanonicalizedResource(c *AZCredentials, u *url.URL) (string, error) {
	cr := bytes.NewBufferString("/")
	cr.WriteString(c.AccountName)

	if len(u.Path) > 0 {
		cr.WriteString(u.EscapedPath())
	} else {
		cr.WriteString("/")
	}

	params, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return "", fmt.Errorf("failed to parse query params: %v", err)
	}

	if len(params) > 0 {
		var paramNames []string
		for paramName := range params {
			paramNames = append(paramNames, paramName)
		}
		sort.Strings(paramNames)

		for _, paramName := range paramNames {
			paramValues := params[paramName]
			sort.Strings(paramValues)
			cr.WriteString("\n" + strings.ToLower(paramName) + ":" + strings.Join(paramValues, ","))
		}
	}

	return cr.String(), nil
}
