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
		panic(err)
	}
}

func run() error {
	signalCtx, signalCancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer signalCancel()

	ctx, cancel := context.WithCancelCause(signalCtx)
	defer cancel(nil)

	region := flag.String("region", "", "")
	outDir := flag.String("output-dir", "", "")

	flag.Parse()

	if *region == "" || *outDir == "" {
		flag.Usage()
		return nil
	}

	buf := &queue{}

	client := xivmarketgo.DefaultRestClient()

	recent := make(chan xivmarketgo.WorldItemRecency, 8*1024)

	go func() {
		if err := client.StreamMostRecentlyUpdatedItems(ctx, "", *region, recent); err != nil {
			slog.Error("error streaming most recently updated items", "err", err)
			cancel(err)
		}
	}()

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

			batch := buf.Batch()

			ids := []int{}
			for _, item := range batch {
				ids = append(ids, int(item.ItemId))
			}

			items, err := client.MarketBoardCurrentData(ctx, ids, batch[0].WorldName, 1000)
			if err != nil {
				slog.ErrorContext(ctx, "error fetching market board data", "err", err)
				time.Sleep(time.Second)

				continue
			}

			for _, item := range items {
				timestamp := time.Now().UTC().Format("2006-01-02T150405Z")
				name := fmt.Sprintf("%s_%d_%d_%s.json", timestamp, item.LastUploadTime, item.ItemId, item.WorldName)
				file := path.Join(*outDir, name)

				f, err := os.Create(file)
				if err != nil {
					return fmt.Errorf("failed to create file %s: %w", file, err)
				}

				if err := json.NewEncoder(f).Encode(item); err != nil {
					return fmt.Errorf("failed to encode item to file %s: %w", file, err)
				}

				if err := f.Close(); err != nil {
					return fmt.Errorf("failed to close file %s: %w", file, err)
				}
			}

			if len(items) != len(batch) {
				slog.WarnContext(ctx, "mismatch between batch and items",
					"batch", batch,
					"items", items,
				)
			}

			slog.InfoContext(ctx, "processed batch",
				"world", batch[0].WorldName,
				"count", len(batch),
				"items", len(items),
				"buf", buf.Len(),
			)
			buf.Remove(batch)
		}
	}
}
