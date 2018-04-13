package main

import (
	"context"
	_ "net/http/pprof"

	driver "github.com/arangodb/go-driver"
)

// StatisticsDescription is the JSON representation of the result of an _admin/statistics-description call.
type StatisticsDescription struct {
	Groups  []StatisticGroup  `json:"groups"`
	Figures []StatisticFigure `json:"figures"`
}

// StatisticGroup describes a group of statistics.
type StatisticGroup struct {
	Group       string `json:"group"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// StatisticFigure describes a single statistic.
type StatisticFigure struct {
	Group       string     `json:"group"`
	Identifier  string     `json:"identifier"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Type        FigureType `json:"type"`
	Units       string     `json:"unit,omitempty"`
	Cuts        []float64  `json:"cuts,omitempty"`
}

// FigureType is a strongly typed type of statistic.
type FigureType string

const (
	FigureTypeAccumulated  FigureType = "accumulated"
	FigureTypeCurrent      FigureType = "current"
	FigureTypeDistribution FigureType = "distribution"
)

// Statistics is a stringly typed map containing statistic values
type Statistics map[string]interface{}

// GetGroup returns the statistics for the given group.
// If not found, nil is returned
func (s Statistics) GetGroup(group string) Statistics {
	entry, ok := s[group]
	if !ok {
		return nil
	}
	result, ok := entry.(map[string]interface{})
	if !ok {
		return nil
	}
	return result
}

// GetFloat returns a float value for a statistic with given key.
// If not found, or the value could not be converted to a float, false is returned.
func (s Statistics) GetFloat(key string) (float64, bool) {
	raw, ok := s[key]
	if !ok {
		return 0.0, false
	}
	if f, ok := raw.(float64); ok {
		return f, true
	}
	if f, ok := raw.(int64); ok {
		return float64(f), true
	}
	return 0.0, false
}

// GetCounts returns an int64 array value for a statistic with given key.
// If not found, or the value could not be converted to an int64 array, false is returned.
func (s Statistics) GetCounts(key string) ([]int64, bool) {
	raw, ok := s[key]
	if !ok {
		return nil, false
	}
	if rawSlice, ok := raw.([]interface{}); ok {
		result := make([]int64, len(rawSlice))
		for i, x := range rawSlice {
			if v, ok := x.(int64); ok {
				result[i] = v
			} else if v, ok := x.(int); ok {
				result[i] = int64(v)
			} else if v, ok := x.(float64); ok {
				result[i] = int64(v)
			} else {
				return nil, false
			}
		}
		return result, true
	}
	return nil, false
}

// GetInt returns an int value for a statistic with given key.
// If not found, or the value could not be converted to an int, false is returned.
func (s Statistics) GetInt(key string) (int64, bool) {
	raw, ok := s[key]
	if !ok {
		return 0.0, false
	}
	if i, ok := raw.(int64); ok {
		return i, true
	}
	if i, ok := raw.(int); ok {
		return int64(i), true
	}
	return 0, false
}

// GetStatistics requests the statistics values from the given connection.
func GetStatistics(ctx context.Context, conn driver.Connection) (Statistics, error) {
	req, err := conn.NewRequest("GET", "_admin/statistics")
	if err != nil {
		return nil, maskAny(err)
	}
	resp, err := conn.Do(ctx, req)
	if err != nil {
		return nil, maskAny(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, maskAny(err)
	}
	var result Statistics
	if err := resp.ParseBody("", &result); err != nil {
		return nil, maskAny(err)
	}
	return result, nil
}

// GetStatisticsDescription requests the statistics description from the given connection.
func GetStatisticsDescription(ctx context.Context, conn driver.Connection) (StatisticsDescription, error) {
	req, err := conn.NewRequest("GET", "_admin/statistics-description")
	if err != nil {
		return StatisticsDescription{}, maskAny(err)
	}
	resp, err := conn.Do(ctx, req)
	if err != nil {
		return StatisticsDescription{}, maskAny(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return StatisticsDescription{}, maskAny(err)
	}
	var result StatisticsDescription
	if err := resp.ParseBody("", &result); err != nil {
		return StatisticsDescription{}, maskAny(err)
	}
	return result, nil
}
