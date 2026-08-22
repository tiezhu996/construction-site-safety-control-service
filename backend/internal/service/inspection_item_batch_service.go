package service

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"safetyplatform/internal/constants"
	"safetyplatform/internal/model"
	"safetyplatform/internal/repository"
	"safetyplatform/internal/util"
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
// Each worker owns a buffered slot in both result and error channels, so a
// failing or slow worker never blocks its peers or the aggregator. The
// WaitGroup counter is incremented before each goroutine is launched so the
// wait below cannot observe a zero count while workers are still starting.
func (s *InspectionItemBatchService) Process(ctx context.Context, items []model.InspectionItem) ([]model.InspectionItem, error) {
	results := make(chan model.InspectionItem, len(items))
	errCh := make(chan error, len(items))
	var wg sync.WaitGroup
	for i := range items {
		item := items[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.updater.UpdateBatchItem(ctx, &item); err != nil {
				errCh <- fmt.Errorf("batch item %s: %w", item.BatchKey(), mapBatchError(err))
				return
			}
			results <- item
		}()
	}
	wg.Wait()
	close(results)
	close(errCh)

	var aggErr error
	for err := range errCh {
		aggErr = errors.Join(aggErr, err)
	}

	completed := make([]model.InspectionItem, 0, len(items))
	for item := range results {
		completed = append(completed, item)
	}
	return completed, aggErr
}

// mapBatchError converts persistence-layer not-found conditions into a 404
// business error so the HTTP layer can surface the right status; all other
// errors pass through unchanged.
func mapBatchError(err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return util.NewAppError(constants.CodeNotFound, "InspectionItem batch: item not found")
	}
	return err
}
