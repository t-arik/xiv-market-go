package xivmarketgo

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

type Client struct {
	address *url.URL
	client  *http.Client
}

func DefaultRestClient() *Client {
	address, err := url.Parse("https://universalis.app")
	if err != nil {
		panic(err)
	}
	return &Client{
		address: address,
		client:  http.DefaultClient,
	}
}

func (client *Client) MostRecentlyUpdatedItems(ctx context.Context, world string, dc string) ([]WorldItemRecency, error) {
	addr := client.address.JoinPath("api", "v2", "extra", "stats", "most-recently-updated")

	values := url.Values{}
	if world != "" {
		values.Add("world", world)
	}
	if dc != "" {
		values.Add("dcName", dc)
	}
	values.Add("entries", "200")

	addr.RawQuery = values.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, addr.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	type responseModel struct {
		Items []WorldItemRecency
	}

	var result responseModel

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Items, nil
}

func (client *Client) StreamMostRecentlyUpdatedItems(
	ctx context.Context,
	world string,
	dc string,
	c chan WorldItemRecency,
) error {
	prevItems := []WorldItemRecency{}

	ticker := time.NewTicker(2 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			newItems, err := client.MostRecentlyUpdatedItems(ctx, world, dc)
			if err != nil {
				return err
			}
			slices.SortFunc(newItems, func(a, b WorldItemRecency) int {
				return int(a.LastUploadTime - b.LastUploadTime)
			})
			for _, item := range newItems {
				if !slices.Contains(prevItems, item) {
					select {
					case c <- item:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			}
			prevItems = newItems
		}
	}
}

func (client *Client) MarketBoardCurrentData(ctx context.Context, itemIds []int, worldDcRegion string) ([]CurrentlyShown, error) {
	itemIdsStr := strings.Join(strings.Fields(strings.Trim(fmt.Sprint(itemIds), "[]")), ",") // eek
	addr := client.address.JoinPath("api", "v2", worldDcRegion, itemIdsStr)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, addr.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result []CurrentlyShown

	if len(itemIds) == 1 {
		var view CurrentlyShown

		if err := json.NewDecoder(resp.Body).Decode(&view); err != nil {
			return nil, err
		}

		result = []CurrentlyShown{view}
	} else {
		var view MultiView[CurrentlyShown]

		if err := json.NewDecoder(resp.Body).Decode(&view); err != nil {
			return nil, err
		}

		result = slices.Collect(maps.Values(view.Items))

		for i := range result {
			if result[i].RegionName == "" && view.RegionName != "" {
				result[i].RegionName = view.RegionName
			}
			if result[i].DcName == "" && view.DcName != "" {
				result[i].DcName = view.DcName
			}
			if result[i].WorldName == "" && view.WorldName != "" {
				result[i].WorldName = view.WorldName
			}
		}
	}

	for _, view := range result {
		for i := range view.Listings {
			if view.Listings[i].ItemId == 0 && view.ItemId != 0 {
				view.Listings[i].ItemId = view.ItemId
			}
			if view.Listings[i].WorldId == 0 && view.WorldId != 0 {
				view.Listings[i].WorldId = view.WorldId
			}
			if view.Listings[i].WorldName == "" && view.WorldName != "" {
				view.Listings[i].WorldName = view.WorldName
			}
		}

		for i := range view.RecentHistory {
			if view.RecentHistory[i].ItemId == 0 && view.ItemId != 0 {
				view.RecentHistory[i].ItemId = view.ItemId
			}
			if view.RecentHistory[i].WorldId == 0 && view.WorldId != 0 {
				view.RecentHistory[i].WorldId = view.WorldId
			}
			if view.RecentHistory[i].WorldName == "" && view.WorldName != "" {
				view.RecentHistory[i].WorldName = view.WorldName
			}
		}
	}

	return result, nil
}
