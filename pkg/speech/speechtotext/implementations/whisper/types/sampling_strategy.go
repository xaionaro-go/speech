package types

type SamplingStrategy int

const (
	SamplingStrategyUndefined = SamplingStrategy(iota)
	SamplingStrategyGreedy
	SamplingStrategyBreamSearch
)
