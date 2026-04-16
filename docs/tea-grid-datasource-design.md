# tea-grid Datasource Row Model

## Motivation

tea-grid's current row model requires all row data in memory via `WithRows([]T)`.
The grid owns sorting, filtering, and grouping over this slice. This works well
for datasets up to ~100K rows, but breaks down for larger data:

- A log viewer over a 10 GB file can't parse and hold millions of `LogRecord`
  values on the Go heap.
- A database-backed grid can't (and shouldn't) fetch every row upfront.
- An application tailing a stream has no finite row set at all.

AG Grid solves this with multiple row models. The most relevant is the
**Server-Side Row Model**, where the grid delegates sort/filter/grouping to a
datasource and only requests the rows it needs to render. We propose an analogous
model for tea-grid.

## Design Principles

1. **The grid drives the interaction.** The grid owns the UI state for sorting,
   filtering, grouping, and scrolling. When state changes, the grid tells the
   datasource what it needs. The datasource never pushes unsolicited data.

2. **Elm architecture.** Requests are issued as `tea.Cmd`s. Responses arrive as
   `tea.Msg`s. No shared mutable state, no channels, no callbacks.

3. **The datasource owns the computation.** The grid does not sort, filter, or
   group in datasource mode. It passes the full state to the datasource and
   renders whatever comes back. The datasource can implement these operations
   however it wants — linear scan, index lookup, SQL query.

4. **Client-side model is unchanged.** `WithRows([]T)` continues to work exactly
   as it does today. The datasource model is opt-in via `WithDatasource`.

## API

### Datasource Interface

```go
// Datasource provides row data on demand for datasets too large to hold in
// memory. The grid calls Request when it needs rows, passing the full
// sort/filter/group state. The datasource returns a tea.Cmd that produces
// a DatasourceResponseMsg[T] when executed.
type Datasource[T any] interface {
    Request(req DatasourceRequest[T]) tea.Cmd
}
```

### Request

```go
type DatasourceRequest[T any] struct {
    // Sequence is a monotonically increasing counter. The datasource must echo
    // it back in the response. The grid uses it to discard stale responses
    // (e.g. a response for a filter that has since changed).
    Sequence uint64

    // Row window. The grid requests rows [StartRow, EndRow).
    StartRow int
    EndRow   int

    // Sort criteria, in priority order. Empty means natural/insertion order.
    Sort []sort.SortCriterion

    // Column filters. Each entry carries the column ID and the filter instance
    // (which holds its own state: text, set values, range bounds, etc.).
    Filters []ColumnFilter

    // Quick filter text. Empty means no quick filter.
    QuickFilter string

    // Grouping columns, in nesting order. Empty means no grouping.
    GroupCols []string

    // Group keys. When the user expands a group, the grid sends the key path
    // from root to the expanded node. Empty means top-level rows/groups.
    // Example: GroupCols=["country","year"], GroupKeys=["Argentina"] means
    // "give me the year-level groups (or leaf rows) under Argentina."
    GroupKeys []string
}

type ColumnFilter struct {
    ColumnID string
    Filter   filter.Filter
}
```

### Response

```go
// DatasourceResponseMsg is the tea.Msg produced by the Cmd returned from
// Datasource.Request.
type DatasourceResponseMsg[T any] struct {
    // Sequence echoed from the request.
    Sequence uint64

    // Rows for the requested window. May be shorter than EndRow-StartRow if
    // the window extends past the end of the data.
    Rows []T

    // RowCount is the total number of rows matching the current
    // sort/filter/group state.
    //   >0 : exact count (grid renders a scrollbar of this height)
    //    0 : no matching rows
    //   -1 : unknown (grid allows infinite scroll; datasource signals the end
    //         by returning fewer rows than requested)
    RowCount int
}
```

### Grid Option

```go
func WithDatasource[T any](ds Datasource[T]) Option[T] {
    return func(m *Model[T]) {
        m.datasource = ds
    }
}
```

`WithDatasource` and `WithRows` are mutually exclusive. Passing both is a
programming error (panic at construction time is fine).

## Grid Behavior in Datasource Mode

### State

```go
// Added to grid.Model[T]:
datasource Datasource[T]
dsSequence uint64              // bumped on every state change
dsCache    map[blockKey][]RowNode[T]
dsTotalRows int                // from most recent response (-1 = unknown)
dsLoading   bool               // true while a request is in flight
dsStale     []RowNode[T]       // previous displayRows, shown during loading
```

### State Machine

**Sort/filter/group change:**
1. Bump `dsSequence`.
2. Clear `dsCache`.
3. Save current `displayRows` as `dsStale`.
4. Set `dsLoading = true`.
5. Issue `Request` for the current visible window.
6. Render `dsStale` with a loading indicator.

**Scroll (cache miss):**
1. Bump `dsSequence`.
2. Set `dsLoading = true`.
3. Issue `Request` for the new visible range.
4. Render cached rows where available; gaps show placeholder rows.

**Scroll (cache hit):**
1. No request issued. Build `displayRows` from cache. This is the fast path.

**Response received:**
1. If `msg.Sequence != m.dsSequence`, discard (stale).
2. Populate `dsCache` with the response block.
3. Set `dsLoading = false`.
4. Build `displayRows` from cache for the current visible range.
5. Update `dsTotalRows` from `msg.RowCount`.
6. Optionally issue a pre-fetch `Request` for the next block in the scroll
   direction.

### Rendering During Loading

