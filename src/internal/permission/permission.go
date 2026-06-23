package permission

import (
	"database/sql"
	"strings"
)

// Perm defines a single access right.
type Perm int

const (
	PermSee    Perm = iota // file visibility
	PermRead               // download / read content
	PermList               // list directory contents
	PermUpload             // upload files / create dirs
	PermDelete             // delete files / dirs
	PermArchive            // batch ZIP download
)

// PermSet holds all 6 permissions for a user-path pair.
type PermSet struct {
	CanSee    bool `json:"canSee"`
	CanRead   bool `json:"canRead"`
	CanList   bool `json:"canList"`
	CanUpload bool `json:"canUpload"`
	CanDelete bool `json:"canDelete"`
	CanArchive bool `json:"canArchive"`
}

// PermissionRecord is a row from the permissions table.
type PermissionRecord struct {
	ID       int     `json:"id"`
	Username string  `json:"username"`
	VFSPath  string  `json:"vfsPath"`
	PermSet
}

// Engine checks permissions against the database.
type Engine struct {
	db *sql.DB
}

// NewEngine creates a permission engine with the given database connection.
func NewEngine(db *sql.DB) *Engine {
	return &Engine{db: db}
}

// DefaultPerms returns the default permission set for a logged-in user
// who has no explicit permission records: can see, read, list — but not upload/delete/archive.
func DefaultPerms() PermSet {
	return PermSet{
		CanSee:    true,
		CanRead:   true,
		CanList:   true,
		CanUpload: false,
		CanDelete: false,
		CanArchive: false,
	}
}

// Check returns the effective PermSet for a username and VFS path.
// Inheritance: if no exact match is found, walks up parent paths until root.
// Admin users always get all permissions.
func (e *Engine) Check(username, role, vfsPath string) PermSet {
	// Admin bypasses all checks.
	if role == "admin" {
		return PermSet{
			CanSee:    true,
			CanRead:   true,
			CanList:   true,
			CanUpload: true,
			CanDelete: true,
			CanArchive: true,
		}
	}

	// Walk up the path hierarchy looking for a matching permission record.
	path := cleanVFSPath(vfsPath)
	for {
		ps, found := e.lookup(username, path)
		if found {
			return ps
		}
		if path == "/" || path == "" {
			break
		}
		path = parentPath(path)
	}

	return DefaultPerms()
}

// Can checks if a specific permission is granted.
func (e *Engine) Can(username, role, vfsPath string, perm Perm) bool {
	ps := e.Check(username, role, vfsPath)
	switch perm {
	case PermSee:
		return ps.CanSee
	case PermRead:
		return ps.CanRead
	case PermList:
		return ps.CanList
	case PermUpload:
		return ps.CanUpload
	case PermDelete:
		return ps.CanDelete
	case PermArchive:
		return ps.CanArchive
	}
	return false
}

// lookup queries the permissions table for an exact username + vfs_path match.
func (e *Engine) lookup(username, vfsPath string) (PermSet, bool) {
	var ps PermSet
	err := e.db.QueryRow(
		`SELECT can_see, can_read, can_list, can_upload, can_delete, can_archive
		 FROM permissions WHERE username = ? AND vfs_path = ?`,
		username, vfsPath,
	).Scan(&ps.CanSee, &ps.CanRead, &ps.CanList, &ps.CanUpload, &ps.CanDelete, &ps.CanArchive)
	if err == sql.ErrNoRows {
		return PermSet{}, false
	}
	if err != nil {
		return PermSet{}, false
	}
	return ps, true
}

