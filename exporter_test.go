//
// DISCLAIMER
//
// Copyright 2018 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//
// Author Ewout Prangsma
//

package main

import (
	"fmt"
	_ "net/http/pprof"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

// TestMetric tests the result of metricKey & newMetric for various inputs.
func TestMetric(t *testing.T) {
	g1 := StatisticGroup{
		Group:       "g1",
		Name:        "g1-name",
		Description: "Something g1",
	}
	tests := []struct {
		Group               StatisticGroup
		Figure              StatisticFigure
		Postfix             string
		KeyResult           string
		CollectionPostfixes []string
	}{
		{g1, StatisticFigure{"g1", "f1", "f1-name", "descr-f1", FigureTypeAccumulated, "", nil}, "_pf", "g1-name_f1-name_pf", []string{""}},
		{g1, StatisticFigure{"g1", "f1", "f1-name", "descr-f1", FigureTypeAccumulated, "tick", nil}, "_pf", "g1-name_f1-name_pf_tick", []string{""}},
		{g1, StatisticFigure{"g1", "f2", "f2-name", "descr-f2", FigureTypeDistribution, "", []float64{0.1, 0.2}}, "_pf", "g1-name_f2-name_pf", []string{"_sum", "_count", "_bucket"}},
	}

	for i, test := range tests {
		result := metricKey(test.Group, test.Figure, test.Postfix)
		if result != test.KeyResult {
			t.Errorf("metricKey for test %d failed: got '%s', expected '%s'", i, result, test.KeyResult)
		}
		colls := newMetric(test.Group, test.Figure)
		if len(colls) != len(test.CollectionPostfixes) {
			t.Errorf("newMetric for test %d returns unexpected #collectors: got %d, expected %d", i, len(colls), len(test.CollectionPostfixes))
		} else {
			for ci, c := range colls {
				descrChan := make(chan *prometheus.Desc, 1)
				c.Describe(descrChan)
				d := <-descrChan
				result := d.String()
				expectedPostfix := test.CollectionPostfixes[ci]
				if !strings.Contains(result, fmt.Sprintf("%s\", help", expectedPostfix)) {
					t.Errorf("newMetric for test %d returns collector %d with wrong expectation. got '%s', expected it to contain '%s'", i, ci, result, expectedPostfix)
				}
			}
		}
	}
}
