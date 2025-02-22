// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/cortexproject/cortex/blob/master/pkg/querier/astmapper/parallel_test.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Cortex Authors.

package astmapper

import (
	"fmt"
	"testing"

	"github.com/go-kit/log"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
	"github.com/stretchr/testify/require"
)

func TestCanParallel(t *testing.T) {
	testExpr := []struct {
		input    parser.Expr
		expected bool
	}{
		// simple sum
		{
			&parser.AggregateExpr{
				Op:      parser.SUM,
				Without: true,
				Expr: &parser.VectorSelector{
					Name: "some_metric",
					LabelMatchers: []*labels.Matcher{
						mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "some_metric"),
					},
				},
				Grouping: []string{"foo"},
			},
			true,
		},
		/*
			  sum(
				  sum by (foo) bar1{baz=”blip”}[1m])
				/
				  sum by (foo) bar2{baz=”blip”}[1m]))
			  )
		*/
		{
			&parser.AggregateExpr{
				Op: parser.SUM,
				Expr: &parser.BinaryExpr{
					Op: parser.DIV,
					LHS: &parser.AggregateExpr{
						Op:       parser.SUM,
						Grouping: []string{"foo"},
						Expr: &parser.VectorSelector{
							Name: "idk",
							LabelMatchers: []*labels.Matcher{
								mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "bar1"),
							},
						},
					},
					RHS: &parser.AggregateExpr{
						Op:       parser.SUM,
						Grouping: []string{"foo"},
						Expr: &parser.VectorSelector{
							Name: "idk",
							LabelMatchers: []*labels.Matcher{
								mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "bar2"),
							},
						},
					},
				},
			},
			false,
		},
		// sum by (foo) bar1{baz=”blip”}[1m]) ---- this is the first leg of the above
		{
			&parser.AggregateExpr{
				Op:       parser.SUM,
				Grouping: []string{"foo"},
				Expr: &parser.VectorSelector{
					Name: "idk",
					LabelMatchers: []*labels.Matcher{
						mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "bar1"),
					},
				},
			},
			true,
		},
	}

	for i, c := range testExpr {
		t.Run(fmt.Sprintf("[%d]", i), func(t *testing.T) {
			res := CanParallelize(c.input, log.NewNopLogger())
			require.Equal(t, c.expected, res)
		})
	}
}

func TestCanParallel_String(t *testing.T) {
	testExpr := []struct {
		input    string
		expected bool
	}{
		{
			`sum by (foo) (rate(bar1{baz="blip"}[1m]))`,
			true,
		},
		{
			`sum by (foo) (histogram_quantile(0.9, rate(http_request_duration_seconds_bucket[10m])))`,
			false,
		},
		{
			`sum by (foo) (
			  quantile_over_time(0.9, http_request_duration_seconds_bucket[10m])
			)`,
			true,
		},
		{
			`sum(
				count(
					count(
						foo{bar="baz"}
					)  by (a,b)
				)  by (instance)
			)`,
			false,
		},
		{
			`min_over_time(
				sum by(group_1) (
					rate(metric_counter[5m])
				)[10m:2m]
			)`,
			false,
		},
		{
			`sum_over_time(
				rate(metric_counter[5m])
				[10m:2m])`,
			true,
		},
		{
			`sum by(group_1) (
				min_over_time(
					rate(metric_counter[5m])
				[10m:2m])
			)`,
			true,
		},
		{
			`max_over_time(
				deriv(
					rate(metric_counter[10m])
				[5m:1m])
			[10m:])`,
			true,
		},
		{
			`max_over_time(
				stddev_over_time(
					deriv(
						rate(metric_counter[10m])
					[5m:1m])
				[2m:])
			[10m:])`,
			true,
		},
		{
			`max_over_time(
				deriv(
					sum by (foo) (
						rate(metric_counter[10m])
					)[5m:1m]
				)
			[10m:])`,
			false,
		},
		{
			`max_over_time(
				stddev_over_time(
					deriv(
						count by (foo) (rate(metric_counter[10m]))
					[5m:1m])
				[2m:])
			[10m:])`,
			false,
		},
	}

	for i, c := range testExpr {
		t.Run(fmt.Sprintf("[%d]", i), func(t *testing.T) {
			expr, err := parser.ParseExpr(c.input)
			require.Nil(t, err)
			res := CanParallelize(expr, log.NewNopLogger())
			require.Equal(t, c.expected, res)
		})
	}
}
