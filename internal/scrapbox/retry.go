package scrapbox

import (
	"context"
	"errors"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

// RetryPolicy は一時的な失敗に対する再試行の方針である。
// 対象はネットワークエラーと 429 / 502 / 503 / 504 である。GET のみを送るため安全に再送できる。
type RetryPolicy struct {
	// MaxRetries は最初の試行に加えて再試行する最大回数である。0 なら再試行しない。
	MaxRetries int
	// BaseDelay は指数バックオフの初期待機時間である。
	BaseDelay time.Duration
	// MaxDelay は 1 回あたりの待機時間の上限である。Retry-After ヘッダの値にも適用する。
	MaxDelay time.Duration
}

// DefaultRetryPolicy は Client のデフォルトの再試行方針である。
var DefaultRetryPolicy = RetryPolicy{
	MaxRetries: 3,
	BaseDelay:  500 * time.Millisecond,
	MaxDelay:   10 * time.Second,
}

// WithRetryPolicy は再試行の方針を変更する。
func WithRetryPolicy(p RetryPolicy) Option {
	return func(c *Client) error {
		if p.MaxRetries < 0 {
			return errors.New("retry: MaxRetries must not be negative")
		}
		c.retry = p
		return nil
	}
}

// retryable はレスポンスのステータスコードが再試行の対象かを返す。
func retryable(status int) bool {
	switch status {
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	return false
}

// delay は attempt 回目 (0 始まり) の再試行までの待機時間を返す。
// Retry-After ヘッダがあればそれに従い、なければ full jitter 付きの指数バックオフにする。
func (p RetryPolicy) delay(attempt int, resp *http.Response) time.Duration {
	if resp != nil {
		if d, ok := parseRetryAfter(resp.Header.Get("Retry-After")); ok {
			return min(d, p.MaxDelay)
		}
	}
	backoff := p.BaseDelay << attempt
	if backoff <= 0 || backoff > p.MaxDelay { // シフトのオーバーフローも上限に丸める
		backoff = p.MaxDelay
	}
	if backoff <= 0 {
		return 0
	}
	return rand.N(backoff + 1)
}

// parseRetryAfter は Retry-After ヘッダ (秒数 または HTTP 日付) を解釈する。
func parseRetryAfter(v string) (time.Duration, bool) {
	if v == "" {
		return 0, false
	}
	if sec, err := strconv.Atoi(v); err == nil && sec >= 0 {
		return time.Duration(sec) * time.Second, true
	}
	if t, err := http.ParseTime(v); err == nil {
		return max(time.Until(t), 0), true
	}
	return 0, false
}

// sleep は d だけ待つ。ctx がキャンセルされたら即座に ctx のエラーを返す。
func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// discard は再試行前にレスポンスボディを読み捨てて閉じ、コネクションを再利用できるようにする。
func discard(resp *http.Response) {
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxErrorBody))
	_ = resp.Body.Close()
}
