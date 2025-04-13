package benchmark

import (
	"runtime"
	"testing"
)

// シングルワーカーとチャネルを使用したベンチマーク（タスク処理はerrgroup.Goで実行）
func BenchmarkSingleWorkerWithChannel(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if err := UsingSingleWorkerWithChannel(); err != nil {
			b.Fatal(err)
		}
	}
}

// タスクごとにgoroutineを起動するベンチマーク
func BenchmarkGoroutinePerTask(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if err := UsingGoroutinePerTask(); err != nil {
			b.Fatal(err)
		}
	}
}

// 比較用：複数ワーカーとチャネルを使用したベンチマーク
func BenchmarkWorkerPoolWithChannel(b *testing.B) {
	numWorkers := 4 // 固定数のワーカー数

	b.Run("4Workers", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if err := UsingWorkerPoolWithChannel(numWorkers); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// 様々なワーカー数でのベンチマーク比較
func BenchmarkWorkerPoolWithChannelVaryingWorkers(b *testing.B) {
	workerCounts := []int{1, 2, 4, 8, 16}

	for _, count := range workerCounts {
		b.Run(string("Workers"+string(rune(count+'0'))), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if err := UsingWorkerPoolWithChannel(count); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// semaphoreを使用したベンチマーク
func BenchmarkSemaphoreWithGoroutines(b *testing.B) {
	numWorkers := int64(runtime.NumCPU()) // デフォルトはCPU数

	b.Run("DefaultConcurrency", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if err := UsingSemaphoreWithGoroutines(numWorkers); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// 様々な同時実行数でのsemaphoreベンチマーク
func BenchmarkSemaphoreWithVaryingConcurrency(b *testing.B) {
	concurrencyCounts := []int64{1, 2, 4, 8, 16}

	for _, count := range concurrencyCounts {
		b.Run(string("Concurrency"+string(rune(count+'0'))), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if err := UsingSemaphoreWithGoroutines(count); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
