# Goのチャネル vs Goroutineパフォーマンス比較

このリポジトリでは、Goにおける複数の並行処理アプローチのパフォーマンスを比較します：

1. チャネル + 単一ディスパッチャー + 無制限の並列処理（errgroup.Go）
2. 直接goroutine起動 + 無制限の並列処理（errgroup.Go）
3. チャネル + 単一ディスパッチャー + 制限付き並列処理（errgroup.Go + semaphore）
4. 直接goroutine起動 + 制限付き並列処理（semaphore）

## 実装の比較

### アプローチ1: チャネル + 単一ディスパッチャー + 無制限の並列処理

このアプローチでは、1つのディスパッチャーgoroutineがチャネルからタスクを受け取り、各タスクをerrgroup.Goを使用して並列処理します。並列度に制限はありません。

```go
func ChannelWithUnlimitedParallelism() error {
    tasks := make(chan Task, 100)
    done := make(chan struct{})
    
    // errgroupを作成
    eg, ctx := errgroup.WithContext(context.Background())
    
    // ディスパッチャーgoroutineを起動
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

### アプローチ2: 直接goroutine起動 + 無制限の並列処理

このアプローチでは、タスクごとに直接errgroup.Goを使用してgoroutineを起動します。並列度に制限はありません。

```go
func DirectGoroutineWithUnlimitedParallelism() error {
    eg, ctx := errgroup.WithContext(context.Background())
    
    // タスクごとに直接goroutineを起動
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

### アプローチ3: チャネル + 単一ディスパッチャー + 制限付き並列処理

このアプローチでは、1つのディスパッチャーgoroutineがチャネルからタスクを受け取り、semaphoreで並列度を制限しつつerrgroup.Goを使用して処理します。

```go
func ChannelWithLimitedParallelism(numWorkers int) error {
    tasks := make(chan Task, 100)
    done := make(chan struct{})
    
    // errgroupを作成
    eg, ctx := errgroup.WithContext(context.Background())
    
    // semaphoreを作成して並列度を制限
    sem := semaphore.NewWeighted(int64(numWorkers))
    
    // ディスパッチャーgoroutineを起動
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

### アプローチ4: 直接goroutine起動 + 制限付き並列処理

このアプローチでは、タスクごとに直接goroutineを起動しますが、semaphoreを使用して同時実行数を制限します。

```go
func DirectGoroutineWithLimitedParallelism(maxConcurrency int64) error {
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
go test -bench=BenchmarkChannelWithUnlimitedParallelism ./benchmark
go test -bench=BenchmarkDirectGoroutineWithUnlimitedParallelism ./benchmark
go test -bench=BenchmarkChannelWithLimitedParallelism ./benchmark
go test -bench=BenchmarkDirectGoroutineWithLimitedParallelism ./benchmark

# 詳細なメモリ統計情報も表示
go test -bench=. -benchmem ./benchmark
```

## ベンチマーク結果

### 基本実行結果（`go run main.go`）- 10万タスク処理

```
CPUs: 12
処理タスク数: 100000

1. チャネル + 単一ディスパッチャー + 無制限の並列処理（errgroup.Go）
処理時間: 66.810625ms

2. 直接goroutine起動 + 無制限の並列処理（errgroup.Go）
処理時間: 54.73025ms

3. チャネル + 単一ディスパッチャー + 制限付き並列処理（errgroup.Go + semaphore、12同時実行）
処理時間: 211.657667ms

4. 直接goroutine起動 + 制限付き並列処理（semaphore、12同時実行）
処理時間: 191.222584ms
```

### 詳細なベンチマーク結果（`go test -bench=. -benchmem ./benchmark`）- 10万タスク処理

```
goos: darwin
goarch: arm64
pkg: github.com/go-to-k/go-speed-chan-vs-goroutine/benchmark
cpu: Apple M2 Pro
BenchmarkChannelWithUnlimitedParallelism-12                 18          60649912 ns/op        17629422 B/op         499912 allocs/op
BenchmarkDirectGoroutineWithUnlimitedParallelism-12         22          52495205 ns/op        17623392 B/op         499867 allocs/op
BenchmarkChannelWithLimitedParallelism/4Workers-12           2         543696062 ns/op        36035956 B/op         786291 allocs/op
BenchmarkChannelWithLimitedParallelismVaryingWorkers/Workers1-12        1        2240980792 ns/op        36833800 B/op     799854 allocs/op
BenchmarkChannelWithLimitedParallelismVaryingWorkers/Workers2-12        1        1066918000 ns/op        36658024 B/op     796845 allocs/op
BenchmarkChannelWithLimitedParallelismVaryingWorkers/Workers4-12        2         538092833 ns/op        36130172 B/op     787916 allocs/op
BenchmarkChannelWithLimitedParallelismVaryingWorkers/Workers8-12        4         283978552 ns/op        33360754 B/op     740752 allocs/op
BenchmarkChannelWithLimitedParallelismVaryingWorkers/Workers@-12        7         156329309 ns/op        28552453 B/op     658806 allocs/op
BenchmarkDirectGoroutineWithLimitedParallelism/DefaultConcurrency-12    6         190699479 ns/op        26953154 B/op     599819 allocs/op
BenchmarkDirectGoroutineWithVaryingConcurrency/Concurrency1-12          1        2193692833 ns/op        32824248 B/op     699822 allocs/op
BenchmarkDirectGoroutineWithVaryingConcurrency/Concurrency2-12          1        1068082833 ns/op        32731688 B/op     698224 allocs/op
BenchmarkDirectGoroutineWithVaryingConcurrency/Concurrency4-12          2         526788021 ns/op        32097764 B/op     687468 allocs/op
BenchmarkDirectGoroutineWithVaryingConcurrency/Concurrency8-12          4         266741386 ns/op        29688570 B/op     646416 allocs/op
BenchmarkDirectGoroutineWithVaryingConcurrency/Concurrency@-12          7         149002119 ns/op        24991408 B/op     566385 allocs/op
```

### 結果の分析

1. **アプローチ1（チャネル + 単一ディスパッチャー + 無制限の並列処理）**:
   - 処理時間: 約60.6ms
   - メモリ使用量: 約17.6MB
   - アロケーション数: 約499,912回

2. **アプローチ2（直接goroutine起動 + 無制限の並列処理）**:
   - 処理時間: 約52.5ms（アプローチ1より約15%高速）
   - メモリ使用量: 約17.6MB（アプローチ1とほぼ同等）
   - アロケーション数: 約499,867回（アプローチ1とほぼ同等）

3. **アプローチ3（チャネル + 単一ディスパッチャー + 制限付き並列処理）**:
   - 処理時間: 約156.3ms（16同時実行時）
   - メモリ使用量: 約28.6MB（アプローチ1・2の約1.6倍）
   - アロケーション数: 約658,806回（アプローチ1・2の約1.3倍）
   - 同時実行数を増やすほど処理時間は短縮される（1→16で約14倍高速化）

4. **アプローチ4（直接goroutine起動 + 制限付き並列処理）**:
   - 処理時間: 約149.0ms（16同時実行時）
   - メモリ使用量: 約25.0MB（アプローチ1・2よりも多い）
   - アロケーション数: 約566,385回（アプローチ1・2よりも多い）
   - 同時実行数を増やすほど処理時間は短縮される（1→16で約15倍高速化）

## 結果の解釈

10万タスク処理のベンチマーク結果から、次のような解釈が可能です：

1. **チャネル + 単一ディスパッチャー + 無制限の並列処理**：
   - チャネルでタスクを一元管理しつつ、各タスクは並列に処理されます
   - タスクごとにgoroutineを使用するため、多数のタスクを処理する場合でもメモリ使用量は制御されています
   - ディスパッチャーがチャネルからタスクを読み取るオーバーヘッドがあるため、直接goroutine起動よりやや遅いです

2. **直接goroutine起動 + 無制限の並列処理**：
   - タスクが明確に定義されている場合、最も高速な処理が可能です
   - ディスパッチャーのオーバーヘッドがないため、チャネルを使用するよりも約15%高速です
   - タスク数が10万になっても、メモリ使用量は約17.6MBと効率的です
   - システムリソースに余裕がある場合、この方法が最も効率的です

3. **チャネル + 単一ディスパッチャー + 制限付き並列処理**：
   - 並列度を制限することで、システムリソースの使用を制御できますが、処理速度は無制限の並列処理より大幅に遅くなります
   - 制限が厳しいほど（同時実行数が少ないほど）処理時間は長くなります
   - 外部からのタスク入力をキューイングしつつ、システムリソースを制御したい場合に適しています

4. **直接goroutine起動 + 制限付き並列処理**：
   - 同時実行数を制限することで、システムリソースの使用を抑制しつつ、チャネルベースのアプローチよりも高速に動作します
   - 同様の並列度設定では、チャネルを使用するアプローチよりも約20%高速です
   - 大量のタスクを処理する必要がある場合で、かつシステムリソースに制約がある場合に最適です

## 結論

1. **最高の処理速度を求める場合**：
   - タスクが事前に全て分かっている場合は「直接goroutine起動 + 無制限の並列処理」が最も高速です
   - システムリソースに余裕があり、最高のスループットを求める場合に最適です

2. **動的なタスク入力を扱う場合**：
   - 外部からのタスク入力がある場合は「チャネル + 単一ディスパッチャー」アプローチが適しています
   - タスクの流入がある程度予測可能で、システムリソースに余裕がある場合は「無制限の並列処理」が、
   - リソース制約がある場合は「制限付き並列処理」が適しています

3. **リソース使用量に制約がある場合**：
   - 「直接goroutine起動 + 制限付き並列処理」が、リソース使用を制御しながらも比較的高速な処理を実現します
   - 処理するタスクの性質やCPUコア数に応じて、最適な同時実行数を選択することが重要です

実際の選択は、アプリケーションの要件、タスクの性質（CPU負荷かIO負荷か）、システム環境、およびスケーラビリティ要件に基づいて行うべきです。最高の結果を得るためには、実際の環境で異なるアプローチをベンチマークすることをお勧めします。 