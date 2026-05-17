package spider

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidSpec_Valid(t *testing.T) {
	cases := []string{
		"0 */20 * * * ?",
		"0 0 0 * * *",
		"30 0 0 * * *",
	}
	for _, s := range cases {
		require.NoError(t, ValidSpec(s), "expect valid: %q", s)
	}
}

func TestValidSpec_Invalid(t *testing.T) {
	cases := []string{
		"",
		"not-cron",
		"0 0",
		"99 0 0 * * *", // seconds field 越界
	}
	for _, s := range cases {
		require.Error(t, ValidSpec(s), "expect invalid: %q", s)
	}
}

// TestTryAcquireCronLock_Reentry 验证重入锁的核心语义:
//  1. 同一 id 第一次抢锁成功 (返回 release 不为 nil)
//  2. 在 release 之前再次抢同一 id 失败 (返回 nil), 防止长任务与下次 cron 触发叠加
//  3. release 后同一 id 可再次抢到
//  4. 不同 id 互不影响
func TestTryAcquireCronLock_Reentry(t *testing.T) {
	resetCronLocks := func() {
		cronTaskMu.Lock()
		cronTaskRunning = make(map[string]bool)
		cronTaskMu.Unlock()
	}
	resetCronLocks()
	defer resetCronLocks()

	// 1. 第一次抢锁成功
	release := tryAcquireCronLock("taskA")
	if release == nil {
		t.Fatal("first acquire should succeed")
	}

	// 2. 未释放前再次抢同一 id 失败
	if r := tryAcquireCronLock("taskA"); r != nil {
		t.Fatal("reentry on same id while running must return nil")
	}

	// 4. 不同 id 不受影响
	releaseB := tryAcquireCronLock("taskB")
	if releaseB == nil {
		t.Fatal("acquire on different id should succeed")
	}
	releaseB()

	// 3. release 后同一 id 可再次抢到
	release()
	release2 := tryAcquireCronLock("taskA")
	if release2 == nil {
		t.Fatal("acquire after release should succeed")
	}
	release2()
}

// TestTryAcquireCronLock_ConcurrentRace 并发抢锁: N 个 goroutine 抢同一 id, 只能有一个赢.
// 防止 sync.Mutex 内置的检查后写入序列性退化, 也防止重入锁的 Time-Of-Check-Time-Of-Use 漏洞.
func TestTryAcquireCronLock_ConcurrentRace(t *testing.T) {
	cronTaskMu.Lock()
	cronTaskRunning = make(map[string]bool)
	cronTaskMu.Unlock()
	t.Cleanup(func() {
		cronTaskMu.Lock()
		cronTaskRunning = make(map[string]bool)
		cronTaskMu.Unlock()
	})

	const N = 50
	var (
		wg   sync.WaitGroup
		wins int64
	)
	start := make(chan struct{})
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			<-start
			if r := tryAcquireCronLock("hot"); r != nil {
				atomic.AddInt64(&wins, 1)
				// 不立即释放, 模拟长任务运行期内的争抢
				time.Sleep(5 * time.Millisecond)
				r()
			}
		}()
	}
	close(start)
	wg.Wait()

	if wins != 1 {
		t.Fatalf("exactly 1 goroutine should win the lock, got %d", wins)
	}
}
