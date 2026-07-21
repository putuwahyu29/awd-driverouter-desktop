package provider

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

type OAuthResult struct {
	Code  string
	Error error
}

// StartOAuthListener starts a temporary HTTP server on port 5998 and waits for the authorization code.
func StartOAuthListener(expectedState string) (string, error) {
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    "127.0.0.1:5998",
		Handler: mux,
	}
	listener, listenErr := net.Listen("tcp", server.Addr)
	if listenErr != nil {
		return "", fmt.Errorf("oauth callback port busy: %w", listenErr)
	}
	defer listener.Close()

	resultChan := make(chan OAuthResult, 1)

	mux.HandleFunc("/oauth/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		if state != expectedState {
			resultChan <- OAuthResult{Error: fmt.Errorf("state mismatch: expected %s, got %s", expectedState, state)}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`
				<html>
				<body style="font-family: Arial, sans-serif; text-align: center; margin-top: 100px; background-color: #fce8e6; color: #c5221f;">
					<h1>Authentication Failed</h1>
					<p>Invalid state token. Please close this window and try again.</p>
				</body>
				</html>
			`))
			return
		}

		if code == "" {
			errStr := r.URL.Query().Get("error_description")
			if errStr == "" {
				errStr = r.URL.Query().Get("error")
			}
			resultChan <- OAuthResult{Error: fmt.Errorf("oauth error: %s", errStr)}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf(`
				<html>
				<body style="font-family: Arial, sans-serif; text-align: center; margin-top: 100px; background-color: #fce8e6; color: #c5221f;">
					<h1>Authentication Failed</h1>
					<p>%s</p>
				</body>
				</html>
			`, errStr)))
			return
		}

		resultChan <- OAuthResult{Code: code}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			<html>
			<body style="font-family: Arial, sans-serif; text-align: center; margin-top: 100px; background-color: #e6f4ea; color: #137333;">
				<div style="border: 1px solid #ceead6; background-color: #f1f8f5; border-radius: 8px; display: inline-block; padding: 40px 60px;">
					<h1>Awd DriveRouter Connected Successfully!</h1>
					<p style="font-size: 16px; color: #5f6368;">You have successfully authenticated your cloud account.</p>
					<p style="font-size: 14px; font-weight: bold; color: #1e8e3e; margin-top: 20px;">You can now close this browser window and return to the application.</p>
				</div>
			</body>
			</html>
		`))
	})

	// Run server in background
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			resultChan <- OAuthResult{Error: fmt.Errorf("local server failed: %w", err)}
		}
	}()

	// Wait for code or timeout of 5 minutes
	var code string
	var err error
	select {
	case res := <-resultChan:
		code = res.Code
		err = res.Error
	case <-time.After(5 * time.Minute):
		err = fmt.Errorf("oauth callback timed out after 5 minutes")
	}

	// Shutdown the server cleanly
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)

	return code, err
}
