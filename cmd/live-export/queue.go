package main

import (
	"slices"

	xivmarketgo "github.com/t-arik/xiv-market-go"
)

type queue struct {
	buf []xivmarketgo.WorldItemRecency
}

func (q *queue) Len() int {
	return len(q.buf)
}

func (q *queue) Add(item xivmarketgo.WorldItemRecency) {
	idx := slices.IndexFunc(q.buf, func(existing xivmarketgo.WorldItemRecency) bool {
		return existing.ItemId == item.ItemId && existing.WorldName == item.WorldName
	})

	if idx == -1 {
		q.buf = append(q.buf, item)
		return
	}

	if item.LastUploadTime > q.buf[idx].LastUploadTime {
		q.buf[idx] = item
	}
}

func (q *queue) MinLastUploadTime() xivmarketgo.WorldItemRecency {
	return slices.MinFunc(q.buf, func(a, b xivmarketgo.WorldItemRecency) int {
		return int(a.LastUploadTime - b.LastUploadTime)
	})
}

func (q *queue) Batch() []xivmarketgo.WorldItemRecency {
	world := q.MinLastUploadTime().WorldName
	worldItems := []xivmarketgo.WorldItemRecency{}

	for _, item := range q.buf {
		if item.WorldName == world {
			worldItems = append(worldItems, item)
		}
	}

	slices.SortFunc(q.buf, func(a, b xivmarketgo.WorldItemRecency) int {
		return int(a.LastUploadTime - b.LastUploadTime)
	})

	return worldItems[:min(100, len(worldItems))]
}

func (q *queue) Remove(cutset []xivmarketgo.WorldItemRecency) {
	newBuf := make([]xivmarketgo.WorldItemRecency, 0, q.Len())
	for _, item := range q.buf {
		if !slices.Contains(cutset, item) {
			newBuf = append(newBuf, item)
		}
	}

	q.buf = newBuf
}
