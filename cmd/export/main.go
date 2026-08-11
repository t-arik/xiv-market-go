package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path"
	"time"

	xivmarketgo "github.com/t-arik/xiv-market-go"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		panic(err)
	}
}

func run(ctx context.Context) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	region := flag.String("region", "", "")
	outputDir := flag.String("outputDir", "", "")
	flag.Parse()

	if *region == "" || *outputDir == "" {
		flag.Usage()
		return nil
	}

	client := xivmarketgo.DefaultRestClient()

	items := make(chan xivmarketgo.WorldItemRecency, 4*1024)

	go func() {
		if err := client.StreamMostRecentlyUpdatedItems(ctx, "", *region, items); err != nil {
			cancel(err)
		}
	}()

	buf := &queue{}

	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case item := <-items:
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
				return err
			}

			for _, item := range items {
				filename := fmt.Sprintf("%d-%d-%s.json", item.LastUploadTime, item.ItemId, item.WorldName)
				filepath := path.Join(*outputDir, filename)
				f, err := os.Create(filepath)
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
			buf.Remove(batch)
		}
	}
}
