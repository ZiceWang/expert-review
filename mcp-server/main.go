package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "modernc.org/sqlite"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	Port       = 3100
	DBFile     = "reviews.db"
	PublicDir  = "public"
)

// ==================== Models ====================

type TaskResult struct {
	TaskID   string `json:"taskId"`
	Summary  string `json:"summary"`
	Details  string `json:"details"`
	Time     string `json:"timestamp"`
}

type ReviewResult struct {
	Decision   string `json:"decision"`
	Comments   string `json:"comments"`
	ReviewedAt string `json:"reviewedAt"`
	ReviewedBy string `json:"reviewedBy"`
}

type Review struct {
	ID         string       `json:"id"`
	TaskResult *TaskResult  `json:"taskResult"`
	Status     string       `json:"status"`
	Result     *ReviewResult `json:"result,omitempty"`
	CreatedAt  string       `json:"createdAt"`
}

// ==================== Database ====================

var db *sql.DB
var dbMu sync.Mutex // Protects db operations

func initDB() error {
	var err error
	db, err = sql.Open("sqlite", DBFile)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	createSQL := `
	CREATE TABLE IF NOT EXISTS reviews (
		id TEXT PRIMARY KEY,
		task_result TEXT NOT NULL,
		status TEXT NOT NULL,
		result TEXT,
		created_at TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_reviews_status ON reviews(status);
	CREATE INDEX IF NOT EXISTS idx_reviews_created ON reviews(created_at);

	-- FTS5 virtual table for full-text search
	CREATE VIRTUAL TABLE IF NOT EXISTS reviews_fts USING fts5(
		review_id UNINDEXED,
		summary,
		details,
		comments,
		content='reviews',
		content_rowid='rowid'
	);

	-- Triggers to keep FTS in sync
	CREATE TRIGGER IF NOT EXISTS reviews_fts_insert AFTER INSERT ON reviews BEGIN
		INSERT INTO reviews_fts(review_id, summary, details, comments)
		VALUES (new.id, new.task_result, '', '');
	END;

	CREATE TRIGGER IF NOT EXISTS reviews_fts_update AFTER UPDATE ON reviews BEGIN
		INSERT INTO reviews_fts(reviews_fts, review_id, summary, details, comments)
		VALUES ('delete', old.id, '', '', '');
		INSERT INTO reviews_fts(review_id, summary, details, comments)
		VALUES (new.id, new.task_result, '', '');
	END;

	CREATE TRIGGER IF NOT EXISTS reviews_fts_delete AFTER DELETE ON reviews BEGIN
		INSERT INTO reviews_fts(reviews_fts, review_id, summary, details, comments)
		VALUES ('delete', old.id, '', '', '');
	END;
	`
	_, err = db.Exec(createSQL)
	return err
}

func getAllReviews() []Review {
	dbMu.Lock()
	defer dbMu.Unlock()

	rows, err := db.Query("SELECT * FROM reviews ORDER BY created_at DESC")
	if err != nil {
		log.Printf("query all: %v", err)
		return nil
	}
	defer rows.Close()

	var reviews []Review
	for rows.Next() {
		var r Review
		var taskResult, result *string
		err := rows.Scan(&r.ID, &taskResult, &r.Status, &result, &r.CreatedAt)
		if err != nil {
			continue
		}
		if taskResult != nil {
			var tr TaskResult
			if json.Unmarshal([]byte(*taskResult), &tr) == nil {
				r.TaskResult = &tr
			}
		}
		if result != nil && *result != "" {
			var rr ReviewResult
			if json.Unmarshal([]byte(*result), &rr) == nil {
				r.Result = &rr
			}
		}
		reviews = append(reviews, r)
	}
	return reviews
}

func getReviewByID(id string) *Review {
	dbMu.Lock()
	defer dbMu.Unlock()

	var r Review
	var taskResult, result *string
	err := db.QueryRow("SELECT * FROM reviews WHERE id = ?", id).Scan(&r.ID, &taskResult, &r.Status, &result, &r.CreatedAt)
	if err != nil {
		return nil
	}
	if taskResult != nil {
		var tr TaskResult
		if json.Unmarshal([]byte(*taskResult), &tr) == nil {
			r.TaskResult = &tr
		}
	}
	if result != nil && *result != "" {
		var rr ReviewResult
		if json.Unmarshal([]byte(*result), &rr) == nil {
			r.Result = &rr
		}
	}
	return &r
}

