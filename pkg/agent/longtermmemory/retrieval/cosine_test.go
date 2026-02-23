package retrieval

import (
	"math"
	"testing"
)

func TestDotProduct(t *testing.T) {
	tests := []struct {
		name string
		a, b []float32
		want float64
	}{
		{
			name: "identical unit vectors",
			a:    []float32{1, 0, 0},
			b:    []float32{1, 0, 0},
			want: 1.0,
		},
		{
			name: "orthogonal vectors",
			a:    []float32{1, 0},
			b:    []float32{0, 1},
			want: 0.0,
		},
		{
			name: "general case",
			a:    []float32{1, 2, 3},
			b:    []float32{4, 5, 6},
			want: 32.0, // 1*4 + 2*5 + 3*6
		},
		{
			name: "b shorter than a uses min length",
			a:    []float32{1, 2, 3},
			b:    []float32{1, 1},
			want: 3.0, // 1*1 + 2*1 only
		},
		{
			name: "empty vectors",
			a:    []float32{},
			b:    []float32{},
			want: 0.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := dotProduct(tc.a, tc.b)
			if math.Abs(got-tc.want) > 1e-6 {
				t.Errorf("dotProduct() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNormalise(t *testing.T) {
	t.Run("zero vector returns original", func(t *testing.T) {
		v := []float32{0, 0, 0}
		got := Normalise(v)
		if got[0] != 0 || got[1] != 0 || got[2] != 0 {
			t.Errorf("expected zero vector back, got %v", got)
		}
	})

	t.Run("already unit vector unchanged in magnitude", func(t *testing.T) {
		v := []float32{1, 0, 0}
		got := Normalise(v)
		mag := math.Sqrt(float64(got[0]*got[0] + got[1]*got[1] + got[2]*got[2]))
		if math.Abs(mag-1.0) > 1e-6 {
			t.Errorf("magnitude = %v, want 1.0", mag)
		}
	})

	t.Run("general vector has unit length after normalise", func(t *testing.T) {
		v := []float32{3, 4} // magnitude = 5
		got := Normalise(v)
		mag := math.Sqrt(float64(got[0]*got[0] + got[1]*got[1]))
		if math.Abs(mag-1.0) > 1e-6 {
			t.Errorf("magnitude = %v, want 1.0", mag)
		}
		// 3/5 = 0.6, 4/5 = 0.8
		if math.Abs(float64(got[0])-0.6) > 1e-6 {
			t.Errorf("got[0] = %v, want 0.6", got[0])
		}
		if math.Abs(float64(got[1])-0.8) > 1e-6 {
			t.Errorf("got[1] = %v, want 0.8", got[1])
		}
	})

	t.Run("returns a copy, does not modify input", func(t *testing.T) {
		v := []float32{3, 4}
		_ = Normalise(v)
		if v[0] != 3 || v[1] != 4 {
			t.Error("Normalise modified the input slice")
		}
	})
}

func TestCosineTopK(t *testing.T) {
	makeVec := func(x, y float32) []float32 { return Normalise([]float32{x, y}) }

	entries := []MemoryVector{
		{Memory: makeMemoryFile("a", "alpha", ""), Vector: makeVec(1, 0)},
		{Memory: makeMemoryFile("b", "beta", ""), Vector: makeVec(0, 1)},
		{Memory: makeMemoryFile("c", "gamma", ""), Vector: makeVec(1, 1)},
	}

	t.Run("k larger than entries returns all", func(t *testing.T) {
		query := makeVec(1, 0)
		got := cosineTopK(entries, query, 10)
		if len(got) != 3 {
			t.Errorf("len = %d, want 3", len(got))
		}
	})

	t.Run("returns top k by cosine similarity", func(t *testing.T) {
		query := makeVec(1, 0) // closest to "a"
		got := cosineTopK(entries, query, 2)
		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}
		// "a" should be first (dot product = 1.0), "c" second
		if got[0].Memory.Meta.ID != "a" {
			t.Errorf("first result ID = %q, want %q", got[0].Memory.Meta.ID, "a")
		}
	})

	t.Run("empty entries returns empty", func(t *testing.T) {
		got := cosineTopK(nil, makeVec(1, 0), 5)
		if len(got) != 0 {
			t.Errorf("expected empty, got %v", got)
		}
	})

	t.Run("k=0 returns empty", func(t *testing.T) {
		got := cosineTopK(entries, makeVec(1, 0), 0)
		if len(got) != 0 {
			t.Errorf("expected empty, got %d entries", len(got))
		}
	})
}
