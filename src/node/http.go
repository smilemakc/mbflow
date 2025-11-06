package node

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

const HTTPNodeType = "HTTP"

// IRequestBuilder - интерфейс для построения HTTP-запросов

type IRequestBuilder interface {
	BuildRequest() (*http.Request, error)
}

// JSONRequestBuilder - билдер для JSON-запросов

type JSONRequestBuilder struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    string
}

func (b *JSONRequestBuilder) BuildRequest() (*http.Request, error) {
	reader := strings.NewReader(b.Body)
	req, err := http.NewRequest(b.Method, b.URL, reader)
	if err != nil {
		return nil, err
	}
	for key, value := range b.Headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// MultipartRequestBuilder - билдер для multipart-запросов
type FileData struct {
	Filename string
	Reader   io.Reader
}

type MultipartRequestBuilder struct {
	Method  string
	URL     string
	Headers map[string]string
	Fields  map[string]string
	Files   map[string]FileData
}

func (b *MultipartRequestBuilder) BuildRequest() (*http.Request, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Добавляем поля
	for key, value := range b.Fields {
		_ = writer.WriteField(key, value)
	}

	// Добавляем файлы
	fmt.Println(b.Files)
	for fieldName, fileData := range b.Files {
		part, err := writer.CreateFormFile(fieldName, fileData.Filename)
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(part, fileData.Reader)
		if err != nil {
			return nil, err
		}
	}

	writer.Close()

	req, err := http.NewRequest(b.Method, b.URL, &body)
	if err != nil {
		return nil, err
	}
	for key, value := range b.Headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, nil
}

// HTTPRequestData - структура запроса для HTTPNode

type HTTPRequestData struct {
	Builder IRequestBuilder
}

// HTTPResponseData - структура ответа для HTTPNode

type HTTPResponseData struct {
	StatusCode int
	Headers    map[string][]string
	Body       []byte
}

// HTTPNode - выполняет HTTP-запрос

type HTTPNode struct {
	BaseNode[HTTPRequestData, HTTPResponseData]
	Client     *http.Client
	Timeout    time.Duration
	RetryCount int
}

func NewHTTPNode(id, name string, client *http.Client, timeout time.Duration, retryCount int) *HTTPNode {
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	return &HTTPNode{
		BaseNode: BaseNode[HTTPRequestData, HTTPResponseData]{
			ID:   id,
			Type: HTTPNodeType,
			Name: name,
		},
		Client:     client,
		Timeout:    timeout,
		RetryCount: retryCount,
	}
}

func (h *HTTPNode) Execute(inputData HTTPRequestData) (HTTPResponseData, error) {
	req, err := inputData.Builder.BuildRequest()
	if err != nil {
		return HTTPResponseData{}, err
	}

	var resp *http.Response
	for i := 0; i <= h.RetryCount; i++ {
		log.Printf("[HTTPNode] Attempt %d: %s %s", i+1, req.Method, req.URL)
		resp, err = h.Client.Do(req)
		if err == nil {
			break
		}
		log.Printf("[HTTPNode] Error: %v, retrying...", err)
		time.Sleep(time.Second * 2) // задержка перед повторной попыткой
	}

	if err != nil {
		return HTTPResponseData{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return HTTPResponseData{}, err
	}

	log.Printf("[HTTPNode] Response: %d %s", resp.StatusCode, string(body))

	return HTTPResponseData{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       body,
	}, nil
}
