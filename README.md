# Goのチャネル vs Goroutineパフォーマンス比較

このリポジトリでは、Goにおける複数の並行処理アプローチのパフォーマンスを比較します：

1. 事前に1つのgoroutineを起動してチャネル経由でタスクを受け取り、各タスクをerrgroup.Goで処理する方法
2. ループ内で都度goroutineを起動してタスクを処理する方法（errgroup使用）
3. ワーカープールとチャネルを使った方法（比較用）
4. semaphoreを使用してgoroutineの同時実行数を制限する方法

## 実装の比較

### アプローチ1: シングルワーカーとチャネル（タスク処理はerrgroup.Goで実行）

このアプローチでは、1つのワーカーgoroutineを事前に起動し、チャネルを通じてタスクを受信します。ワーカーはチャネルからタスクを受け取り、各タスクをerrgroup.Goを使用して処理します。

```go
func UsingSingleWorkerWithChannel() error {
    tasks := make(chan Task, 100)
    done := make(chan struct{})
    
    // errgroupを作成
    eg, ctx := errgroup.WithContext(context.Background())
    
    // 1つのワーカーgoroutineを起動
    go func() {
        defer close(done)
        for task := range tasks {
            task := task // ループ変数をキャプチャ
            eg.Go(func() error {
                return processTask(task)
            })
        }
        // すべてのタスク処理が完了するのを待つ
        eg.Wait()
    }()

    // タスクをチャネルに送信
    for i := 0; i < numTasks; i++ {
        tasks <- Task{ID: i}
    }
    
    close(tasks)
    <-done
    return nil
}
```

### アプローチ2: タスクごとにGoroutineを起動

このアプローチでは、タスクごとに新しいgoroutineを起動します。`golang.org/x/sync/errgroup`パッケージを使用して、複数のgoroutineを管理し、エラーハンドリングを容易にします。

```go
func UsingGoroutinePerTask() error {
    eg, ctx := errgroup.WithContext(context.Background())
    
    // タスクごとにgoroutineを起動
    for i := 0; i < numTasks; i++ {
        i := i
        task := Task{ID: i}
        
        eg.Go(func() error {
            return processTask(task)
        })
    }
    
    return eg.Wait()
}
```

### アプローチ3: ワーカープールとチャネル

このアプローチでは、固定数のワーカーgoroutineを事前に起動し、チャネルを通じてタスクを分配します。各ワーカーはチャネルからタスクを受け取り、処理します。

```go
func UsingWorkerPoolWithChannel(numWorkers int) error {
    tasks := make(chan Task, 100)
    var wg sync.WaitGroup
    
    // 複数のワーカーgoroutineを起動
    for w := 0; w < numWorkers; w++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for task := range tasks {
                processTask(task)
            }
        }()
    }

    // タスクをチャネルに送信
    for i := 0; i < numTasks; i++ {
        tasks <- Task{ID: i}
    }
    
    close(tasks)
    wg.Wait()
    return nil
}
```

### アプローチ4: Semaphoreを使用した同時実行制限

このアプローチでは、`golang.org/x/sync/semaphore`パッケージを使用して、同時に実行されるgoroutineの数を制限します。タスクごとに新しいgoroutineを起動しつつも、システムリソースの使用を制御できます。

```go
func UsingSemaphoreWithGoroutines(maxConcurrency int64) error {
    ctx := context.Background()
    
    // 同時実行数を制限するsemaphoreを作成
    sem := semaphore.NewWeighted(maxConcurrency)
    
    var wg sync.WaitGroup
    
    // タスクごとにgoroutineを起動（semaphoreで同時実行数を制限）
    for i := 0; i < numTasks; i++ {
        i := i
        task := Task{ID: i}
        
        // semaphoreの空きを待つ
        if err := sem.Acquire(ctx, 1); err != nil {
            return err
        }
        
        wg.Add(1)
        go func() {
            defer sem.Release(1)
            defer wg.Done()
            processTask(task)
        }()
    }
    
    wg.Wait()
    return nil
}
```

## 使用方法

### 通常の実行

```bash
go run main.go
```

これにより、各アプローチの実行時間が出力されます。

### ベンチマークの実行

より正確な測定のために、Go標準のベンチマーク機能を使用できます：

```bash
# 全てのベンチマークを実行
go test -bench=. ./benchmark

# 特定のベンチマークを実行
go test -bench=BenchmarkSingleWorkerWithChannel ./benchmark
go test -bench=BenchmarkGoroutinePerTask ./benchmark
go test -bench=BenchmarkWorkerPoolWithChannel ./benchmark
go test -bench=BenchmarkSemaphoreWithGoroutines ./benchmark

# 詳細なメモリ統計情報も表示
go test -bench=. -benchmem ./benchmark
```

## ベンチマーク結果

### 基本実行結果（`go run main.go`）

```
CPUs: 12
処理タスク数: 10000

1. 事前に1つのgoroutineを起動してチャネル経由でタスクを受け取り、errgroup.Goで並列処理
処理時間: 7.516ms

2. ループ内でgoroutineを毎回起動
処理時間: 6.686084ms

3. 12個のワーカープールを使用したチャネル実装
処理時間: 18.977ms

4. semaphoreを使用して同時実行数を12に制限
処理時間: 19.860708ms
```

