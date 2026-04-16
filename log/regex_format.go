package log

import (
	"fmt"
	"regexp"
	"time"
)

type regexConfig struct {
	Name       string            `toml:"name"`
	Pattern    string            `toml:"pattern"`
	TimeFormat string            `toml:"time_format"`
	LevelMap   map[string]string `toml:"level_map"`
}

type RegexFormat struct {
	name       string
	re         *regexp.Regexp
	groupNames []string
	timeFormat string
	levelMap   map[string]string
}

func newRegexFormat(cfg regexConfig) (*RegexFormat, error) {
	if cfg.Name == "" {
		return nil, fmt.Errorf("regex format: name is required")
	}
	if cfg.Pattern == "" {
		return nil, fmt.Errorf("regex format %q: pattern is required", cfg.Name)
	}
	re, err := regexp.Compile(cfg.Pattern)
	if err != nil {
		return nil, fmt.Errorf("regex format %q: %w", cfg.Name, err)
	}
	return &RegexFormat{
		name:       cfg.Name,
		re:         re,
		groupNames: re.SubexpNames(),
		timeFormat: cfg.TimeFormat,
		levelMap:   cfg.LevelMap,
	}, nil
}

func (f *RegexFormat) Name() string { return f.name }

func (f *RegexFormat) ParseRecord(line string) (LogRecord, error) {
	match := f.re.FindStringSubmatchIndex(line)
	if match == nil {
		return LogRecord{}, fmt.Errorf("line does not match %s pattern", f.name)
	}
	rec := LogRecord{
		RawJSON: line,
		Attrs:   make(Attrs, 0, len(f.groupNames)),
	}
	for i, name := range f.groupNames {
		if name == "" || i*2 >= len(match) {
			continue
		}
		start, end := match[i*2], match[i*2+1]
		if start < 0 {
			continue
		}
		value := line[start:end]
		switch name {
		case "time":
			rec.Time = f.parseTime(value)
		case "level":
			rec.Level = f.parseLevel(value)
		case "msg":
			rec.Msg = value
		default:
			rec.Attrs.Set(name, inferValue(value))
		}
	}
	return rec, nil
}

func (f *RegexFormat) parseTime(s string) time.Time {
	if f.timeFormat != "" {
		if t, err := time.Parse(f.timeFormat, s); err == nil {
			return t
		}
	}
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func (f *RegexFormat) parseLevel(s string) string {
	if f.levelMap != nil {
		if mapped, ok := f.levelMap[s]; ok {
			return mapped
		}
	}
	return normalizeLevel(s)
}
