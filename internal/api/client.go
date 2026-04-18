package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const baseURL = "https://api.poe.com/usage"

// AuthError is returned when the API responds with a 401.
type AuthError struct {
	StatusCode int
	Body       string
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("authentication failed (HTTP %d): %s", e.StatusCode, e.Body)
}

// breakdownValue is a points value that can be an int or a string like "25 points (2882 tokens)".
type breakdownValue int

func (b *breakdownValue) UnmarshalJSON(data []byte) error {
	// Try integer first
	var n int
	if err := json.Unmarshal(data, &n); err == nil {
		*b = breakdownValue(n)
		return nil
	}
	// Try string: "25 points (2882 tokens)" or "100 points"
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	// Extract leading integer
	var val int
	fmt.Sscanf(s, "%d", &val)
	*b = breakdownValue(val)
	return nil
}

// CostBreakdown holds the breakdown of points cost.
type CostBreakdown struct {
	Input         breakdownValue `json:"Input"`
	Output        breakdownValue `json:"Output"`
	CacheWrite    breakdownValue `json:"Cache write"`
	CacheDiscount breakdownValue `json:"Cache discount"`
	Total         breakdownValue `json:"Total"`
}

// UsageRecord represents a single usage record from the API.
type UsageRecord struct {
	BotName       string         `json:"bot_name"`
	CreationTime  int64          `json:"creation_time"` // unix microseconds
	QueryID       string         `json:"query_id"`
	CostUSD       string         `json:"cost_usd"`
	CostPoints    int            `json:"cost_points"`
	Breakdown     CostBreakdown  `json:"cost_breakdown_in_points"`
	UsageType     string         `json:"usage_type"`
	ChatName      *string        `json:"chat_name"`
	CanvasTabName *string        `json:"canvas_tab_name"`
}

// Time returns the record's creation time as a UTC time.Time.
func (r *UsageRecord) Time() time.Time {
	return time.UnixMicro(r.CreationTime).UTC()
}

// BalanceResponse is the response from the balance endpoint.
type BalanceResponse struct {
	CurrentPointBalance  int    `json:"current_point_balance"`
	PlanPointsBalance    int    `json:"plan_points_balance"`
	AddonPointBalance    int    `json:"addon_point_balance"`
	TotalBalanceUSD      string `json:"total_balance_usd"`
	NextMonthlyGrantTime int64  `json:"next_monthly_grant_time"` // unix microseconds
	NextMonthlyGrant     int    `json:"next_monthly_grant_amount"`
}

// PeriodEnd returns the date the current billing period ends.
func (b *BalanceResponse) PeriodEnd() time.Time {
	return time.UnixMicro(b.NextMonthlyGrantTime).UTC()
}

// HistoryPage is one page of history from the API.
type HistoryPage struct {
	HasMore bool          `json:"has_more"`
	Length  int           `json:"length"`
	Data    []UsageRecord `json:"data"`
}

// Client is an HTTP client for the Poe usage API.
type Client struct {
	apiKey  string
	http    *http.Client
	verbose bool
	stderr  io.Writer
}

// NewClient creates a new API client.
func NewClient(apiKey string, timeout int, verbose bool, stderr io.Writer) *Client {
	return &Client{
		apiKey: apiKey,
		http: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		verbose: verbose,
		stderr:  stderr,
	}
}

func (c *Client) doGet(ctx context.Context, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	if c.verbose && c.stderr != nil {
		fmt.Fprintf(c.stderr, "%s %s\n", req.Method, req.URL.String())
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if c.verbose && c.stderr != nil {
		fmt.Fprintf(c.stderr, "→ HTTP %d\n", resp.StatusCode)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, &AuthError{StatusCode: resp.StatusCode, Body: string(body)}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// GetBalance fetches the current point balance.
func (c *Client) GetBalance(ctx context.Context) (*BalanceResponse, error) {
	data, err := c.doGet(ctx, baseURL+"/current_balance")
	if err != nil {
		return nil, err
	}
	var resp BalanceResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse balance response: %w", err)
	}
	return &resp, nil
}

// GetHistoryPage fetches a single page of history.
func (c *Client) GetHistoryPage(ctx context.Context, pageSize int, cursor string) (*HistoryPage, error) {
	u, err := url.Parse(baseURL + "/points_history")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("limit", strconv.Itoa(pageSize))
	if cursor != "" {
		q.Set("starting_after", cursor)
	}
	u.RawQuery = q.Encode()

	data, err := c.doGet(ctx, u.String())
	if err != nil {
		return nil, err
	}
	var page HistoryPage
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, fmt.Errorf("failed to parse history response: %w", err)
	}
	return &page, nil
}

// GetAllHistory fetches all history records up to totalLimit (0 = unlimited).
// progressFn is called after each page with the running total count.
func (c *Client) GetAllHistory(ctx context.Context, pageSize int, totalLimit int, progressFn func(int)) ([]UsageRecord, error) {
	var all []UsageRecord
	cursor := ""

	for {
		fetchSize := pageSize
		if totalLimit > 0 {
			remaining := totalLimit - len(all)
			if remaining <= 0 {
				break
			}
			if remaining < fetchSize {
				fetchSize = remaining
			}
		}

		page, err := c.GetHistoryPage(ctx, fetchSize, cursor)
		if err != nil {
			return nil, err
		}

		all = append(all, page.Data...)

		if progressFn != nil {
			progressFn(len(all))
		}

		if !page.HasMore || len(page.Data) == 0 {
			break
		}

		if totalLimit > 0 && len(all) >= totalLimit {
			break
		}

		// Advance cursor using the last record's QueryID
		cursor = page.Data[len(page.Data)-1].QueryID
	}

	return all, nil
}
