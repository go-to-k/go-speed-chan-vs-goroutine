package benchmark

import (
	"runtime"
	"testing"
)

// チャネル + 単一ディスパッチャー + 無制限の並列処理
func BenchmarkChannelWithUnlimitedParallelism(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if err := ChannelWithUnlimitedParallelism(); err != nil {
			b.Fatal(err)
		}
	}
}

// 直接goroutine起動 + 無制限の並列処理
func BenchmarkDirectGoroutineWithUnlimitedParallelism(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if err := DirectGoroutineWithUnlimitedParallelism(); err != nil {
			b.Fatal(err)
		}
	}
}

// チャネル + 単一ディスパッチャー + 制限付き並列処理
func BenchmarkChannelWithLimitedParallelism(b *testing.B) {
	numWorkers := 4 // 固定数のワーカー数

	b.Run("4Workers", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if err := ChannelWithLimitedParallelism(numWorkers); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// 様々なワーカー数でのベンチマーク比較（チャネル + 制限付き並列処理）
func BenchmarkChannelWithLimitedParallelismVaryingWorkers(b *testing.B) {
	workerCounts := []int{1, 2, 4, 8, 16}

	for _, count := range workerCounts {
		b.Run(string("Workers"+string(rune(count+'0'))), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if err := ChannelWithLimitedParallelism(count); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// 直接goroutine起動 + 制限付き並列処理
func BenchmarkDirectGoroutineWithLimitedParallelism(b *testing.B) {
	numWorkers := int64(runtime.NumCPU()) // デフォルトはCPU数

	b.Run("DefaultConcurrency", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if err := DirectGoroutineWithLimitedParallelism(numWorkers); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// 様々な同時実行数でのベンチマーク（直接goroutine起動 + 制限付き並列処理）
func BenchmarkDirectGoroutineWithVaryingConcurrency(b *testing.B) {
	concurrencyCounts := []int64{1, 2, 4, 8, 16}

	for _, count := range concurrencyCounts {
		b.Run(string("Concurrency"+string(rune(count+'0'))), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if err := DirectGoroutineWithLimitedParallelism(count); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
