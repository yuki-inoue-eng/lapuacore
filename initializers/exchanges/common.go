package exchanges

// Insight represents an exchange data source that can report its readiness.
type Insight interface {
	EXName() string
	IsEverythingReady() bool
}

// Insights holds all registered Insight instances.
var Insights []Insight

// AppendInsight registers an Insight to the global registry.
func AppendInsight(ins Insight) {
	Insights = append(Insights, ins)
}
