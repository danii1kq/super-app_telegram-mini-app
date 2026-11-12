package split

import (
	"context"
	"log/slog"
	"time"
)

// PurgeResult reports what one cleanup pass did.
type PurgeResult struct {
	// Scanned is how many rows were considered.
	Scanned int
	// Purged is how many photos were deleted.
	Purged int
	// BytesFreed is the sum of the stored image sizes that were removed.
	BytesFreed int64
}

// PurgeExpiredImages deletes photos whose retention has run out.
//
// It works in bounded batches so it never holds the pool or blocks user
// queries, and it only touches the image columns: the receipt, its items,
// shares, expenses, debts and repayments all stay and keep working in the
// report. A row in `processing` is skipped, so an active recognition is never
// deleted under itself.
//
// The pass is safe to repeat: a purged row no longer matches the filter.
func (s *Service) PurgeExpiredImages(ctx context.Context, batch int) (PurgeResult, error) {
	if batch <= 0 || batch > 500 {
		batch = 100
	}

	var result PurgeResult

	rows, err := s.pool.Query(ctx,
		`SELECT id, image_bytes
		   FROM split_receipts
		  WHERE image_state = 'stored'
		    AND image_expires_at IS NOT NULL
		    AND image_expires_at <= now()
		  ORDER BY image_expires_at
		  LIMIT $1
		  FOR UPDATE SKIP LOCKED`, batch)
	if err != nil {
		return result, err
	}

	type candidate struct {
		id    int64
		bytes int64
	}
	found := make([]candidate, 0, batch)
	for rows.Next() {
		var item candidate
		if err := rows.Scan(&item.id, &item.bytes); err != nil {
			rows.Close()
			return result, err
		}
		found = append(found, item)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return result, err
	}
	result.Scanned = len(found)

	for _, item := range found {
		// One row at a time: a short transaction per photo keeps the pool free
		// for user requests.
		tag, err := s.pool.Exec(ctx,
			`UPDATE split_receipts
			    SET image_sealed = ''::bytea, image_key_id = 'purged',
			        image_state = 'purged', image_bytes = 0, updated_at = now()
			  WHERE id = $1 AND image_state = 'stored'`, item.id)
		if err != nil {
			return result, err
		}
		if tag.RowsAffected() == 0 {
			// Someone else purged it first: nothing to count.
			continue
		}
		result.Purged++
		result.BytesFreed += item.bytes
	}

	if result.Purged > 0 {
		// No VACUUM FULL: it locks the table. Autovacuum reclaims the space on
		// its own, and the freed tuples are reused by later writes.
		slog.Info("split: receipt photos purged",
			"purged", result.Purged, "bytes_freed", result.BytesFreed)
	}
	return result, nil
}

// RunImagePurge runs the cleanup on an interval until ctx is cancelled.
func (s *Service) RunImagePurge(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.PurgeExpiredImages(ctx, 100); err != nil {
				slog.Error("split: image purge", "err", err)
			}
		}
	}
}

// ScheduleSavedImage moves a confirmed receipt's photo to the saved retention.
//
// Called when a draft becomes an expense: a confirmed receipt keeps its photo
// for the longer period, a draft keeps the short one.
func (s *Service) ScheduleSavedImage(ctx context.Context, receiptID int64) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE split_receipts
		    SET image_expires_at = now() + $2::interval, updated_at = now()
		  WHERE id = $1 AND image_state = 'stored'`,
		receiptID, savedImageTTL)
	return err
}

// ImagePurgePreview is the dry run for a batch over old photos.
type ImagePurgePreview struct {
	// Candidates is how many photos would be deleted by the given policy.
	Candidates int
	// TotalBytes is their stored size.
	TotalBytes int64
	// Oldest and Newest bound the candidate dates.
	Oldest *time.Time
	Newest *time.Time
}

// PreviewImagePurge reports what a purge would do, without changing anything.
//
// The policy is passed in explicitly: a new retention must never silently delete
// old history, so an operator runs this first and sees the numbers.
func (s *Service) PreviewImagePurge(ctx context.Context, olderThan time.Duration) (ImagePurgePreview, error) {
	var preview ImagePurgePreview
	cutoff := time.Now().Add(-olderThan)

	err := s.pool.QueryRow(ctx,
		`SELECT count(*), coalesce(sum(image_bytes), 0), min(created_at), max(created_at)
		   FROM split_receipts
		  WHERE image_state = 'stored' AND created_at <= $1`, cutoff,
	).Scan(&preview.Candidates, &preview.TotalBytes, &preview.Oldest, &preview.Newest)
	return preview, err
}
