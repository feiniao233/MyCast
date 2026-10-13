// SPDX-License-Identifier: AGPL-3.0-or-later
package core

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

var ErrScanBudget = errors.New("application scan reached its budget")

type ScanOptions struct {
	MaxEntries, MaxApplications, MaxDepth int
	AllowPath                             func(string) error
	// Ordinary resolved links can be classified without changing launch data.
	// Missing/unresolved/special links are kept independently by the indexer.
	IncludeResolvedTarget func(LaunchIdentity) bool
}

func DefaultScanOptions() ScanOptions {
	return ScanOptions{MaxEntries: 20000, MaxApplications: 5000, MaxDepth: 32}
}

// Bounded iteration reads at most 128 directory entries per batch. Cancellation
// is cooperative between filesystem/native calls, which cannot be interrupted.
func boundedWalk(ctx context.Context, path string, opts ScanOptions, visited *int, depth int, visit fs.WalkDirFunc) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if depth > opts.MaxDepth || *visited >= opts.MaxEntries {
		return ErrScanBudget
	}
	if opts.AllowPath != nil {
		if err := opts.AllowPath(path); err != nil {
			return visit(path, nil, err)
		}
	}
	info, err := os.Lstat(path)
	if err != nil {
		return visit(path, nil, err)
	}
	entry := fs.FileInfoToDirEntry(info)
	*visited++
	if err := visit(path, entry, nil); err != nil {
		if err == fs.SkipDir && entry.IsDir() {
			return nil
		}
		return err
	}
	if !entry.IsDir() || entry.Type()&fs.ModeSymlink != 0 {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	file, err := os.Open(path)
	if err != nil {
		return visit(path, entry, err)
	}
	defer file.Close()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		entries, readErr := file.ReadDir(128)
		for _, child := range entries {
			if err := boundedWalk(ctx, filepath.Join(path, child.Name()), opts, visited, depth+1, visit); err != nil {
				return err
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return visit(path, entry, readErr)
		}
	}
}

// ScanQueue is owned by the UI thread. Requests coalesce while one worker is
// active; its completion is discarded before starting the latest request.
type ScanQueue struct {
	Generation, active uint64
	Running, pending   bool
}

func (q *ScanQueue) Request() (uint64, bool) {
	q.Generation++
	if q.Running {
		q.pending = true
		return q.Generation, false
	}
	q.Running = true
	q.active = q.Generation
	return q.active, true
}
func (q *ScanQueue) Finish(generation uint64) (publish, restart bool) {
	if !q.Running || generation != q.active {
		return false, false
	}
	q.Running = false
	restart = q.pending
	q.pending = false
	return generation == q.Generation && !restart, restart
}