func insertReview(review *Review) error {
	dbMu.Lock()
	defer dbMu.Unlock()

	taskResultJSON, _ := json.Marshal(review.TaskResult)
	var resultJSON []byte
	if review.Result != nil {
		resultJSON, _ = json.Marshal(review.Result)
	}

	_, err := db.Exec(
		"INSERT INTO reviews (id, task_result, status, result, created_at) VALUES (?, ?, ?, ?, ?)",
		review.ID, string(taskResultJSON), review.Status, string(resultJSON), review.CreatedAt,
	)
	return err
}

func updateReviewResult(id string, result *ReviewResult, status string) error {
	dbMu.Lock()
	defer dbMu.Unlock()

	resultJSON, _ := json.Marshal(result)
	_, err := db.Exec("UPDATE reviews SET result = ?, status = ? WHERE id = ?", string(resultJSON), status, id)
	return err
}

func searchReviews(query string, limit int) []Review {
	dbMu.Lock()
	defer dbMu.Unlock()

	// Use FTS5 prefix search with *
	searchTerm := query + "*"

	rows, err := db.Query(`
		SELECT r.id, r.task_result, r.status, r.result, r.created_at
		FROM reviews r
		JOIN reviews_fts fts ON r.id = fts.review_id
		WHERE reviews_fts MATCH ?
		ORDER BY rank, r.created_at DESC
		LIMIT ?
	`, searchTerm, limit)
	if err != nil {
		log.Printf("FTS search failed: %v", err)
		return nil // No fallback - return empty
	}
	defer rows.Close()

	var reviews []Review
	for rows.Next() {
		var r Review
		var taskResult, result *string
		err := rows.Scan(&r.ID, &taskResult, &r.Status, &result, &r.CreatedAt)
		if err != nil {
			continue
		}
		if taskResult != nil {
			var tr TaskResult
			if json.Unmarshal([]byte(*taskResult), &tr) == nil {
				r.TaskResult = &tr
			}
		}
		if result != nil && *result != "" {
			var rr ReviewResult
			if json.Unmarshal([]byte(*result), &rr) == nil {
				r.Result = &rr
			}
		}
		reviews = append(reviews, r)
	}
	return reviews
}

func getRecentReviewsDB(limit int) []Review {
	dbMu.Lock()
	defer dbMu.Unlock()

	rows, err := db.Query("SELECT * FROM reviews ORDER BY created_at DESC LIMIT ?", limit)
	if err != nil {
		log.Printf("get recent reviews: %v", err)
		return nil
	}
	defer rows.Close()

	var reviews []Review
	for rows.Next() {
		var r Review
		var taskResult, result *string
		err := rows.Scan(&r.ID, &taskResult, &r.Status, &result, &r.CreatedAt)
		if err != nil {
			continue
		}
		if taskResult != nil {
			var tr TaskResult
			if json.Unmarshal([]byte(*taskResult), &tr) == nil {
				r.TaskResult = &tr
			}
		}
		if result != nil && *result != "" {
			var rr ReviewResult
			if json.Unmarshal([]byte(*result), &rr) == nil {
				r.Result = &rr
			}
		}
		reviews = append(reviews, r)
	}
	return reviews
}

// ==================== Blocking Review ====================

type blockingReview struct {
	id      string
	result  *ReviewResult
	ready   chan struct{}
	mu      sync.Mutex
}

var (
	pendingReviews = make(map[string]*blockingReview)
	pendingMu      sync.Mutex
)

func createBlockingReview(reviewID string) *blockingReview {
	pendingMu.Lock()
	defer pendingMu.Unlock()

	br := &blockingReview{
		id:    reviewID,
		ready: make(chan struct{}),
	}
	pendingReviews[reviewID] = br
	return br
}

