package node

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPNode_Execute_Success(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	httpNode := NewHTTPNode("test-node", "Test HTTP Node", client, 5*time.Second, 3)

	builder := &JSONRequestBuilder{
		Method:  "GET",
		URL:     server.URL,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    "",
	}

	input := HTTPRequestData{Builder: builder}
	response, err := httpNode.Execute(input)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", response.StatusCode)
	}
}

func TestHTTPNode_Execute_Failure(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	httpNode := NewHTTPNode("test-node", "Test HTTP Node", client, 5*time.Second, 1)

	builder := &JSONRequestBuilder{
		Method:  "GET",
		URL:     server.URL,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    "",
	}

	input := HTTPRequestData{Builder: builder}
	response, err := httpNode.Execute(input)

	if err != nil {
		t.Fatalf("UnExpected an error %v", err)
	}

	if response.StatusCode != 404 {
		t.Errorf("Expected status code 404, got %d", response.StatusCode)
	}
}

func TestHTTPNode_Execute_Timeout(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second)
		w.WriteHeader(http.StatusOK)
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	client := &http.Client{Timeout: 1 * time.Second}
	httpNode := NewHTTPNode("test-node", "Test HTTP Node", client, 1*time.Second, 1)

	builder := &JSONRequestBuilder{
		Method:  "GET",
		URL:     server.URL,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    "",
	}

	input := HTTPRequestData{Builder: builder}
	_, err := httpNode.Execute(input)

	if err == nil {
		t.Fatalf("Expected a timeout error, but got none")
	}
}

func TestHTTPNode_Execute_POST(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"message": "created"}`))
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	httpNode := NewHTTPNode("test-node", "Test HTTP Node", client, 5*time.Second, 3)

	builder := &JSONRequestBuilder{
		Method:  "POST",
		URL:     server.URL,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    "{\"name\": \"test\"}",
	}

	input := HTTPRequestData{Builder: builder}
	response, err := httpNode.Execute(input)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if response.StatusCode != http.StatusCreated {
		t.Errorf("Expected status code 201, got %d", response.StatusCode)
	}
}

func TestHTTPNode_Execute_Multipart(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Проверка типа контента
		contentType := r.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "multipart/form-data") {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}

		// Разбор multipart данных
		err := r.ParseMultipartForm(10 << 20) // 10 MB max
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Проверка наличия поля
		fieldValue := r.FormValue("field1")
		if fieldValue != "value1" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// Проверка наличия файла
		file, fileHeader, err := r.FormFile("file1")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer file.Close()

		if fileHeader.Filename != "file.txt" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("multipart success"))
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	httpNode := NewHTTPNode("test-node", "Test HTTP Node", client, 5*time.Second, 3)

	fileContent := []byte("file content")
	builder := &MultipartRequestBuilder{
		Method: "POST",
		URL:    server.URL,
		Fields: map[string]string{"field1": "value1"},
		Files: map[string]FileData{"file1": {Filename: "file.txt",
			Reader: bytes.NewReader(fileContent)},
		},
	}

	input := HTTPRequestData{Builder: builder}
	response, err := httpNode.Execute(input)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", response.StatusCode)
	}
}
