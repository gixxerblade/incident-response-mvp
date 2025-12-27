package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"gorm.io/gorm"
)

// HTTPRequestAction makes generic HTTP requests to any API
type HTTPRequestAction struct {
	db *gorm.DB
}

func (a *HTTPRequestAction) Execute(params map[string]interface{}) (interface{}, error) {
	url := getStringParam(params, "url", "")
	method := getStringParam(params, "method", "GET")
	headers := params["headers"]
	body := params["body"]
	timeout := getIntParam(params, "timeout", 30)

	if url == "" {
		return nil, fmt.Errorf("url parameter is required")
	}

	log.Printf("[ACTION] [HTTP] %s %s", method, url)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	// Prepare request body
	var bodyReader io.Reader
	if body != nil {
		bodyJSON, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyJSON)
	}

	// Create request
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	if headers != nil {
		if headerMap, ok := headers.(map[string]interface{}); ok {
			for k, v := range headerMap {
				req.Header.Set(k, fmt.Sprintf("%v", v))
			}
		}
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse JSON response if possible
	var jsonResp interface{}
	if err := json.Unmarshal(respBody, &jsonResp); err == nil {
		return map[string]interface{}{
			"status_code": resp.StatusCode,
			"headers":     resp.Header,
			"body":        jsonResp,
			"success":     resp.StatusCode >= 200 && resp.StatusCode < 300,
		}, nil
	}

	// Return raw response if not JSON
	return map[string]interface{}{
		"status_code": resp.StatusCode,
		"headers":     resp.Header,
		"body":        string(respBody),
		"success":     resp.StatusCode >= 200 && resp.StatusCode < 300,
	}, nil
}

// ShellScriptAction executes arbitrary shell scripts/commands
type ShellScriptAction struct {
	db *gorm.DB
}

func (a *ShellScriptAction) Execute(params map[string]interface{}) (interface{}, error) {
	script := getStringParam(params, "script", "")
	shell := getStringParam(params, "shell", "/bin/bash")
	timeout := getIntParam(params, "timeout", 300)
	workdir := getStringParam(params, "workdir", "")

	if script == "" {
		return nil, fmt.Errorf("script parameter is required")
	}

	log.Printf("[ACTION] [SHELL] Executing script (timeout: %ds)", timeout)
	log.Printf("[ACTION] [SHELL] Script: %s", script)

	// Create command
	cmd := exec.Command(shell, "-c", script)
	if workdir != "" {
		cmd.Dir = workdir
	}

	// Set timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Wait with timeout
	select {
	case err := <-done:
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				return nil, fmt.Errorf("script execution failed: %w", err)
			}
		}

		return map[string]interface{}{
			"exit_code": exitCode,
			"stdout":    stdout.String(),
			"stderr":    stderr.String(),
			"success":   exitCode == 0,
		}, nil

	case <-time.After(time.Duration(timeout) * time.Second):
		cmd.Process.Kill()
		return nil, fmt.Errorf("script execution timed out after %d seconds", timeout)
	}
}

// WebhookAction sends data to any webhook URL
type WebhookAction struct {
	db *gorm.DB
}

func (a *WebhookAction) Execute(params map[string]interface{}) (interface{}, error) {
	url := getStringParam(params, "url", "")
	payload := params["payload"]
	method := getStringParam(params, "method", "POST")
	headers := params["headers"]

	if url == "" {
		return nil, fmt.Errorf("url parameter is required")
	}

	log.Printf("[ACTION] [WEBHOOK] Sending to %s", url)

	// Default payload structure
	if payload == nil {
		payload = map[string]interface{}{
			"event":     "incident",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
	}

	// Marshal payload
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create request
	req, err := http.NewRequest(method, url, bytes.NewReader(payloadJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default Content-Type
	req.Header.Set("Content-Type", "application/json")

	// Add custom headers
	if headers != nil {
		if headerMap, ok := headers.(map[string]interface{}); ok {
			for k, v := range headerMap {
				req.Header.Set(k, fmt.Sprintf("%v", v))
			}
		}
	}

	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return map[string]interface{}{
		"status_code": resp.StatusCode,
		"response":    string(respBody),
		"success":     resp.StatusCode >= 200 && resp.StatusCode < 300,
	}, nil
}

// PythonScriptAction executes Python scripts (useful for complex integrations)
type PythonScriptAction struct {
	db *gorm.DB
}

func (a *PythonScriptAction) Execute(params map[string]interface{}) (interface{}, error) {
	script := getStringParam(params, "script", "")
	pythonPath := getStringParam(params, "python", "python3")
	args := params["args"]

	if script == "" {
		return nil, fmt.Errorf("script parameter is required")
	}

	// Build command
	cmdArgs := []string{script}
	if args != nil {
		if argList, ok := args.([]interface{}); ok {
			for _, arg := range argList {
				cmdArgs = append(cmdArgs, fmt.Sprintf("%v", arg))
			}
		}
	}

	log.Printf("[ACTION] [PYTHON] Executing: %s %s", pythonPath, strings.Join(cmdArgs, " "))

	// Execute Python script
	cmd := exec.Command(pythonPath, cmdArgs...)
	output, err := cmd.CombinedOutput()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	return map[string]interface{}{
		"exit_code": exitCode,
		"output":    string(output),
		"success":   exitCode == 0,
	}, nil
}
