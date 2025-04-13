package benchmark

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

// 処理するタスクの数
const numTasks = 100000

// タスクを模擬する構造体
type Task struct {
	ID   int
	Data string
}

// タスクを処理する関数（タスクIDによって処理時間を変えることができる）
func processTask(task Task) error {
	// シミュレートされた処理時間
	// タスクのIDによって処理時間を可変にする（より現実的なワークロード）
	processingTime := 10 * time.Microsecond
	if task.ID%10 == 0 {
		// 10個に1つは少し重いタスク
		processingTime = 50 * time.Microsecond
	}
	if task.ID%100 == 0 {
		// 100個に1つはさらに重いタスク
		processingTime = 200 * time.Microsecond
	}

	time.Sleep(processingTime)
	return nil
}

// チャネルを使用した実装：1つのgoroutineを事前に起動
func ChannelWithUnlimitedParallelism() error {
	tasks := make(chan Task, 100)
	done := make(chan struct{})

	// errgroupを作成
	eg, ctx := errgroup.WithContext(context.Background())

	// ワーカーgoroutineを一つ起動
	go func() {
		defer close(done)
		for task := range tasks {
			task := task // ループ変数をキャプチャ
			// errgroup.Goを使用してタスク処理を実行
			eg.Go(func() error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					if err := processTask(task); err != nil {
						log.Printf("Error processing task %d: %v", task.ID, err)
					}
					return nil
				}
			})
		}

		// すべてのタスク処理が完了するのを待つ
		if err := eg.Wait(); err != nil {
			log.Printf("Error in worker: %v", err)
		}
	}()

	// タスクをチャネルに送信
	for i := 0; i < numTasks; i++ {
		task := Task{
			ID:   i,
			Data: fmt.Sprintf("Task data %d", i),
		}
		tasks <- task
	}

	// タスクの送信が終了したらチャネルを閉じる
	close(tasks)

	// ワーカーの終了を待つ
	<-done
	return nil
}

// goroutineをループ内で起動する実装
func DirectGoroutineWithUnlimitedParallelism() error {
	// errgroupでgoroutineの実行を管理
	eg, ctx := errgroup.WithContext(context.Background())

	// タスクごとにgoroutineを起動
	for i := 0; i < numTasks; i++ {
		i := i // ループ変数をキャプチャ
		task := Task{
			ID:   i,
			Data: fmt.Sprintf("Task data %d", i),
		}

		eg.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return processTask(task)
			}
		})
	}

	// すべてのgoroutineの終了を待つ
	return eg.Wait()
}

// 複数のワーカーを使用するチャネル実装（比較用）
func ChannelWithLimitedParallelism(numWorkers int) error {
	tasks := make(chan Task, 100)
	done := make(chan struct{})

	// errgroupを作成
	eg, ctx := errgroup.WithContext(context.Background())

	// semaphoreを作成して並列度を制限
	sem := semaphore.NewWeighted(int64(numWorkers))

	// ディスパッチャーgoroutineを一つ起動
	go func() {
		defer close(done)
		for task := range tasks {
			task := task // ループ変数をキャプチャ

			// semaphoreの空きを待つ
			if err := sem.Acquire(ctx, 1); err != nil {
				log.Printf("Failed to acquire semaphore: %v", err)
				continue
			}

			// errgroup.Goを使用してタスク処理を実行（semaphoreで制限）
			eg.Go(func() error {
				defer sem.Release(1) // 処理完了時にsemaphoreを解放

				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					if err := processTask(task); err != nil {
						log.Printf("Error processing task %d: %v", task.ID, err)
					}
					return nil
				}
			})
		}

		// すべてのタスク処理が完了するのを待つ
		if err := eg.Wait(); err != nil {
			log.Printf("Error in worker: %v", err)
		}
	}()

	// タスクをチャネルに送信
	for i := 0; i < numTasks; i++ {
		task := Task{
			ID:   i,
			Data: fmt.Sprintf("Task data %d", i),
		}
		tasks <- task
	}

	// タスクの送信が終了したらチャネルを閉じる
	close(tasks)

	// ディスパッチャーの終了を待つ
	<-done
	return nil
}

// semaphoreを使用してgoroutineの同時実行数を制限する実装
func DirectGoroutineWithLimitedParallelism(maxConcurrency int64) error {
	// コンテキストを作成
	ctx := context.Background()

	// 同時実行数を制限するsemaphoreを作成
	sem := semaphore.NewWeighted(maxConcurrency)

	// 完了を待つためのWaitGroup
	var wg sync.WaitGroup

	// タスクごとにgoroutineを起動（semaphoreで同時実行数を制限）
	for i := 0; i < numTasks; i++ {
		i := i // ループ変数をキャプチャ
		task := Task{
			ID:   i,
			Data: fmt.Sprintf("Task data %d", i),
		}

		// semaphoreの空きを待つ
		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}

		wg.Add(1)
		go func() {
			defer sem.Release(1)
			defer wg.Done()

			if err := processTask(task); err != nil {
				log.Printf("Error processing task %d: %v", task.ID, err)
			}
		}()
	}

	// すべてのgoroutineの終了を待つ
	wg.Wait()
	return nil
}

// ベンチマークを実行する関数
func Run() error {
	fmt.Printf("CPUs: %d\n", runtime.NumCPU())
	fmt.Printf("処理タスク数: %d\n\n", numTasks)

	fmt.Println("1. チャネル + 単一ディスパッチャー + 無制限の並列処理（errgroup.Go）")
	start := time.Now()
	if err := ChannelWithUnlimitedParallelism(); err != nil {
		return err
	}
	fmt.Printf("処理時間: %v\n\n", time.Since(start))

	fmt.Println("2. 直接goroutine起動 + 無制限の並列処理（errgroup.Go）")
	start = time.Now()
	if err := DirectGoroutineWithUnlimitedParallelism(); err != nil {
		return err
	}
	fmt.Printf("処理時間: %v\n\n", time.Since(start))

	// 比較のために複数ワーカーのチャネル実装も実行
	numWorkers := runtime.NumCPU()
	fmt.Printf("3. チャネル + 単一ディスパッチャー + 制限付き並列処理（errgroup.Go + semaphore、%d同時実行）\n", numWorkers)
	start = time.Now()
	if err := ChannelWithLimitedParallelism(numWorkers); err != nil {
		return err
	}
	fmt.Printf("処理時間: %v\n\n", time.Since(start))

	// 4つ目のアプローチ：semaphoreを使用した実装
	fmt.Printf("4. 直接goroutine起動 + 制限付き並列処理（semaphore、%d同時実行）\n", numWorkers)
	start = time.Now()
	if err := DirectGoroutineWithLimitedParallelism(int64(numWorkers)); err != nil {
		return err
	}
	fmt.Printf("処理時間: %v\n\n", time.Since(start))

	return nil
}
