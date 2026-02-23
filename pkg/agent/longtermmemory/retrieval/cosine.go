package retrieval

import (
	"math"
	"sort"
)

// cosineTopK returns the top-k entries by cosine similarity to query.
// Assumes all vectors are normalised (unit length); uses dot product directly.
func cosineTopK(entries []MemoryVector, query []float32, k int) []MemoryVector {
	type scored struct {
		entry MemoryVector
		score float64
	}
	scores := make([]scored, 0, len(entries))
	for _, e := range entries {
		scores = append(scores, scored{entry: e, score: dotProduct(query, e.Vector)})
	}
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})
	if k > len(scores) {
		k = len(scores)
	}
	out := make([]MemoryVector, k)
	for i := range out {
		out[i] = scores[i].entry
	}
	return out
}

func dotProduct(a, b []float32) float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	var sum float64
	for i := 0; i < n; i++ {
		sum += float64(a[i]) * float64(b[i])
	}
	return sum
}

// Normalise returns a unit-length copy of v. Used when the embedding provider
// does not guarantee normalised output.
func Normalise(v []float32) []float32 {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	norm := math.Sqrt(sum)
	if norm == 0 {
		return v
	}
	out := make([]float32, len(v))
	for i, x := range v {
		out[i] = float32(float64(x) / norm)
	}
	return out
}
