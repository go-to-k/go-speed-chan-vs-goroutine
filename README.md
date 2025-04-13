# Goのチャネル vs Goroutineパフォーマンス比較

このリポジトリでは、Goにおける2つの並行処理アプローチのパフォーマンスを比較します：

1. 事前に1つのgoroutineを起動してチャネル経由でタスクを受け取り、各タスクをerrgroup.Goで処理する方法
2. ループ内で都度goroutineを起動してタスクを処理する方法（errgroup使用）

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

# 詳細なメモリ統計情報も表示
go test -bench=. -benchmem ./benchmark
```

## 結果の解釈

ベンチマーク結果はシステム環境によって異なりますが、一般的には：

1. **シングルワーカー+チャネル**：goroutineの作成オーバーヘッドが少なく、小さなタスクの大量処理に適しています。
2. **タスクごとのgoroutine**：大量のgoroutine作成によるオーバーヘッドがありますが、各タスクが独立して並列実行されるため、重いタスクの処理に適しています。

実際の利用シナリオに合わせて適切なアプローチを選択してください。 