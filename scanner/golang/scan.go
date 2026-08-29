package golang

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"github.com/grokify/deppolicy/component"
	"github.com/grokify/deppolicy/graph"
)

// ScanResult is the per-module outcome of a concurrent scan.
type ScanResult struct {
	Dir           string
	ModuleLocator string
	Dependencies  []graph.Dependency
	Err           error
}

// ScanModules scans multiple module directories concurrently for
// source-import dependency evidence (see ImportDependencies), using a
// bounded worker pool so a fleet-wide scan does not open thousands of files
// at once. workers <= 0 defaults to runtime.GOMAXPROCS(0).
//
// Workers only parse files and emit facts; aggregation happens in the
// single-goroutine result loop below (results[idx] = ...), so no lock is
// needed around shared graph state -- each worker owns a disjoint slice
// index, which is safe to write concurrently without synchronization.
//
// One module's scan error does not abort the others: it is recorded in
// that module's ScanResult, so a handful of broken repositories cannot
// block a fleet-wide scan. Results are returned in the same order as dirs,
// for deterministic output regardless of goroutine scheduling.
func ScanModules(ctx context.Context, dirs []string, registry *component.Registry, workers int) []ScanResult {
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	if workers > len(dirs) {
		workers = len(dirs)
	}
	if workers < 1 {
		return nil
	}

	results := make([]ScanResult, len(dirs))
	jobs := make(chan int)

	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			for idx := range jobs {
				if err := ctx.Err(); err != nil {
					results[idx] = ScanResult{Dir: dirs[idx], Err: err}
					continue
				}
				dir := dirs[idx]
				moduleLocator, deps, err := ImportDependencies(dir, registry)
				results[idx] = ScanResult{
					Dir:           dir,
					ModuleLocator: moduleLocator,
					Dependencies:  deps,
					Err:           err,
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for i := range dirs {
			select {
			case jobs <- i:
			case <-ctx.Done():
				return
			}
		}
	}()

	wg.Wait()
	return results
}

// MergeResults flattens every successful ScanResult's dependencies into one
// deduplicated edge set (see graph.MergeDependencies) and separately
// collects per-module errors, so a caller can report scan failures without
// losing the rest of the scan.
func MergeResults(results []ScanResult) (deps []graph.Dependency, errs []error) {
	var all []graph.Dependency
	for _, r := range results {
		if r.Err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", r.Dir, r.Err))
			continue
		}
		all = append(all, r.Dependencies...)
	}
	return graph.MergeDependencies(all), errs
}
