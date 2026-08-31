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

	client := xivmarketgo.DefaultRestClient()

	itemIDs, err := client.MarketableItems(ctx)
	if err != nil {
		return err
	}

	retryCount := 0

	var outFile *os.File

	for i, itemId := range itemIDs {
		select {
		case <-ctx.Done():
			return nil
		default:
		retry:
			for {
				ids := []int{itemId}

				items, err := client.MarketBoardCurrentData(ctx, ids, region, 50_000)
				if errors.Is(err, context.Canceled) {
					return nil
				}

				if err != nil {
					retryCount++
					slog.Error("error fetching market board data", "err", err, "retryCount", retryCount)

					if retryCount > 32 {
						return fmt.Errorf("too many errors fetching market board data: %w", err)
					}

					time.Sleep(time.Second)

					continue retry
				}

				if len(items) != 1 {
					return fmt.Errorf("unexpected number of items returned for item ID %d: got %d, want 1", itemId, len(items))
				}

				retryCount = 0

				outFile, err = getOutFile(time.Now().UTC(), outDir, outFile)
				if err != nil {
					return fmt.Errorf("failed to get output file: %w", err)
				}

				if err := json.NewEncoder(outFile).Encode(items[0]); err != nil {
					return fmt.Errorf("failed to encode item to output file: %w", err)
				}

				if err := outFile.Sync(); err != nil {
					return fmt.Errorf("failed to sync output file: %w", err)
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
