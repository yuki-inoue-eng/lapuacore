package configs

import (
	"encoding/json"
	"fmt"
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
type ParamMap struct {
	mu sync.RWMutex
	m  map[string]string
}

func (p *ParamMap) setParams(m map[string]string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.m = m
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
		panic(fmt.Errorf("param not found: %s", key))
	}
	boolValue, err := strconv.ParseBool(v)
	if err != nil {
		panic(fmt.Errorf("param is not bool value: %s", key))
	}
	return boolValue
}

func (p *ParamMap) GetInt(key string) int {
	n, err := strconv.Atoi(p.Get(key))
	if err != nil {
		panic(err)
	}
	return n
}

func (p *ParamMap) GetInt64(key string) int64 {
	n, err := strconv.ParseInt(p.Get(key), 10, 64)
	if err != nil {
		panic(err)
	}
	return n
}

func (p *ParamMap) GetFloat32(key string) float32 {
	n, err := strconv.ParseFloat(p.Get(key), 64)
	if err != nil {
		panic(err)
	}
	return float32(n)
}

func (p *ParamMap) GetListStr(key string) []string {
	listStr := p.Get(key)
	var l []string
	if err := json.Unmarshal([]byte(listStr), &l); err != nil {
		panic(err)
	}
	return l
}

func (p *ParamMap) GetListInt(key string) []int {
	listStr := p.Get(key)
	var l []int
	if err := json.Unmarshal([]byte(listStr), &l); err != nil {
		panic(err)
	}
	return l
}

func (p *ParamMap) GetListInt64(key string) []int64 {
	listStr := p.Get(key)
	var l []int64
	if err := json.Unmarshal([]byte(listStr), &l); err != nil {
		panic(err)
	}
	return l
}

func (p *ParamMap) GetListFloat(key string) []float32 {
	listStr := p.Get(key)
	var l []float32
	if err := json.Unmarshal([]byte(listStr), &l); err != nil {
		panic(err)
	}
	return l
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
	symbol := domains.GetSymbol(p.Get(key))
	if symbol == domains.SymbolUnknown {
		panic(fmt.Sprintf("failed to load symbol: unknown symbol (key: %s)", key))
	}
	return symbol
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
			m: raw.Params,
		},
	}
	return conf
}

// update refreshes the parameters from a new raw config. Only params are updatable.
func (c *Config) update(raw *RawConfig) {
	c.Params.setParams(raw.Params)
}
