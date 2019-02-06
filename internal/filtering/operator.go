package filtering

type QueryFilterOperator int8

const (
	QueryFilterOperatorAnd QueryFilterOperator = 0
	QueryFilterOperatorOr  QueryFilterOperator = 1
)
