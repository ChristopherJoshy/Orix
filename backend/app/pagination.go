package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type listCursor struct {
	Time  time.Time `json:"t"`
	ID    string    `json:"id"`
	Scope string    `json:"scope"`
}
type listPage struct {
	Limit  int
	Before *listCursor
	Next   *listCursor
	Scope  string
}
type listPageKey struct{}

func preparePage(w http.ResponseWriter, r *http.Request, scope string, defaultLimit int) (*http.Request, bool) {
	page := &listPage{Limit: defaultLimit, Scope: scope}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 100 {
			writeJSON(w, 422, map[string]string{"detail": "Limit must be between 1 and 100"})
			return r, false
		}
		page.Limit = value
	}
	if raw := r.URL.Query().Get("before"); raw != "" {
		var cursor listCursor
		decoded, err := base64.RawURLEncoding.DecodeString(raw)
		if len(raw) > 1024 || err != nil || json.Unmarshal(decoded, &cursor) != nil || cursor.Time.IsZero() || cursor.ID == "" || len(cursor.ID) > 128 || cursor.Scope != scope {
			writeJSON(w, 422, map[string]string{"detail": "Invalid page cursor"})
			return r, false
		}
		page.Before = &cursor
	}
	return r.WithContext(context.WithValue(r.Context(), listPageKey{}, page)), true
}
func requestPage(ctx context.Context) *listPage {
	page, _ := ctx.Value(listPageKey{}).(*listPage)
	return page
}
func pageLimit(ctx context.Context, fallback int) int64 {
	if page := requestPage(ctx); page != nil {
		return int64(page.Limit + 1)
	}
	return int64(fallback)
}
func pageQuery(ctx context.Context, query bson.M, timeField, idField string) {
	page := requestPage(ctx)
	if page == nil || page.Before == nil {
		return
	}
	cursor := page.Before
	condition := bson.M{"$or": bson.A{bson.M{timeField: bson.M{"$lt": cursor.Time}}, bson.M{timeField: cursor.Time, idField: bson.M{"$lt": cursor.ID}}}}
	and, _ := query["$and"].(bson.A)
	query["$and"] = append(and, condition)
}
func trimPage[T any](ctx context.Context, records []T, position func(T) (time.Time, string)) []T {
	page := requestPage(ctx)
	if page == nil || len(records) <= page.Limit {
		return records
	}
	records = records[:page.Limit]
	timestamp, id := position(records[len(records)-1])
	page.Next = &listCursor{timestamp.UTC(), id, page.Scope}
	return records
}
func writePageHeader(w http.ResponseWriter, r *http.Request) {
	page := requestPage(r.Context())
	if page == nil || page.Next == nil {
		return
	}
	encoded, _ := json.Marshal(page.Next)
	w.Header().Set("X-Next-Cursor", base64.RawURLEncoding.EncodeToString(encoded))
}