### 詳細なベンチマーク結果（`go test -bench=. -benchmem ./benchmark`）

```
goos: darwin
goarch: arm64
pkg: github.com/go-to-k/go-speed-chan-vs-goroutine/benchmark
cpu: Apple M2 Pro
BenchmarkSingleWorkerWithChannel-12                          195           6118184 ns/op         1763846 B/op          49767 allocs/op
BenchmarkGoroutinePerTask-12                                 223           4959157 ns/op         1760949 B/op          49762 allocs/op
BenchmarkWorkerPoolWithChannel/4Workers-12                    22          50556850 ns/op          241410 B/op          19755 allocs/op
BenchmarkWorkerPoolWithChannelVaryingWorkers/Workers1-12       5         211595183 ns/op          241235 B/op          19749 allocs/op
BenchmarkWorkerPoolWithChannelVaryingWorkers/Workers2-12      10         102134692 ns/op          241180 B/op          19751 allocs/op
BenchmarkWorkerPoolWithChannelVaryingWorkers/Workers4-12      21          50768460 ns/op          241432 B/op          19755 allocs/op
BenchmarkWorkerPoolWithChannelVaryingWorkers/Workers8-12      43          24966724 ns/op          241811 B/op          19763 allocs/op
BenchmarkWorkerPoolWithChannelVaryingWorkers/Workers@-12      88          14955557 ns/op          242674 B/op          19780 allocs/op
BenchmarkSemaphoreWithGoroutines/DefaultConcurrency-12                61          18934442 ns/op         2698816 B/op          59839 allocs/op
BenchmarkSemaphoreWithVaryingConcurrency/Concurrency1-12               5         218344800 ns/op         3280139 B/op          69749 allocs/op
BenchmarkSemaphoreWithVaryingConcurrency/Concurrency2-12              10         108389938 ns/op         3269420 B/op          69565 allocs/op
BenchmarkSemaphoreWithVaryingConcurrency/Concurrency4-12              22          52402504 ns/op         3210457 B/op          68563 allocs/op
BenchmarkSemaphoreWithVaryingConcurrency/Concurrency8-12              42          27283199 ns/op         2955147 B/op          64212 allocs/op
BenchmarkSemaphoreWithVaryingConcurrency/Concurrency@-12              82          15012667 ns/op         2514693 B/op          56709 allocs/op
```

### 結果の分析

1. **アプローチ1（シングルワーカー + チャネル + errgroup.Go）**:
   - 処理時間: 約6.1ms
   - メモリ使用量: 約1.76MB
   - アロケーション数: 約49,767回

2. **アプローチ2（タスクごとにgoroutine）**:
   - 処理時間: 約5.0ms（アプローチ1より約20%高速）
   - メモリ使用量: 約1.76MB（アプローチ1とほぼ同等）
   - アロケーション数: 約49,762回（アプローチ1とほぼ同等）

3. **ワーカープール方式（4ワーカー）**:
   - 処理時間: 約50.6ms（アプローチ1の約8倍、アプローチ2の約10倍遅い）
   - メモリ使用量: 約241KB（アプローチ1・2の約14%程度）
   - アロケーション数: 約19,755回（アプローチ1・2の約40%程度）

4. **アプローチ4（Semaphoreによる同時実行制限）**:
   - 処理時間: 約19ms（同時実行数12の場合）
   - メモリ使用量: 約2.7MB（アプローチ1・2よりも多い）
   - アロケーション数: 約59,839回（アプローチ1・2よりも多い）
   - 同時実行数を増やすほど処理時間は短縮され、メモリ使用量とアロケーション数は減少する傾向

## 結果の解釈

ベンチマーク結果はシステム環境によって異なりますが、一般的には：

1. **シングルワーカー+チャネル（errgroup.Go使用）**：
   - チャネルでタスクを一元管理しつつ、各タスクは並列に処理されます
   - タスクごとにgoroutineを使用するため、メモリ使用量は増加しますが、処理速度は大幅に向上します
   - タスクキューの制御とタスク処理の並列化を両立させたい場合に適しています

2. **タスクごとのgoroutine**：
   - 各タスクが独立して並列実行されるため、高速な処理が可能です
   - シングルワーカー+チャネル（errgroup.Go使用）とほぼ同等のパフォーマンスを示します
   - コードがシンプルで、タスクキューの管理が不要な場合に適しています

3. **ワーカープール+チャネル**（比較用実装）：
   - goroutineの数を制限できるため、システムリソースの使用を抑えられます
   - メモリ効率は良好ですが、他の2つのアプローチと比較すると処理速度は遅くなります
   - 大量のタスクを処理しながらもシステムリソースの使用を制限したい場合に適しています

4. **Semaphoreによる同時実行制限**：
   - goroutineの数を制限でき、システムリソースの使用を抑制できます
   - ワーカープール方式とほぼ同等の処理速度ですが、実装がよりシンプルです
   - タスクごとにgoroutineを作成するため、ワーカープール方式よりもメモリ使用量が多くなります
   - 大量のタスクをキューイングしつつ処理のスループットを制御したい場合に適しています

実際の利用シナリオ、タスクの性質（CPU負荷かIO負荷か）、およびシステムのリソース制約に合わせて適切なアプローチを選択してください。 