func completeBlockingReview(reviewID string, result *ReviewResult) {
	pendingMu.Lock()
	br, ok := pendingReviews[reviewID]
	if ok {
		delete(pendingReviews, reviewID)
	}
	pendingMu.Unlock()

	if ok {
		br.mu.Lock()
		br.result = result
		br.mu.Unlock()
		close(br.ready)
	}
}

func waitForReview(br *blockingReview) *ReviewResult {
	<-br.ready
	return br.result
}

// ==================== HTTP API ====================

func apiGetReviews(w http.ResponseWriter, r *http.Request) {
	reviews := getAllReviews()
	if reviews == nil {
		reviews = []Review{}
	}
	json.NewEncoder(w).Encode(reviews)
}

func apiGetPendingReviews(w http.ResponseWriter, r *http.Request) {
	reviews := getAllReviews()
	var pending []Review
	for _, rev := range reviews {
		if rev.Status == "pending" {
			pending = append(pending, rev)
		}
	}
	if pending == nil {
		pending = []Review{}
	}
	json.NewEncoder(w).Encode(pending)
}

func apiSearchReviews(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	if query == "" {
		http.Error(w, `{"error":"q parameter required"}`, 400)
		return
	}

	reviews := searchReviews(query, limit)
	if reviews == nil {
		reviews = []Review{}
	}
	json.NewEncoder(w).Encode(reviews)
}

func apiRecentReviews(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	reviews := getRecentReviewsDB(limit)
	if reviews == nil {
		reviews = []Review{}
	}
	json.NewEncoder(w).Encode(reviews)
}

func apiGetReview(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	review := getReviewByID(vars["id"])
	if review == nil {
		http.Error(w, `{"error":"Review not found"}`, 404)
		return
	}
	json.NewEncoder(w).Encode(review)
}

