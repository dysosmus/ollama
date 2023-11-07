package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jmorganca/ollama/api"
	"io/fs"
	"time"
)

func ptr[T any](t T) *T {
	return &t
}

func LambdaGenerateHandler(ctx context.Context, req api.GenerateRequest) (*string, error) {
	loaded.mu.Lock()
	defer loaded.mu.Unlock()

	checkpointStart := time.Now()

	model, err := GetModel(req.Model)
	if err != nil {
		var pErr *fs.PathError
		if errors.As(err, &pErr) {
			return ptr(""), fmt.Errorf("model '%s' not found, try pulling it first", req.Model)
		}
		return ptr(""), err
	}

	workDir := "/root/.ollama/models/"

	// TODO: set this duration from the request if specified
	sessionDuration := defaultSessionDuration
	if err := load(ctx, workDir, model, req.Options, sessionDuration); err != nil {
		return ptr(""), err
	}

	checkpointLoaded := time.Now()

	prompt, err := model.Prompt(req)
	if err != nil {
		return ptr(""), err
	}

	ch := make(chan any)
	go func() {
		defer close(ch)
		// an empty request loads the model
		if req.Prompt == "" && req.Template == "" && req.System == "" {
			ch <- api.GenerateResponse{CreatedAt: time.Now().UTC(), Model: req.Model, Done: true}
			return
		}

		fn := func(r api.GenerateResponse) {
			loaded.expireAt = time.Now().Add(sessionDuration)
			loaded.expireTimer.Reset(sessionDuration)

			r.Model = req.Model
			r.CreatedAt = time.Now().UTC()
			if r.Done {
				r.TotalDuration = time.Since(checkpointStart)
				r.LoadDuration = checkpointLoaded.Sub(checkpointStart)
			}

			ch <- r
		}

		if err := loaded.runner.Predict(ctx, req.Context, prompt, fn); err != nil {
			ch <- gin.H{"error": err.Error()}
		}
	}()

	if req.Stream != nil && *req.Stream {
		return ptr(""), fmt.Errorf("streaming not supported")
	}

	var response api.GenerateResponse
	generated := ""
	for resp := range ch {
		if r, ok := resp.(api.GenerateResponse); ok {
			generated += r.Response
			response = r
		} else {
			return ptr(""), err
		}
	}
	response.Response = generated
	return ptr(generated), nil
}