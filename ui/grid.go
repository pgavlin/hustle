package ui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/filter"
	"github.com/pgavlin/tea-grid/grid"
	"github.com/pgavlin/tea-grid/selection"

	logpkg "github.com/pgavlin/hustle/log"
)

func logColumns() []data.Column[logpkg.LogRecord] {
	return []data.Column[logpkg.LogRecord]{
		{
			ColumnID:   "time",
			HeaderName: "Time",
			ValueGetter: func(r logpkg.LogRecord) any {
				return r.Time
			},
			ValueFormatter: func(v any, r logpkg.LogRecord) string {
				return r.Time.Format("15:04:05.000")
			},
			Width:      14,
			Filterable: true,
			Filter:     filter.NewTimeFilter(),
			Sortable:   true,
		},
		{
			ColumnID:   "level",
			HeaderName: "Level",
			ValueGetter: func(r logpkg.LogRecord) any {
				return r.Level
			},
			Width:      7,
			Filterable: true,
			Filter:     filter.NewSetFilter("DEBUG", "INFO", "WARN", "ERROR"),
			Sortable:   true,
		},
		{
			ColumnID:   "msg",
			HeaderName: "Message",
			ValueGetter: func(r logpkg.LogRecord) any {
				return r.Msg
			},
			Flex:       2,
			Filterable: true,
			Filter:     filter.NewTextFilter(),
			Sortable:   true,
		},
		{
			ColumnID:   "attrs",
			HeaderName: "Attributes",
			ValueGetter: func(r logpkg.LogRecord) any {
				return formatAttrs(r.Attrs)
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

func formatAttrs(attrs map[string]any) string {
	if len(attrs) == 0 {
		return ""
	}
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, attrs[k]))
	}
	return strings.Join(parts, ", ")
}