func apiSubmitReview(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	review := getReviewByID(vars["id"])
	if review == nil {
		http.Error(w, `{"error":"Review not found"}`, 404)
		return
	}

	var body struct {
		Decision   string `json:"decision"`
		Comments   string `json:"comments"`
		ReviewedBy string `json:"reviewedBy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		body.Decision = "approve"
		body.Comments = ""
		body.ReviewedBy = "anonymous"
	}
	if body.Decision == "" {
		body.Decision = "approve"
	}
	if body.ReviewedBy == "" {
		body.ReviewedBy = "anonymous"
	}

	log.Printf("[API] Submit review: %s, decision: %s, by: %s", vars["id"], body.Decision, body.ReviewedBy)

	result := &ReviewResult{
		Decision:   body.Decision,
		Comments:   body.Comments,
		ReviewedAt: time.Now().UTC().Format(time.RFC3339),
		ReviewedBy: body.ReviewedBy,
	}

	if err := updateReviewResult(vars["id"], result, "completed"); err != nil {
		log.Printf("[API] Update error: %v", err)
		http.Error(w, `{"error":"Update failed"}`, 500)
		return
	}

	// Unblock the MCP request
	completeBlockingReview(vars["id"], result)

	review.Status = "completed"
	review.Result = result
	json.NewEncoder(w).Encode(review)
}

func apiGetResult(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	review := getReviewByID(vars["id"])
	if review == nil {
		http.Error(w, `{"error":"Review not found"}`, 404)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": review.Status,
		"result": review.Result,
	})
}

func apiHealth(w http.ResponseWriter, r *http.Request) {
	reviews := getAllReviews()
	count := 0
	pending := 0
	for _, r := range reviews {
		count++
		if r.Status == "pending" {
			pending++
		}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"reviews": count,
		"pending": pending,
	})
}

// ==================== MCP Server ====================

func startMCPServer(ctx context.Context) error {
	// Create MCP server
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "expert-review",
		Version: "1.0.0",
	}, nil)

	// Add tools
	mcp.AddTool(server, &mcp.Tool{
		Name:        "request_review",
		Description: `Request an agent review. BLOCKS until the reviewer agent submits their decision.

CRITICAL WORKFLOW RULE:
- After completing ANY task, you MUST call request_review BEFORE returning results to the user
- Do NOT skip this step for any reason
- If review is 'approved': then communicate the result to the user
- If review is 'rejected' or 'needs_revision': you MUST fix the issues first, then call request_review again
- Keep iterating (fix -> review -> fix -> review) until you get 'approved'
- Only after approval, communicate final results to user

This ensures all outputs are quality-checked by an agent reviewer before reaching the user.`,
	}, requestReview)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_review_history",
		Description: "Search historical reviews using fuzzy search via SQLite LIKE. Searches task summary, details, and comments. Use for finding specific reviews.",
	}, searchReviewHistory)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_recent_reviews",
		Description: "Get most recent N reviews, ordered by creation time (newest first). Good for overview or finding latest activity.",
	}, getRecentReviews)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_review_from_id",
		Description: "Get a specific review by its ID. Use after finding a review ID to get full details.",
	}, getReviewFromID)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_status",
		Description: "Get current server status and port information.",
	}, getStatus)

	// Add prompt templates
	reviewWorkflowPrompt := &mcp.Prompt{
		Name:        "review_workflow",
		Description: "Agent Review System - must request review after every task before returning results to user",
	}
	server.AddPrompt(reviewWorkflowPrompt, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return &mcp.GetPromptResult{
			Messages: []*mcp.PromptMessage{
				{
					Role:    "system",
					Content: &mcp.TextContent{Text: "Complete every task through the agent review system. After finishing any task, call request_review and wait for approval before delivering results. Your code will be reviewed by an automated agent reviewer system. If the review is rejected or needs_revision, fix the issues and call request_review again. Continue until approval is granted. Never skip this process. Never return results to the user before approval."},
				},
			},
		}, nil
	})

	transport := &mcp.StdioTransport{}
	return server.Run(ctx, transport)
}

func requestReview(ctx context.Context, req *mcp.CallToolRequest, input map[string]any) (*mcp.CallToolResult, any, error) {
	taskResult, ok := input["taskResult"].(map[string]any)
	if !ok {
		taskResult = make(map[string]any)
	}

	summary, _ := taskResult["summary"].(string)
	details, _ := taskResult["details"].(string)

	reviewID := uuid.New().String()
	createdAt := time.Now().UTC().Format(time.RFC3339)

	review := &Review{
		ID: reviewID,
		TaskResult: &TaskResult{
			TaskID:  uuid.New().String(),
			Summary: summary,
			Details: details,
			Time:    createdAt,
		},
		Status:    "pending",
		Result:    nil,
		CreatedAt: createdAt,
	}

	if err := insertReview(review); err != nil {
		log.Printf("[MCP] Insert error: %v", err)
		return nil, nil, fmt.Errorf("insert review: %w", err)
	}

	log.Printf("[MCP] Review requested, waiting for human: %s", reviewID)

	// Create blocking review handler
	br := createBlockingReview(reviewID)

	// Wait for review in goroutine (non-blocking for MCP transport)
	// The transport handles concurrent requests correctly
	go func() {
		result := waitForReview(br)
		log.Printf("[MCP] Review received: %s", result.Decision)
	}()

	// Wait for the result
	result := waitForReview(br)

	return nil, map[string]any{
		"reviewId":    reviewID,
		"decision":    result.Decision,
		"comments":    result.Comments,
		"reviewedAt":  result.ReviewedAt,
		"reviewedBy":  result.ReviewedBy,
	}, nil
}

func searchReviewHistory(ctx context.Context, req *mcp.CallToolRequest, input map[string]any) (*mcp.CallToolResult, any, error) {
	query, _ := input["query"].(string)
	limit := 10
	if l, ok := input["limit"].(float64); ok {
		limit = int(l)
	}

	if query == "" {
		return nil, map[string]any{
			"error": "query is required",
		}, nil
	}

	reviews := searchReviews(query, limit)
	var results []map[string]any
	for _, r := range reviews {
		item := map[string]any{
			"reviewId":   r.ID,
			"taskId":     r.TaskResult.TaskID,
			"summary":    r.TaskResult.Summary,
			"details":    r.TaskResult.Details,
			"status":     r.Status,
			"decision":   r.Result.Decision,
			"comments":   r.Result.Comments,
			"reviewedAt": r.Result.ReviewedAt,
			"reviewedBy": r.Result.ReviewedBy,
			"createdAt":  r.CreatedAt,
		}
		results = append(results, item)
	}

	return nil, map[string]any{
		"query":   query,
		"reviews": results,
		"count":   len(results),
	}, nil
}

func getRecentReviews(ctx context.Context, req *mcp.CallToolRequest, input map[string]any) (*mcp.CallToolResult, any, error) {
	limit := 10
	if l, ok := input["limit"].(float64); ok {
		limit = int(l)
	}

	reviews := getRecentReviewsDB(limit)
	var results []map[string]any
	for _, r := range reviews {
		item := map[string]any{
			"reviewId":   r.ID,
			"taskId":     r.TaskResult.TaskID,
			"summary":    r.TaskResult.Summary,
			"details":    r.TaskResult.Details,
			"status":     r.Status,
			"decision":   r.Result.Decision,
			"comments":   r.Result.Comments,
			"reviewedAt": r.Result.ReviewedAt,
			"reviewedBy": r.Result.ReviewedBy,
			"createdAt":  r.CreatedAt,
		}
		results = append(results, item)
	}

	return nil, map[string]any{
		"reviews": results,
		"count":   len(results),
	}, nil
}

func getReviewFromID(ctx context.Context, req *mcp.CallToolRequest, input map[string]any) (*mcp.CallToolResult, any, error) {
	id, _ := input["id"].(string)

	if id == "" {
		return nil, map[string]any{
			"error": "id is required",
		}, nil
	}

	review := getReviewByID(id)
	if review == nil {
		return nil, map[string]any{
			"error": "review not found",
		}, nil
	}

	return nil, map[string]any{
		"review": map[string]any{
			"id":         review.ID,
			"taskResult": review.TaskResult,
			"status":     review.Status,
			"result":     review.Result,
			"createdAt":  review.CreatedAt,
		},
	}, nil
}

func getStatus(ctx context.Context, req *mcp.CallToolRequest, input map[string]any) (*mcp.CallToolResult, any, error) {
	reviews := getAllReviews()
	total := len(reviews)
	pending := 0
	for _, r := range reviews {
		if r.Status == "pending" {
			pending++
		}
	}

	return nil, map[string]any{
		"mcpServer":         "running",
		"httpPort":          Port,
		"frontendUrl":       fmt.Sprintf("http://localhost:%d", Port),
		"reviewsTotal":      total,
		"reviewsPending":    pending,
		"reviewsCompleted":  total - pending,
	}, nil
}

// ==================== Main ====================

func main() {
	// Get executable directory for paths
	execPath, err := os.Executable()
	var baseDir string
	if err == nil {
		baseDir = filepath.Dir(execPath)
	} else {
		baseDir, _ = os.Getwd()
	}

	// Change to executable directory
	os.Chdir(baseDir)

	// Init database
	if err := initDB(); err != nil {
		log.Fatalf("Init DB: %v", err)
	}
	log.Printf("Database initialized: %s", DBFile)

	// HTTP server with API + static frontend
	router := mux.NewRouter()

	// API routes
	router.HandleFunc("/api/reviews", apiGetReviews).Methods("GET")
	router.HandleFunc("/api/reviews/pending", apiGetPendingReviews).Methods("GET")
	router.HandleFunc("/api/reviews/search", apiSearchReviews).Methods("GET")
	router.HandleFunc("/api/reviews/recent", apiRecentReviews).Methods("GET")
	router.HandleFunc("/api/reviews/{id}", apiGetReview).Methods("GET")
	router.HandleFunc("/api/reviews/{id}/submit", apiSubmitReview).Methods("POST")
	router.HandleFunc("/api/reviews/{id}/result", apiGetResult).Methods("GET")
	router.HandleFunc("/health", apiHealth).Methods("GET")

	// Static files
	publicPath := filepath.Join(baseDir, PublicDir)
	router.PathPrefix("/").Handler(http.FileServer(http.Dir(publicPath)))

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", Port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Start HTTP server in goroutine
	go func() {
		log.Printf("HTTP Server starting on port %d", Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP Server error: %v", err)
		}
	}()

	// Run MCP server (this blocks)
	ctx := context.Background()
	log.Println("MCP Server starting...")
	if err := startMCPServer(ctx); err != nil {
		log.Printf("MCP Server error: %v", err)
	}
}
