package scrapbox

import "time"

// Timestamp は API が返す UNIX 秒である。JSON 上は数値のまま扱う。
type Timestamp int64

// Time は time.Time に変換する。
func (t Timestamp) Time() time.Time { return time.Unix(int64(t), 0) }

// IsZero は値が未設定かを返す。
func (t Timestamp) IsZero() bool { return t == 0 }
