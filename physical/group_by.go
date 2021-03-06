package physical

import (
	"context"
	"fmt"
	"strings"

	"github.com/cube2222/octosql"
	"github.com/cube2222/octosql/execution"
	"github.com/cube2222/octosql/execution/aggregates"
	"github.com/cube2222/octosql/graph"
	"github.com/cube2222/octosql/physical/metadata"
	"github.com/pkg/errors"
)

type Aggregate string

const (
	Avg           Aggregate = "avg"
	AvgDistinct   Aggregate = "avg_distinct"
	Count         Aggregate = "count"
	CountDistinct Aggregate = "count_distinct"
	First         Aggregate = "first"
	Last          Aggregate = "last"
	Max           Aggregate = "max"
	Min           Aggregate = "min"
	Sum           Aggregate = "sum"
	SumDistinct   Aggregate = "sum_distinct"
)

func NewAggregate(aggregate string) Aggregate {
	return Aggregate(strings.ToLower(aggregate))
}

type GroupBy struct {
	Source Node
	Key    []Expression

	Fields     []octosql.VariableName
	Aggregates []Aggregate

	As []octosql.VariableName
}

func NewGroupBy(source Node, key []Expression, fields []octosql.VariableName, aggregates []Aggregate, as []octosql.VariableName) *GroupBy {
	return &GroupBy{Source: source, Key: key, Fields: fields, Aggregates: aggregates, As: as}
}

func (node *GroupBy) Transform(ctx context.Context, transformers *Transformers) Node {
	key := make([]Expression, len(node.Key))
	for i := range node.Key {
		key[i] = node.Key[i].Transform(ctx, transformers)
	}

	source := node.Source.Transform(ctx, transformers)

	var transformed Node = &GroupBy{
		Source:     source,
		Key:        key,
		Fields:     node.Fields,
		Aggregates: node.Aggregates,
		As:         node.As,
	}

	if transformers.NodeT != nil {
		transformed = transformers.NodeT(transformed)
	}

	return transformed
}

func (node *GroupBy) Materialize(ctx context.Context, matCtx *MaterializationContext) (execution.Node, error) {
	source, err := node.Source.Materialize(ctx, matCtx)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't materialize Source node")
	}

	key := make([]execution.Expression, len(node.Key))
	for i := range node.Key {
		keyPart, err := node.Key[i].Materialize(ctx, matCtx)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't materialize group key expression with index %v", i)
		}

		key[i] = keyPart
	}

	aggregatePrototypes := make([]execution.AggregatePrototype, len(node.Aggregates))
	for i := range node.Aggregates {
		aggregatePrototypes[i] = aggregates.AggregateTable[string(node.Aggregates[i])]
	}

	return execution.NewGroupBy(source, key, node.Fields, aggregatePrototypes, node.As), nil
}

func (node *GroupBy) Metadata() *metadata.NodeMetadata {
	var cardinality = node.Metadata().Cardinality()
	if cardinality == metadata.BoundedDoesntFitInLocalStorage {
		cardinality = metadata.BoundedFitsInLocalStorage
	}
	return metadata.NewNodeMetadata(cardinality, octosql.NewVariableName(""))
}

func (node *GroupBy) Visualize() *graph.Node {
	n := graph.NewNode("Group By")

	n.AddChild("source", node.Source.Visualize())
	for i, expr := range node.Key {
		n.AddChild(fmt.Sprintf("key_%d", i), expr.Visualize())
	}

	for i := range node.Fields {
		value := fmt.Sprintf("%s(%s)", node.Aggregates[i], node.Fields[i])
		if !node.As[i].Empty() {
			value += fmt.Sprintf(" as %s", node.As[i])
		}
		n.AddField(fmt.Sprintf("field_%d", i), value)
	}

	return n
}
