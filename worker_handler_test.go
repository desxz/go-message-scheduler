package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
)

func TestWorkerPoolHandler_ControlWorkerPool(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	app := fiber.New()

	mockWorkerPool := NewMockWorkerPool(ctrl)
	handler := NewWorkerPoolHandler(mockWorkerPool)
	handler.RegisterRoutes(app)

	statePath := "/worker-pool/state"

	tests := []struct {
		name        string
		url         string
		requestBody interface{}
		wantStatus  int
		wantBody    string
		beforeSuite func()
	}{
		{
			name:        "should start worker pool with status 200",
			url:         statePath,
			requestBody: WorkerPoolActionRequest{Action: "start"},
			wantStatus:  fiber.StatusOK,
			wantBody:    `{"status":"running"}`,
			beforeSuite: func() {
				mockWorkerPool.EXPECT().ResumeFetching()
				mockWorkerPool.EXPECT().GetStatus().Return(StatusRunning)
			},
		},
		{
			name:        "should pause worker pool with status 200",
			url:         statePath,
			requestBody: WorkerPoolActionRequest{Action: "pause"},
			wantStatus:  fiber.StatusOK,
			wantBody:    `{"status":"paused"}`,
			beforeSuite: func() {
				mockWorkerPool.EXPECT().PauseFetching()
				mockWorkerPool.EXPECT().GetStatus().Return(StatusPaused)
			},
		},
		{
			name:        "should return error with status 400 for invalid action",
			url:         statePath,
			requestBody: WorkerPoolActionRequest{Action: "invalid"},
			wantStatus:  fiber.StatusBadRequest,
			wantBody:    `{"error":"Invalid action. Use 'start' or 'pause'"}`,
			beforeSuite: func() {
			},
		},
		{
			name:        "should return error with status 400 for invalid request body",
			url:         statePath,
			requestBody: "invalid json",
			wantStatus:  fiber.StatusBadRequest,
			wantBody:    `{"error":"Invalid request body"}`,
			beforeSuite: func() {
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.beforeSuite()

			var reqBody io.Reader
			if tt.requestBody != nil {
				if s, ok := tt.requestBody.(string); ok {
					reqBody = bytes.NewBufferString(s)
				} else {
					jsonBody, err := json.Marshal(tt.requestBody)
					assert.NoError(t, err)
					reqBody = bytes.NewBuffer(jsonBody)
				}
			}

			req := httptest.NewRequest(fiber.MethodPut, tt.url, reqBody)
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req, -1)
			defer resp.Body.Close()

			assert.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)

			if tt.wantBody != "" {
				bodyBytes, _ := io.ReadAll(resp.Body)
				assert.JSONEq(t, tt.wantBody, string(bodyBytes))
			}
		})
	}
}