// ListForUser returns all permission records for a given username.
func (e *Engine) ListForUser(username string) ([]PermissionRecord, error) {
	rows, err := e.db.Query(
		`SELECT id, username, vfs_path, can_see, can_read, can_list, can_upload, can_delete, can_archive
		 FROM permissions WHERE username = ? ORDER BY vfs_path`, username,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []PermissionRecord
	for rows.Next() {
		var r PermissionRecord
		if err := rows.Scan(&r.ID, &r.Username, &r.VFSPath,
			&r.CanSee, &r.CanRead, &r.CanList, &r.CanUpload, &r.CanDelete, &r.CanArchive); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// ListAll returns all permission records, optionally filtered by vfs_path.
func (e *Engine) ListAll(vfsPath string) ([]PermissionRecord, error) {
	var rows *sql.Rows
	var err error
	if vfsPath == "" {
		rows, err = e.db.Query(
			`SELECT id, username, vfs_path, can_see, can_read, can_list, can_upload, can_delete, can_archive
			 FROM permissions ORDER BY vfs_path, username`,
		)
	} else {
		rows, err = e.db.Query(
			`SELECT id, username, vfs_path, can_see, can_read, can_list, can_upload, can_delete, can_archive
			 FROM permissions WHERE vfs_path = ? ORDER BY username`, vfsPath,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []PermissionRecord
	for rows.Next() {
		var r PermissionRecord
		if err := rows.Scan(&r.ID, &r.Username, &r.VFSPath,
			&r.CanSee, &r.CanRead, &r.CanList, &r.CanUpload, &r.CanDelete, &r.CanArchive); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// Set creates or updates a permission record for a user-path pair.
func (e *Engine) Set(username, vfsPath string, ps PermSet) error {
	_, err := e.db.Exec(
		`INSERT INTO permissions (username, vfs_path, can_see, can_read, can_list, can_upload, can_delete, can_archive, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
		 ON CONFLICT(username, vfs_path) DO UPDATE SET
		   can_see = excluded.can_see, can_read = excluded.can_read, can_list = excluded.can_list,
		   can_upload = excluded.can_upload, can_delete = excluded.can_delete, can_archive = excluded.can_archive,
		   updated_at = datetime('now')`,
		username, vfsPath, ps.CanSee, ps.CanRead, ps.CanList, ps.CanUpload, ps.CanDelete, ps.CanArchive,
	)
	return err
}

// SetBatch creates or updates permissions for multiple users on the same path.
func (e *Engine) SetBatch(usernames []string, vfsPath string, ps PermSet) error {
	tx, err := e.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(
		`INSERT INTO permissions (username, vfs_path, can_see, can_read, can_list, can_upload, can_delete, can_archive, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
		 ON CONFLICT(username, vfs_path) DO UPDATE SET
		   can_see = excluded.can_see, can_read = excluded.can_read, can_list = excluded.can_list,
		   can_upload = excluded.can_upload, can_delete = excluded.can_delete, can_archive = excluded.can_archive,
		   updated_at = datetime('now')`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, username := range usernames {
		if _, err := stmt.Exec(username, vfsPath, ps.CanSee, ps.CanRead, ps.CanList, ps.CanUpload, ps.CanDelete, ps.CanArchive); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Delete removes a permission record by ID.
func (e *Engine) Delete(id int) error {
	_, err := e.db.Exec(`DELETE FROM permissions WHERE id = ?`, id)
	return err
}

// DeleteByPath removes all permission records for a given VFS path.
func (e *Engine) DeleteByPath(vfsPath string) error {
	_, err := e.db.Exec(`DELETE FROM permissions WHERE vfs_path = ?`, vfsPath)
	return err
}

// cleanVFSPath normalizes a VFS path: ensures leading /, removes trailing / (except root).
func cleanVFSPath(p string) string {
	p = strings.TrimSpace(p)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if len(p) > 1 && strings.HasSuffix(p, "/") {
		p = p[:len(p)-1]
	}
	return p
}

// parentPath returns the parent directory of a VFS path.
// e.g. "/Files/a/b" → "/Files/a", "/Files" → "/", "/" → ""
func parentPath(p string) string {
	if p == "/" || p == "" {
		return ""
	}
	idx := strings.LastIndex(p, "/")
	if idx <= 0 {
		return "/"
	}
	return p[:idx]
}
