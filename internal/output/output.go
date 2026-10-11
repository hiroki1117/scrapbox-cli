// Package output はコマンド結果の出力形式 (text / json) を扱う。
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
	"time"
)

// Format は出力形式である。
type Format string

const (
	// Text は人間向けの整形済みテキストである。
	Text Format = "text"
	// JSON は API のレスポンス構造を保った JSON である。スクリプトや jq との連携に使う。
	JSON Format = "json"
)

var formats = []Format{Text, JSON}

// ParseFormat は文字列を Format に変換する。
func ParseFormat(s string) (Format, error) {
	for _, f := range formats {
		if Format(s) == f {
			return f, nil
		}
	}
	return "", fmt.Errorf("unknown output format %q (available: %v)", s, formats)
}

// Printer は Format に応じて値を出力する。
type Printer struct {
	w      io.Writer
	format Format
}

// NewPrinter は Printer を生成する。
func NewPrinter(w io.Writer, format Format) *Printer {
	return &Printer{w: w, format: format}
}

// Print は JSON 形式なら v を JSON で、Text 形式なら text 関数で出力する。
func (p *Printer) Print(v any, text func(w io.Writer) error) error {
	if p.format == JSON {
		enc := json.NewEncoder(p.w)
		enc.SetIndent("", "  ")
		enc.SetEscapeHTML(false)
		return enc.Encode(v)
	}
	return text(p.w)
}

// Table はタブ区切りの列を揃えて出力する tabwriter を返す。呼び出し側は Flush する必要がある。
func Table(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
}

// Time は時刻を一覧表示向けのローカル時刻文字列にする。ゼロ値は "-" とする。
func Time(t time.Time) string {
	if t.IsZero() || t.Unix() == 0 {
		return "-"
	}
	return t.Local().Format("2006-01-02 15:04")
}
