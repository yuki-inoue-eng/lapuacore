package configs

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains"
)

// Strategy holds the strategy name.
type Strategy struct {
	Name string
}

// ParamMap provides thread-safe typed access to configuration parameters.
// When a typed getter (GetInt, GetBool, etc.) fails to parse a value after
// a hot-reload, ParamMap returns the last successfully parsed value from cache
// and periodically logs a summary of failing keys.
type ParamMap struct {
	mu         sync.RWMutex
	m          map[string]string
	cache      map[string]any
	failedKeys map[string]failedParam // key → failure detail
}

type failedParam struct {
	rawValue string
	typeName string
}

func (p *ParamMap) setParams(m map[string]string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.m = m
	p.failedKeys = map[string]failedParam{}
}

// cacheHit records a successful parse and returns the value.
func cacheHit[T any](p *ParamMap, key string, val T) T {
	p.cache[key] = val
	delete(p.failedKeys, key)
	return val
}

// cacheFallback returns the cached value for the key if available.
// If no cached value exists, it returns the zero value of T.
// It records the failure for periodic logging by the Watcher.
func cacheFallback[T any](p *ParamMap, key, rawValue, typeName string) T {
	if _, already := p.failedKeys[key]; !already {
		slog.Warn("config param parse failed, using cached value", "key", key, "rawValue", rawValue, "expectedType", typeName)
	}
	p.failedKeys[key] = failedParam{rawValue: rawValue, typeName: typeName}
	cached, ok := p.cache[key]
	if !ok {
		var zero T
		return zero
	}
	return cached.(T)
}

// logFailedKeys logs a summary of keys that are currently failing to parse.
// Called periodically by the Watcher.
func (p *ParamMap) logFailedKeys() {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.failedKeys) == 0 {
		return
	}
	slog.Warn(formatFailedKeys(p.failedKeys))
}

func formatFailedKeys(failedKeys map[string]failedParam) string {
	maxKeyLen := 0
	maxRawLen := 0
	for k, fp := range failedKeys {
		if l := len(k) + 1; l > maxKeyLen { // +1 for colon
			maxKeyLen = l
		}
		if l := len(fmt.Sprintf("%q", fp.rawValue)); l > maxRawLen {
			maxRawLen = l
		}
	}

	msg := "The following config params failed to parse, using cached values\n"
	msg += "===== key / raw value / expected type =====\n"
	for k, fp := range failedKeys {
		quoted := fmt.Sprintf("%q", fp.rawValue)
		msg += fmt.Sprintf("%-*s  %-*s  %s\n", maxKeyLen, k+":", maxRawLen, quoted, fp.typeName)
	}
	return msg
}

func (p *ParamMap) Get(key string) string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	v, ok := p.m[key]
	if !ok {
		return ""
	}
	return v
}

func (p *ParamMap) GetBool(key string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	v, ok := p.m[key]
	if !ok {
		return cacheFallback[bool](p, key, "", "bool")
	}
	boolValue, err := strconv.ParseBool(v)
	if err != nil {
		return cacheFallback[bool](p, key, v, "bool")
	}
	return cacheHit(p, key, boolValue)
}

func (p *ParamMap) GetInt(key string) int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	raw := p.m[key]
	n, err := strconv.Atoi(raw)
	if err != nil {
		return cacheFallback[int](p, key, raw, "int")
	}
	return cacheHit(p, key, n)
}

func (p *ParamMap) GetInt64(key string) int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	raw := p.m[key]
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return cacheFallback[int64](p, key, raw, "int64")
	}
	return cacheHit(p, key, n)
}

func (p *ParamMap) GetFloat32(key string) float32 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	raw := p.m[key]
	n, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return cacheFallback[float32](p, key, raw, "float32")
	}
	return cacheHit(p, key, float32(n))
}

func (p *ParamMap) GetListStr(key string) []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	raw := p.m[key]
	var l []string
	if err := json.Unmarshal([]byte(raw), &l); err != nil {
		return cacheFallback[[]string](p, key, raw, "[]string")
	}
	return cacheHit(p, key, l)
}

func (p *ParamMap) GetListInt(key string) []int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	raw := p.m[key]
	var l []int
	if err := json.Unmarshal([]byte(raw), &l); err != nil {
		return cacheFallback[[]int](p, key, raw, "[]int")
	}
	return cacheHit(p, key, l)
}

func (p *ParamMap) GetListInt64(key string) []int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	raw := p.m[key]
	var l []int64
	if err := json.Unmarshal([]byte(raw), &l); err != nil {
		return cacheFallback[[]int64](p, key, raw, "[]int64")
	}
	return cacheHit(p, key, l)
}

func (p *ParamMap) GetListFloat(key string) []float32 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	raw := p.m[key]
	var l []float32
	if err := json.Unmarshal([]byte(raw), &l); err != nil {
		return cacheFallback[[]float32](p, key, raw, "[]float32")
	}
	return cacheHit(p, key, l)
}

func (p *ParamMap) GetListDecimal(key string) []decimal.Decimal {
	lf32 := p.GetListFloat(key)
	var l []decimal.Decimal
	for _, f32 := range lf32 {
		l = append(l, decimal.NewFromFloat32(f32))
	}
	return l
}

func (p *ParamMap) GetFromInt(key string) decimal.Decimal {
	return decimal.NewFromInt(p.GetInt64(key))
}

func (p *ParamMap) GetFromFloat(key string) decimal.Decimal {
	return decimal.NewFromFloat32(p.GetFloat32(key))
}

func (p *ParamMap) GetSymbol(key string) *domains.Symbol {
	p.mu.RLock()
	defer p.mu.RUnlock()
	raw := p.m[key]
	symbol := domains.GetSymbol(raw)
	if symbol == domains.SymbolUnknown {
		return cacheFallback[*domains.Symbol](p, key, raw, "Symbol")
	}
	return cacheHit(p, key, symbol)
}

func (p *ParamMap) GetMilliSec(key string) time.Duration {
	return time.Duration(p.GetInt(key)) * time.Millisecond
}

func (p *ParamMap) GetHour(key string) time.Duration {
	return time.Duration(p.GetInt(key)) * time.Hour
}

func (p *ParamMap) GetSec(key string) time.Duration {
	return time.Duration(p.GetInt(key)) * time.Second
}

// Config holds the parsed configuration with strategy and parameters.
type Config struct {
	Strategy *Strategy
	Params   *ParamMap
}

func newConfig(raw *RawConfig) *Config {
	conf := &Config{
		Strategy: &Strategy{
			Name: raw.Strategy.Name,
		},
		Params: &ParamMap{
			m:          raw.Params,
			cache:      map[string]any{},
			failedKeys: map[string]failedParam{},
		},
	}
	return conf
}

// update refreshes the parameters from a new raw config. Only params are updatable.
func (c *Config) update(raw *RawConfig) {
	c.Params.setParams(raw.Params)
}
