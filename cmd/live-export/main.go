package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path"
	"time"

	xivmarketgo "github.com/t-arik/xiv-market-go"
)

func main() {
	if err := run(); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := setupCtx()
	defer cancel(nil)

	region, outDir, err := parseFlags()
	if err != nil {
		return err
	}

	buf := &queue{}

	client := xivmarketgo.DefaultRestClient()

	recent := make(chan xivmarketgo.WorldItemRecency, 4*1024)

	go func() {
		if err := client.StreamMostRecentlyUpdatedItems(ctx, "", region, recent); err != nil {
			slog.Error("error streaming most recently updated items", "err", err)
			cancel(err)
		}
	}()

	var outFile *os.File
	batchSize := maxBatchSize

	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case item := <-recent:
			buf.Add(item)
		default:
			if buf.Len() == 0 {
				time.Sleep(time.Second)
				continue
			}

			batch := buf.Batch(batchSize)

			ids := batchItemIDs(batch)

			items, err := client.MarketBoardCurrentData(ctx, ids, batch[0].WorldName, 5)
			if err != nil {
				batchSize = calcBatchSizeForFailure(batchSize)
				slog.Error("error fetching market board data",
					"err", err,
					"attempted_items", len(batch),
					"next_batch_limit", batchSize,
					"retry_in", time.Second,
				)

				timer := time.NewTimer(time.Second)
				select {
				case <-ctx.Done():
					timer.Stop()
					return context.Cause(ctx)
				case <-timer.C:
				}

				continue
			}

			outFile, err = getOutFile(time.Now().UTC(), outDir, outFile)
			if err != nil {
				return fmt.Errorf("failed to get output file: %w", err)
			}

			encoder := json.NewEncoder(outFile)
			for _, item := range items {
				if err := encoder.Encode(item); err != nil {
					return fmt.Errorf("failed to encode item to output file: %w", err)
				}
			}

			if err := outFile.Sync(); err != nil {
				return fmt.Errorf("failed to sync output file: %w", err)
			}

			if len(items) != len(batch) {
				slog.Warn("mismatch between batch and items",
					"batch", batch,
					"items", items,
				)
			}

			if len(batch) == batchSize {
				batchSize = calcBatchSizeForSuccess(batchSize)
			}

			slog.Info("processed batch",
				"world", batch[0].WorldName,
				"count", len(batch),
				"items", len(items),
				"next_batch_limit", batchSize,
				"buf", buf.Len(),
			)
			buf.Remove(batch)
		}
	}
}

const (
	minBatchSize = 1
	maxBatchSize = 100
)

func calcBatchSizeForFailure(size int) int {
	return max(minBatchSize, size/2)
}

func calcBatchSizeForSuccess(size int) int {
	return min(maxBatchSize, size+1)
}

func getOutFile(now time.Time, outDir string, outFile *os.File) (*os.File, error) {
	filePath := path.Join(outDir, now.Format("2006-01-02")+".jsonl")

	if outFile == nil || outFile.Name() != filePath {
		if outFile != nil {
			if err := outFile.Close(); err != nil {
				return nil, fmt.Errorf("failed to close previous output file: %w", err)
			}
		}

		f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open output file %s: %w", filePath, err)
		}

		return f, nil
	}

	return outFile, nil
}

func parseFlags() (string, string, error) {
	region := flag.String("region", "", "")
	outDir := flag.String("output-dir", "", "")

	flag.Parse()

	if *region == "" || *outDir == "" {
		flag.Usage()
		return "", "", errors.New("region and output-dir flags are required")
	}
	return *region, *outDir, nil
}

func setupCtx() (context.Context, context.CancelCauseFunc) {
	signalCtx, signalCancel := signal.NotifyContext(context.Background(), os.Interrupt)
	ctx, cancel := context.WithCancelCause(signalCtx)

	return ctx, func(cause error) {
		cancel(cause)
		signalCancel()
	}
}

func batchItemIDs(batch []xivmarketgo.WorldItemRecency) []int {
	ids := []int{}
	for _, item := range batch {
		ids = append(ids, int(item.ItemId))
	}
	return ids
}
