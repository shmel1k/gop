package gop

import "testing"

func TestNeedAdditionalWorker(t *testing.T) {
	var testData = []struct {
		poolSize    int32
		maxPoolSize int
		percent     int
		expected    bool
	}{
		{
			poolSize:    10,
			maxPoolSize: 15,
			percent:     50,
			expected:    true,
		},
		{
			poolSize:    10,
			maxPoolSize: 0,
			percent:     100500,
			expected:    false,
		},
		{
			poolSize:    10,
			maxPoolSize: 100,
			percent:     80,
			expected:    false,
		},
	}

	for i, v := range testData {
		got := needAdditionalWorker(v.poolSize, v.maxPoolSize, v.percent)
		if got != v.expected {
			t.Errorf("TestNeedAdditionalWorker[%d]: got %v, expected %v", i, got, v.expected)
		}
	}
}

func BenchmarkNeedAdditionalWorker(b *testing.B) {
	currentPoolSize := int32(42)
	maxPoolSize := int(80)
	percent := int(90)

	for i := 0; i < b.N; i++ {
		needAdditionalWorker(currentPoolSize, maxPoolSize, percent)
	}
}
