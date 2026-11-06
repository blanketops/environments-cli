/*
Copyright 2026 The BlanketOps Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// ReportConfig holds configuration for the report server/client
type ReportConfig struct {
	AllureServerURL string
	ProjectName     string
	APIKey          string
}

// ReportClient handles sending test results to Allure server
type ReportClient struct {
	config ReportConfig
	client *http.Client
}

// NewReportClient creates a new report client
func NewReportClient(config ReportConfig) *ReportClient {
	return &ReportClient{
		config: config,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// ReportData represents test/benchmark results
type ReportData struct {
	ProjectName   string                 `json:"project_name"`
	RunID         string                 `json:"run_id"`
	Timestamp     time.Time              `json:"timestamp"`
	TestType      string                 `json:"test_type"` // "test", "benchmark", "scenario"
	Results       []TestResult           `json:"results"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// TestResult represents a single test/benchmark result
type TestResult struct {
	Name        string                 `json:"name"`
	Status      string                 `json:"status"` // "passed", "failed", "skipped"
	Duration    float64                `json:"duration_ms"`
	Package     string                 `json:"package"`
	Output      string                 `json:"output,omitempty"`
	Metrics     map[string]interface{} `json:"metrics,omitempty"` // for benchmarks
	Annotations map[string]string      `json:"annotations,omitempty"`
}

// SendResults sends test results to Allure server
func (c *ReportClient) SendResults(ctx context.Context, data ReportData) error {
	if c.config.AllureServerURL == "" {
		return fmt.Errorf("ALLURE_SERVER_URL not configured")
	}

	// Prepare multipart form with results.json and any attachments
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add results JSON
	resultsJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal results: %w", err)
	}

	part, err := writer.CreateFormFile("results", "results.json")
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(resultsJSON); err != nil {
		return fmt.Errorf("write results: %w", err)
	}

	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST",
		strings.TrimRight(c.config.AllureServerURL, "/")+"/api/v1/results",
		&buf)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error %d: %s", resp.StatusCode, string(body))
	}

	fmt.Println("✅ Results sent to Allure server")
	return nil
}

// ParseGoTestJSON parses `go test -json` output into TestResult slice
func ParseGoTestJSON(r io.Reader) ([]TestResult, error) {
	dec := json.NewDecoder(r)
	var results []TestResult

	for dec.More() {
		var raw map[string]interface{}
		if err := dec.Decode(&raw); err != nil {
			return nil, err
		}

		action, _ := raw["Action"].(string)
		testName, _ := raw["Test"].(string)
		pkg, _ := raw["Package"].(string)
		output, _ := raw["Output"].(string)
		elapsed, _ := raw["Elapsed"].(float64)

		// Only process completed tests
		if action == "pass" || action == "fail" || action == "skip" {
			if testName == "" {
				continue // package-level event
			}

			status := "passed"
			if action == "fail" {
				status = "failed"
			} else if action == "skip" {
				status = "skipped"
			}

			results = append(results, TestResult{
				Name:    testName,
				Status:  status,
				Duration: elapsed * 1000, // convert to ms
				Package: pkg,
				Output:  output,
			})
		}

		// Handle benchmark results
		if action == "bench" {
			metrics := map[string]interface{}{
				"ns_per_op":   raw["NsPerOp"],
				"alloc_per_op": raw["AllocPerOp"],
				"allocs_per_op": raw["AllocsPerOp"],
			}

			results = append(results, TestResult{
				Name:    testName,
				Status:  "benchmarked",
				Duration: elapsed * 1000,
				Package: pkg,
				Metrics: metrics,
				Output:  output,
			})
		}
	}

	return results, nil
}

// ReportCommand returns the cobra command for report operations
func ReportCommand() *cobra.Command {
	var (
		serverURL   string
		projectName string
		apiKey      string
		inputFile   string
		runID       string
		testType    string
	)

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Send test/benchmark results to Allure server",
		Long: `Send test/benchmark results to an Allure server.
		
Reads go test -json output from stdin or file and sends to configured Allure server.

Environment variables:
  ALLURE_SERVER_URL    Base URL of Allure server (e.g. https://allure.blanketops.io)
  ALLURE_PROJECT_NAME  Project name in Allure
  ALLURE_API_KEY       API key for authentication

Examples:
  # Send test results from file
  bops-env report -f test-output.json
  
  # Send benchmark results from stdin
  go test ./... -bench=. -benchmem -json | bops-env report --type benchmark
  
  # Send with custom run ID
  bops-env report -f output.json --run-id "ci-123"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config from env if not provided
			if serverURL == "" {
				serverURL = os.Getenv("ALLURE_SERVER_URL")
			}
			if projectName == "" {
				projectName = os.Getenv("ALLURE_PROJECT_NAME")
			}
			if apiKey == "" {
				apiKey = os.Getenv("ALLURE_API_KEY")
			}
			if runID == "" {
				runID = fmt.Sprintf("run-%d", time.Now().Unix())
			}
			if testType == "" {
				testType = "test"
			}

			if serverURL == "" {
				return fmt.Errorf("ALLURE_SERVER_URL not set (use --server-url or ALLURE_SERVER_URL)")
			}
			if projectName == "" {
				return fmt.Errorf("ALLURE_PROJECT_NAME not set (use --project-name or ALLURE_PROJECT_NAME)")
			}

			var input io.Reader = os.Stdin
			if inputFile != "" {
				f, err := os.Open(inputFile)
				if err != nil {
					return fmt.Errorf("open input file: %w", err)
				}
				defer f.Close()
				input = f
			}

			// Read all input
			data, err := io.ReadAll(input)
			if err != nil {
				return fmt.Errorf("read input: %w", err)
			}

			// Parse go test -json output
			results, err := ParseGoTestJSON(bytes.NewReader(data))
			if err != nil {
				return fmt.Errorf("parse test output: %w", err)
			}

			if len(results) == 0 {
				fmt.Println("No test results found in input")
				return nil
			}

			reportData := ReportData{
				ProjectName: projectName,
				RunID:       runID,
				Timestamp:   time.Now(),
				TestType:    testType,
				Results:     results,
				Metadata: map[string]interface{}{
					"source": "bops-env",
				},
			}

			client := NewReportClient(ReportConfig{
				AllureServerURL: serverURL,
				ProjectName:     projectName,
				APIKey:          apiKey,
			})

			ctx := context.Background()
			if err := client.SendResults(ctx, reportData); err != nil {
				return fmt.Errorf("send results: %w", err)
			}

			fmt.Printf("✅ Sent %d results to Allure server\n", len(results))
			return nil
		},
	}

	cmd.Flags().StringVar(&serverURL, "server-url", "", "Allure server URL (ALLURE_SERVER_URL)")
	cmd.Flags().StringVar(&projectName, "project-name", "", "Allure project name (ALLURE_PROJECT_NAME)")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key for Allure server (ALLURE_API_KEY)")
	cmd.Flags().StringVar(&inputFile, "file", "", "Input file (default: stdin)")
	cmd.Flags().StringVar(&runID, "run-id", "", "Run ID (default: timestamp)")
	cmd.Flags().StringVar(&testType, "type", "test", "Test type: test, benchmark, scenario")

	return cmd
}

// ReportServerCommand returns the cobra command to start a report receiver server
func ReportServerCommand() *cobra.Command {
	var (
		port     int
		allureURL string
	)

	cmd := &cobra.Command{
		Use:   "report-server",
		Short: "Start a report receiver server that forwards to Allure",
		Long: `Start an HTTP server that receives test results and forwards to Allure.
		
Useful for CI systems that can't reach Allure directly but can reach this server.

Environment variables:
  ALLURE_SERVER_URL    Base URL of Allure server
  ALLURE_API_KEY       API key for Allure server`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if allureURL == "" {
				allureURL = os.Getenv("ALLURE_SERVER_URL")
			}
			if allureURL == "" {
				return fmt.Errorf("ALLURE_SERVER_URL not set")
			}

			client := NewReportClient(ReportConfig{
				AllureServerURL: allureURL,
				APIKey:          os.Getenv("ALLURE_API_KEY"),
			})

			mux := http.NewServeMux()
			mux.HandleFunc("/report", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					return
				}

				// Parse multipart form
				err := r.ParseMultipartForm(32 << 20) // 32 MB
				if err != nil {
					http.Error(w, "Failed to parse form", http.StatusBadRequest)
					return
				}

				file, _, err := r.FormFile("results")
				if err != nil {
					http.Error(w, "Missing results file", http.StatusBadRequest)
					return
				}
				defer file.Close()

				data, err := io.ReadAll(file)
				if err != nil {
					http.Error(w, "Failed to read file", http.StatusInternalServerError)
					return
				}

				// Parse and forward
				results, err := ParseGoTestJSON(bytes.NewReader(data))
				if err != nil {
					http.Error(w, "Failed to parse results", http.StatusBadRequest)
					return
				}

				projectName := r.FormValue("project_name")
				if projectName == "" {
					projectName = os.Getenv("ALLURE_PROJECT_NAME")
				}

				reportData := ReportData{
					ProjectName: projectName,
					RunID:       fmt.Sprintf("server-%d", time.Now().Unix()),
					Timestamp:   time.Now(),
					TestType:    r.FormValue("test_type"),
					Results:     results,
				}

				ctx := r.Context()
				if err := client.SendResults(ctx, reportData); err != nil {
					http.Error(w, fmt.Sprintf("Failed to forward: %v", err), http.StatusInternalServerError)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			})

			addr := fmt.Sprintf(":%d", port)
			fmt.Printf("Starting report server on %s, forwarding to %s\n", addr, allureURL)
			return http.ListenAndServe(addr, mux)
		},
	}

	cmd.Flags().IntVar(&port, "port", 8080, "Server port")
	cmd.Flags().StringVar(&allureURL, "allure-url", "", "Allure server URL (ALLURE_SERVER_URL)")

	return cmd
}

// InitReportCommands adds report commands to the root command
func InitReportCommands(root *cobra.Command) {
	root.AddCommand(ReportCommand())
	root.AddCommand(ReportServerCommand())
}