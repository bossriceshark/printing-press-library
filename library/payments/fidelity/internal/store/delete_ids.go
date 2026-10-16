package store

import "fmt"

// DeleteIDs removes generic resource rows, their FTS entries, and the
// matching typed-table rows. Hand-written CSV import uses this when a new
// positions file replaces one account's snapshot.
func (s *Store) DeleteIDs(resourceType string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	s.lockForWrite()
	defer s.unlockAfterWrite()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	table, hasTyped := TypedListTable(resourceType)
	for _, id := range ids {
		if _, err := tx.Exec(`DELETE FROM resources WHERE resource_type = ? AND id = ?`, resourceType, id); err != nil {
			return fmt.Errorf("deleting %s/%s: %w", resourceType, id, err)
		}
		if _, err := tx.Exec(`DELETE FROM resources_fts WHERE rowid = ?`, ftsRowID(resourceType, id)); err != nil {
			return fmt.Errorf("deleting fts %s/%s: %w", resourceType, id, err)
		}
		if hasTyped {
			query := fmt.Sprintf(`DELETE FROM "%s" WHERE id = ?`, table)
			if _, err := tx.Exec(query, id); err != nil {
				return fmt.Errorf("deleting typed %s/%s: %w", resourceType, id, err)
			}
		}
	}
	return tx.Commit()
}
