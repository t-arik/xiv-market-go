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

	client := xivmarketgo.DefaultRestClient()

	itemIDs, err := client.MarketableItems(ctx)
	if err != nil {
		return err
	}

	retryCount := 0

	for i, itemId := range itemIDs {
		select {
		case <-ctx.Done():
			return nil
		default:
		retry:
			for {
				ids := []int{itemId}

				items, err := client.MarketBoardCurrentData(ctx, ids, *region, 50_000)
				if errors.Is(err, context.Canceled) {
					return nil
				}

				if err != nil {
					retryCount++
					slog.Error("error fetching market board data", "err", err, "retryCount", retryCount)

					if retryCount > 60 {
						return fmt.Errorf("too many errors fetching market board data: %w", err)
					}

					time.Sleep(time.Second)

					continue retry
				}

				retryCount = 0

				for _, item := range items {
					timestamp := time.Now().UTC().Format("2006-01-02T150405Z")
					name := fmt.Sprintf("%s_%d_%d_%s.json", timestamp, item.LastUploadTime, item.ItemId, *region)
					file := path.Join(*outDir, name)

					f, err := os.Create(file)
					if err != nil {
						return fmt.Errorf("error creating file %s: %w", file, err)
					}

					if err := json.NewEncoder(f).Encode(item); err != nil {
						return fmt.Errorf("error encoding item to JSON: %w", err)
					}

					if err := f.Close(); err != nil {
						return fmt.Errorf("error closing file: %w", err)
					}
				}

				slog.Info("fetched market board data",
					"completed", i+1,
					"total", len(itemIDs),
					"progress", fmt.Sprintf("%.2f%%", float64(i+1)/float64(len(itemIDs))*100),
				)

				break retry
			}
		}
	}

	return nil
}