While `dsLoading` is true, the grid renders `dsStale` (the previous display
rows) in place. This avoids flicker on filter changes. A configurable loading
style (e.g. dimmed rows, a status line indicator) signals that the data is
stale. If no stale data exists (first load), the grid renders empty rows or a
"Loading..." placeholder.

### Block Cache

The cache is keyed by `(startRow, endRow)` within a single sequence epoch.
Any sequence bump invalidates the entire cache.

Configuration:

```go
// WithDatasourceBlockSize sets the number of rows per cache block.
// Default: 100.
func WithDatasourceBlockSize[T any](n int) Option[T]

// WithDatasourceMaxBlocks sets the maximum number of cached blocks.
// Default: 10. When exceeded, the block furthest from the viewport is evicted.
func WithDatasourceMaxBlocks[T any](n int) Option[T]
```

When the grid needs rows [200, 250) and blockSize=100, it requests block
[200, 300). The extra rows are cached for subsequent scrolling.

### Pre-fetching

After receiving a response, the grid issues a speculative `Request` for the
adjacent block in the user's scroll direction. This keeps one block ahead
of the viewport so smooth scrolling rarely hits a cache miss.

### Features Disabled in Datasource Mode

These features require in-memory access to all rows and are disabled (no-op)
when a datasource is active:

- **Client-side sort/filter/grouping** — the datasource handles these.
- **Row transactions** (`AddRows`, `RemoveRows`, `UpdateRows`) — the
  datasource owns the data; mutations go through it.
- **Dynamic row pinning predicates** — the datasource should pre-sort pinned
  rows to the appropriate positions.

Features that continue to work unchanged:

- **Viewport and scrolling** — the grid still owns the viewport.
- **Selection** — by display index, works as before.
- **Focus and keyboard navigation** — unchanged.
- **Column sizing, pinning, reordering** — purely visual, no data dependency.
- **Cell rendering** — the grid calls `ValueGetter`/`ValueFormatter` on
  whatever `T` the datasource returns.
- **Static pinned rows** — `WithStaticPinnedTop`/`Bottom` still work; these
  rows are provided directly, not through the datasource.

## Example: hustle Log Viewer

hustle views log files that can be multiple gigabytes. With the datasource
model:

```go
type logDatasource struct {
    data    []byte   // mmap'd file
    offsets []int64  // byte offset of each line
    format  log.Format

    mu       sync.Mutex
    index    []int32  // filtered+sorted line indices
    indexSeq uint64   // sequence when index was last built
}

func (ds *logDatasource) Request(req grid.DatasourceRequest[log.LogRecord]) tea.Cmd {
    return func() tea.Msg {
        ds.mu.Lock()
        defer ds.mu.Unlock()

        // Rebuild filtered/sorted index if state changed
        if ds.indexSeq != req.Sequence {
            ds.rebuildIndex(req)
            ds.indexSeq = req.Sequence
        }

        // Parse only the requested window
        end := req.EndRow
        if end > len(ds.index) {
            end = len(ds.index)
        }
        rows := make([]log.LogRecord, 0, end-req.StartRow)
        for i := req.StartRow; i < end; i++ {
            line := ds.lineAt(ds.index[i])
            if rec, err := ds.format.ParseRecord(line); err == nil {
                rows = append(rows, rec)
            }
        }

        return grid.DatasourceResponseMsg[log.LogRecord]{
            Sequence: req.Sequence,
            Rows:     rows,
            RowCount: len(ds.index),
        }
    }
}
```

Memory profile for a 10 GB file with 50M lines:

| Data                         | Size     |
|------------------------------|----------|
| mmap'd file (virtual)        | 10 GB    |
| Resident pages (OS-managed)  | ~100 MB  |
| Line offset index            | 400 MB   |
| Filtered index (worst case)  | 200 MB   |
| Parsed records (visible)     | ~50 KB   |
| **Go heap total**            | **~600 MB** |

Compared to loading all records: ~10–25 GB of Go heap.

## AG Grid Precedent

AG Grid provides four row models: Client-Side, Infinite, Server-Side, and
Viewport. The Server-Side model is the closest analogue. Key similarities:

- Grid sends sort/filter/group state with every request.
- Datasource returns a row window and total count.
- Grid caches blocks and evicts on state change.
- Grouping is lazy: expanding a group sends the group key path, datasource
  returns children.

Key differences from our design:

- AG Grid's `getRows` is callback-based (JavaScript); ours uses `tea.Cmd`/
  `tea.Msg` to fit the Elm architecture.
- AG Grid has a separate Infinite model (flat only, no grouping); we don't
  need this — our single datasource interface handles both cases via the
  optional `GroupCols`/`GroupKeys` fields.
- AG Grid's Viewport model (server pushes data) has no analogue here. If
  needed in the future, it could be a second datasource interface.

## Open Questions

1. **Row IDs in datasource mode.** Client-side mode derives row IDs via
   `WithRowID(func(T) string)`. In datasource mode, should the datasource
   return IDs alongside rows (as part of the response), or should the same
   `WithRowID` function apply to datasource rows? The latter is simpler but
   may require the datasource to provide data that supports stable ID
   derivation.

2. **Partial grouping.** Should the grid support client-side grouping over
   datasource-provided flat rows (for small filtered result sets), or is
   grouping always fully delegated? AG Grid always delegates in server-side
   mode.

3. **Streaming/tailing.** A future extension could allow the datasource to
   send unsolicited `DatasourceResponseMsg` when new rows arrive (e.g. log
   tailing). This would require a subscription mechanism — the grid registers
   interest and the datasource pushes updates. Out of scope for the initial
   design.
