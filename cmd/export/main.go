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
	"slices"
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

	recent := make(chan xivmarketgo.WorldItemRecency, 8*1024)
	current := make(chan xivmarketgo.CurrentlyShown, 1024)

	go streamRecent(ctx, client, *region, recent, cancel)
	go processRecent(ctx, client, recent, current)
	go fetchInitial(ctx, client, worlds, itemIDs, current)

	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case item := <-current:
			name := fmt.Sprintf("%d-%d-%s.json", item.LastUploadTime, item.ItemId, item.WorldName)
			file := path.Join(*outDir, name)

			f, err := os.Create(file)
			if err != nil {
				return err
			}

			if err := json.NewEncoder(f).Encode(item); err != nil {
				return err
			}

			if err := f.Close(); err != nil {
				return err
			}
		}
	}
}

func streamRecent(
	ctx context.Context,
	client *xivmarketgo.Client,
	region string,
	recent chan xivmarketgo.WorldItemRecency,
	cancel context.CancelCauseFunc,
) {
	slog.InfoContext(ctx, "goroutine", "name", "StreamMostRecentlyUpdatedItems")

	if err := client.StreamMostRecentlyUpdatedItems(ctx, "", region, recent); err != nil {
		cancel(err)
	}
}

func processRecent(
	ctx context.Context,
	client *xivmarketgo.Client,
	recent chan xivmarketgo.WorldItemRecency,
	current chan xivmarketgo.CurrentlyShown,
) {
	slog.InfoContext(ctx, "goroutine", "name", "ProcessRecencyItems")

	buf := &queue{}

	for {
		select {
		case <-ctx.Done():
			return
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

			items, err := client.MarketBoardCurrentData(ctx, ids, batch[0].WorldName)
			if err != nil {
				slog.ErrorContext(ctx, "error fetching market board data", "err", err)
				time.Sleep(time.Second)
				continue
			}

			for _, item := range items {
				current <- item
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

func fetchInitial(
	ctx context.Context,
	client *xivmarketgo.Client,
	worlds []string,
	itemIDs []int,
	current chan xivmarketgo.CurrentlyShown,
) {
	slog.InfoContext(ctx, "goroutine", "name", "FetchMarketBoardData")

	total := len(worlds) * len(itemIDs)

	completed := 0

	for _, world := range worlds {
		for ids := range slices.Chunk(itemIDs, 50) {
			select {
			case <-ctx.Done():
				return
			default:
			retry:
				for {
					items, err := client.MarketBoardCurrentData(ctx, ids, world)
					if errors.Is(err, context.Canceled) {
						return
					}

					if err != nil {
						slog.ErrorContext(ctx, "error fetching market board data", "err", err)
						time.Sleep(time.Second)
						continue retry
					}

					for _, item := range items {
						current <- item
						completed += 1
					}

					slog.InfoContext(ctx, "fetched market board data",
						"world", world,
						"batch", len(ids),
						"count", len(items),
						"progress", fmt.Sprintf("%d/%d (%.2f%%)", completed, total, float64(completed)/float64(total)*100),
					)
					break retry
				}
			}
		}
	}
}
