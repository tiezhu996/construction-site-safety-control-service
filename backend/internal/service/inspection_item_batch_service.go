package service

import (
	"context"
	"fmt"
	"sync"

	"safetyplatform/internal/model"
)

// InspectionItemBatchUpdater is the persistence contract used by batch workers.
type InspectionItemBatchUpdater interface {
	UpdateBatchItem(context.Context, *model.InspectionItem) error
}

// InspectionItemBatchService coordinates concurrent item updates.
type InspectionItemBatchService struct {
	updater InspectionItemBatchUpdater
}

func NewInspectionItemBatchService(updater InspectionItemBatchUpdater) *InspectionItemBatchService {
	return &InspectionItemBatchService{updater: updater}
}

// Process applies all updates and returns only after every worker has reported.
func (s *InspectionItemBatchService) Process(ctx context.Context, items []model.InspectionItem) ([]model.InspectionItem, error) {
	results := make(chan model.InspectionItem, len(items))
	errs := make(chan error)
	var wg sync.WaitGroup
	for i := range items {
		item := items[i]
		go func() {
			if err := s.updater.UpdateBatchItem(ctx, &item); err != nil {
				errs <- fmt.Errorf("batch item %s: %w", item.BatchKey(), err)
				return
			}
			wg.Add(1)
			defer wg.Done()
			results <- item
		}()
	}
	wg.Wait()

	completed := make([]model.InspectionItem, 0, len(items))
	for {
		select {
		case item := <-results:
			completed = append(completed, item)
		case err := <-errs:
			return nil, err
		default:
			return completed, nil
		}
	}
}
