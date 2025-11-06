package node

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"mbflow/internal/db"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateTemplateHandler(t *testing.T) {
	// test cases
	tests := []struct {
		input          CreateTemplateRequest
		expectedStatus int
	}{
		{
			// valid input
			input: CreateTemplateRequest{
				Type:        "type1",
				Name:        "name1",
				Description: "desc1",
				Parameters:  map[string]interface{}{"Name": "param1", "Value": "value1"},
			},
			expectedStatus: http.StatusCreated,
		},
		{
			// invalid input
			input:          CreateTemplateRequest{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	// simulate db connection
	dbConn := db.DB()

	for _, tc := range tests {
		requestBody, _ := json.Marshal(tc.input)
		req, _ := http.NewRequest("POST", "/template", bytes.NewBuffer(requestBody))
		resp := httptest.NewRecorder()

		router := gin.Default()
		router.POST("/template", CreateTemplateHandler)

		router.ServeHTTP(resp, req)

		if resp.Code != tc.expectedStatus {
			t.Errorf("Expected response code to be %v, but got %v", tc.expectedStatus, resp.Code)
		}

		// We can also check if template entry is created in db for proper cases
		if tc.expectedStatus == http.StatusCreated {
			var template Template
			dbConn.NewSelect().Model(&template).Where("name = ?", tc.input.Name).Exec(context.Background())
			if template.Name != tc.input.Name {
				t.Errorf("Expected template %s to be created", tc.input.Name)
			}
		}
	}
}
