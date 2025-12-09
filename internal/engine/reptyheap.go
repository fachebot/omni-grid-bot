package engine

import "time"

type retryItem struct {
	strategyID string
	retryTime  time.Time
	index      int
}

type retryHeap []*retryItem

func (h retryHeap) Len() int           { return len(h) }
func (h retryHeap) Less(i, j int) bool { return h[i].retryTime.Before(h[j].retryTime) }
func (h retryHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *retryHeap) Push(x any) {
	n := len(*h)
	item := x.(*retryItem)
	item.index = n
	*h = append(*h, item)
}

func (h *retryHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*h = old[0 : n-1]
	return item
}
