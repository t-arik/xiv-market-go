package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"math/rand"
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
	outDir := flag.String("outputDir", "", "")

	flag.Parse()

	if *region == "" || *outDir == "" {
		flag.Usage()
		return nil
	}

	worlds, ok := xivmarketgo.WorldsByRegion[*region]
	if !ok {
		return fmt.Errorf("unknown region %s", *region)
	}

	client := xivmarketgo.DefaultRestClient()

	itemIDs, err := client.MarketableItems(ctx)
	if err != nil {
		return err
	}

	rand.Shuffle(len(worlds), func(i, j int) {
		worlds[i], worlds[j] = worlds[j], worlds[i]
	})

	rand.Shuffle(len(itemIDs), func(i, j int) {
		itemIDs[i], itemIDs[j] = itemIDs[j], itemIDs[i]
	})

	total := len(worlds) * len(itemIDs)

	backoff := time.Duration(time.Second)

	completed := 0

	chunkSize := 50

	for _, world := range worlds {
		for i := 0; i < len(itemIDs); i += chunkSize {
			select {
			case <-ctx.Done():
				return nil
			default:
			retry:
				for {
					chunkSize = min(chunkSize+8, 100)

					ids := itemIDs[i:min(i+chunkSize, len(itemIDs))]

					items, err := client.MarketBoardCurrentData(ctx, ids, world)
					if errors.Is(err, context.Canceled) {
						return nil
					}

					if err != nil {
						slog.ErrorContext(ctx, "error fetching market board data", "err", err)

						chunkSize = max(1, chunkSize/2)

						time.Sleep(backoff)
						backoff += time.Second

						continue retry
					}

					backoff = min(time.Second)

					for _, item := range items {
						name := fmt.Sprintf("%d-%d-%s.json", item.LastUploadTime, item.ItemId, item.WorldName)
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

						completed += 1
					}

					slog.InfoContext(ctx, "fetched market board data",
						"world", world,
						"batch", len(ids),
						"count", len(items),
						"completed", completed,
						"total", total,
						"progress", fmt.Sprintf("%.2f%%", float64(completed)/float64(total)*100),
					)

					break retry
				}
			}
		}
	}

	return nil
}
