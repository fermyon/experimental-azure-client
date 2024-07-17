package main

// This was built using the Azure API documentation: https://learn.microsoft.com/en-us/rest/api/storageservices/authorize-with-shared-key

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fermyon/experimental-azure-client/azure"
	spinhttp "github.com/fermyon/spin/sdk/go/v2/http"
	"github.com/fermyon/spin/sdk/go/v2/variables"
)

func init() {
	spinhttp.Handle(func(w http.ResponseWriter, r *http.Request) {
		accountName, err := variables.Get("az_account_name")
		if err != nil {
			http.Error(w, "Error retrieving Azure account name", http.StatusInternalServerError)
			return
		}

		sharedKey, err := variables.Get("az_shared_key")
		if err != nil {
			http.Error(w, "Error retrieving Azure shared_key", http.StatusInternalServerError)
			return
		}

		service := r.Header.Get("x-az-service")
		if service == "" {
			http.Error(w, "ERROR: You must include the 'x-az-service' header in your request", http.StatusBadRequest)
			return
		}

		// This gets the vital portions of the uri path, while excluding the route path defined in the spin.toml file
		//See https://developer.fermyon.com/spin/v2/http-trigger#additional-request-information
		uriPath := r.Header.Get("spin-path-info")
		queryString := r.URL.RawQuery
		endpoint := fmt.Sprintf("https://%s.%s.core.windows.net", accountName, service)

		if len(queryString) == 0 {
			if uriPath == "/" {
				http.Error(w, fmt.Sprint("If you are not including a query string, you must have a more specific URI path (i.e. /containerName/path/to/object)"), http.StatusBadRequest)
				return
			} else {
				endpoint += uriPath
			}
		} else {
			endpoint += uriPath + "?" + queryString
		}

		now := time.Now().UTC()

		bodyData, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to read request body: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		r.Body.Close()

		req, err := http.NewRequest(r.Method, endpoint, bytes.NewReader(bodyData))
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create http request: %s", err.Error()), http.StatusInternalServerError)
		}

		resp, err := sendAzureRequest(req, now, accountName, sharedKey, service)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to execute outbound http request: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to read outbound http response: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			http.Error(w, fmt.Sprintf("Response from outbound http request is not OK:\n%v\n%v", resp.Status, string(body)), http.StatusBadRequest)
			return
		}

		w.WriteHeader(resp.StatusCode)

		if len(body) == 0 {
			w.Write([]byte("Response from Azure: " + resp.Status))
		} else {
			w.Write(body)
		}
	})
}

// sendAzureRequest was built to interact with the Blob Storage and Storage Queue services. Please see Microsoft's documentation for other Azure services: https://learn.microsoft.com/en-us/rest/api/azure/
func sendAzureRequest(req *http.Request, now time.Time, accountName, sharedKey, service string) (*http.Response, error) {
	cred, err := azure.ParseAZCredentials(accountName, sharedKey, service)
	if err != nil {
		fmt.Println("Error creating credential:", err)
		return nil, err
	}

	// Setting universally required headers
	req.Header.Set("x-ms-date", now.Format(http.TimeFormat))
	req.Header.Set("x-ms-version", "2024-08-04") // Although not technically required, we strongly recommend specifying the latest Azure Storage API version: https://learn.microsoft.com/en-us/rest/api/storageservices/versioning-for-the-azure-storage-services

	// Setting method and service-specific headers
	if req.Method == "PUT" || req.Method == "POST" {
		req.Header.Set("content-length", fmt.Sprintf("%d", req.ContentLength))

		if service == "blob" {
			req.Header.Set("x-ms-blob-type", "BlockBlob")
		}
	}

	stringToSign, err := azure.BuildStringToSign(cred, req)
	if err != nil {
		fmt.Println("Error building string to sign:", err)
		return nil, err
	}

	signature, err := azure.ComputeHMACSHA256(cred, stringToSign)
	if err != nil {
		fmt.Println("Error computing signature:", err)
		return nil, err
	}
	authHeader := fmt.Sprintf("SharedKey %s:%s", accountName, signature)
	req.Header.Set("authorization", authHeader)

	return spinhttp.Send(req)
}

func main() {}
