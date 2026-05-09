package spider

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"server/model/system"
)

// TestConcurrentPageSpider_AllPagesProcessed 验证并发分页逻辑会处理所有页码,
// 历史 bug: capacity < MAXGoroutine 时主 goroutine 等待 MAXGoroutine 次 chan 接收 → 死锁.
func TestConcurrentPageSpider_AllPagesProcessed(t *testing.T) {
	cases := []int{1, 3, 10, 25}
	for _, capacity := range cases {
		var visited sync.Map
		var count int64
		fn := func(s *system.FilmSource, hour, pg int) {
			visited.Store(pg, struct{}{})
			atomic.AddInt64(&count, 1)
		}

		done := make(chan struct{})
		go func() {
			defer close(done)
			ConcurrentPageSpider(capacity, &system.FilmSource{}, 0, fn)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatalf("ConcurrentPageSpider deadlocked at capacity=%d", capacity)
		}

		// 所有页都被处理
		require.EqualValues(t, capacity, atomic.LoadInt64(&count), "capacity=%d", capacity)
		for i := 1; i <= capacity; i++ {
			_, ok := visited.Load(i)
			require.True(t, ok, "page %d not visited (capacity=%d)", i, capacity)
		}
	}
}

func TestConcurrentPageSpider_ZeroCapacityNoop(t *testing.T) {
	called := int64(0)
	fn := func(s *system.FilmSource, hour, pg int) {
		atomic.AddInt64(&called, 1)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		ConcurrentPageSpider(0, nil, 0, fn)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("zero capacity should return immediately")
	}
	require.EqualValues(t, 0, atomic.LoadInt64(&called))
}
