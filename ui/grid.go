package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/filter"
	"github.com/pgavlin/tea-grid/grid"
	"github.com/pgavlin/tea-grid/selection"

	logpkg "github.com/pgavlin/hustle/log"
)

// containsFold reports whether s contains substr under Unicode case-folding.
func containsFold(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if strings.EqualFold(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func logColumns() []data.Column[logpkg.LogRecord] {
	return []data.Column[logpkg.LogRecord]{
		{
			ColumnID:   "time",
			HeaderName: "Time",
			Value: func(r logpkg.LogRecord) any {
				return r.Time
			},
			Text: func(r *logpkg.LogRecord) string {
				return r.Time.Format("15:04:05.000")
			},
			Compare: func(a, b *logpkg.LogRecord) int {
				return a.Time.Compare(b.Time)
			},
			QuickFilterMatch: func(r *logpkg.LogRecord, word string) bool {
				return containsFold(r.Time.Format("15:04:05.000"), word)
			},
			Width:      14,
			Filterable: true,
			Filter:     filter.NewTimeFilter(),
			Sortable:   true,
		},
		{
			ColumnID:   "level",
			HeaderName: "Level",
			Value: func(r logpkg.LogRecord) any {
				return r.Level
			},
			Text: func(r *logpkg.LogRecord) string {
				return r.Level
			},
			Compare: func(a, b *logpkg.LogRecord) int {
				return strings.Compare(a.Level, b.Level)
			},
			QuickFilterMatch: func(r *logpkg.LogRecord, word string) bool {
				return containsFold(r.Level, word)
			},
			Width:      7,
			Filterable: true,
			Filter:     filter.NewSetFilter("DEBUG", "INFO", "WARN", "ERROR"),
			Sortable:   true,
		},
		{
			ColumnID:   "msg",
			HeaderName: "Message",
			Value: func(r logpkg.LogRecord) any {
				return r.Msg
			},
			Text: func(r *logpkg.LogRecord) string {
				return r.Msg
			},
			Compare: func(a, b *logpkg.LogRecord) int {
				return strings.Compare(a.Msg, b.Msg)
			},
			QuickFilterMatch: func(r *logpkg.LogRecord, word string) bool {
				return containsFold(r.Msg, word)
			},
			Flex:       2,
			Filterable: true,
			Filter:     filter.NewTextFilter(),
			Sortable:   true,
		},
		{
			ColumnID:   "attrs",
			HeaderName: "Attributes",
			Value: func(r logpkg.LogRecord) any {
				return formatAttrs(r.Attrs)
			},
			Text: func(r *logpkg.LogRecord) string {
				return formatAttrs(r.Attrs)
			},
			QuickFilterMatch: func(r *logpkg.LogRecord, word string) bool {
				// Search keys and values directly without sorting or formatting
				for _, kv := range r.Attrs {
					if containsFold(kv.Key, word) {
						return true
					}
					if s, ok := kv.Value.(string); ok {
						if containsFold(s, word) {
							return true
						}
					} else {
						if containsFold(fmt.Sprint(kv.Value), word) {
							return true
						}
					}
				}
				return false
			},
			Flex:       1,
			Filterable: true,
			Filter:     filter.NewTextFilter(),
		},
	}
}

func newLogGrid(records []logpkg.LogRecord, width, height int, extFilter func(logpkg.LogRecord) bool) grid.Model[logpkg.LogRecord] {
	opts := []grid.Option[logpkg.LogRecord]{
		grid.WithColumns(logColumns()),
		grid.WithRows(records),
		grid.WithRowID(func() func(logpkg.LogRecord) string {
			i := 0
			return func(r logpkg.LogRecord) string {
				id := strconv.Itoa(i)
				i++
				return id
			}
		}()),
		grid.WithWidth[logpkg.LogRecord](width),
		grid.WithHeight[logpkg.LogRecord](height),
		grid.WithSelection[logpkg.LogRecord](selection.SelectSingle),
		grid.WithQuickFilter[logpkg.LogRecord](true),
		grid.WithFocused[logpkg.LogRecord](true),
	}
	if extFilter != nil {
		opts = append(opts, grid.WithExternalFilter(extFilter))
	}
	return grid.New(opts...)
}

func formatAttrs(attrs logpkg.Attrs) string {
	if len(attrs) == 0 {
		return ""
	}

	parts := make([]string, 0, len(attrs))
	for _, kv := range attrs {
		parts = append(parts, fmt.Sprintf("%s=%v", kv.Key, kv.Value))
	}
	return strings.Join(parts, ", ")
}
