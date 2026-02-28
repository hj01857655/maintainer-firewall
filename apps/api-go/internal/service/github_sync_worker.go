package service

import (
	"context"
	"log"
	"time"
)

func StartGitHubEventsSyncWorker(ctx context.Context, interval time.Duration, runOnce func(context.Context) (int, int, error)) {
	if interval <= 0 || runOnce == nil {
		return
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
				saved, total, err := runOnce(runCtx)
				cancel()
				if err != nil {
					log.Printf("github events sync failed: %v", err)
					continue
				}
				log.Printf("github events sync done: saved=%d total=%d", saved, total)
			}
		}
	}()
}
