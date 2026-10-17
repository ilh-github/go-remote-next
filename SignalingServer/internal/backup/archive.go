package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/glebarez/go-sqlite"
)

const (
	formatVersion = 1
	manifestName  = "manifest.json"
	databaseName  = "database/quickdesk.db"
)

type FileRecord struct {
	Path   string `json:"path"`
	Size   int64  `json:"size_bytes"`
	SHA256 string `json:"sha256"`
}

type Manifest struct {
	FormatVersion int          `json:"format_version"`
	CreatedAt     time.Time    `json:"created_at"`
	Database      string       `json:"database"`
	Files         []FileRecord `json:"files"`
	TotalBytes    int64        `json:"total_bytes"`
}

type CreateOptions struct {
	DatabasePath  string
	RecordingsDir string
	UpdatesDir    string
	OutputPath    string
}

type RestoreOptions struct {
	ArchivePath   string
	DatabasePath  string
	RecordingsDir string
	UpdatesDir    string
	Force         bool
}

type RestoreResult struct {
	RestoredFiles  int64    `json:"restored_files"`
	RestoredBytes  int64    `json:"restored_bytes"`
	PreviousCopies []string `json:"previous_copies,omitempty"`
}

// Create writes one portable archive containing a consistent SQLite snapshot
// and the media directories the database points at. PostgreSQL deployments
// continue to use pg_dump; this command targets the single-file mode where a
// filesystem copy of a live WAL database would not be consistent.
func Create(ctx context.Context, opts CreateOptions) (Manifest, error) {
	var manifest Manifest
	dbPath, err := requiredFile(opts.DatabasePath, "database path")
	if err != nil {
		return manifest, err
	}
	output, err := filepath.Abs(strings.TrimSpace(opts.OutputPath))
	if err != nil || strings.TrimSpace(opts.OutputPath) == "" {
		return manifest, errors.New("output path is required")
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o750); err != nil {
		return manifest, fmt.Errorf("create output directory: %w", err)
	}

	stage, err := os.MkdirTemp(filepath.Dir(output), ".quickdesk-backup-stage-*")
	if err != nil {
		return manifest, fmt.Errorf("create staging directory: %w", err)
	}
	defer os.RemoveAll(stage)

	dbCopy := filepath.Join(stage, filepath.FromSlash(databaseName))
	if err := snapshotSQLite(ctx, dbPath, dbCopy); err != nil {
		return manifest, err
	}
	if strings.TrimSpace(opts.RecordingsDir) != "" {
		if err := copyTree(opts.RecordingsDir, filepath.Join(stage, "recordings")); err != nil {
			return manifest, fmt.Errorf("copy recordings: %w", err)
		}
	}
	if strings.TrimSpace(opts.UpdatesDir) != "" {
		if err := copyTree(opts.UpdatesDir, filepath.Join(stage, "updates")); err != nil {
			return manifest, fmt.Errorf("copy update artifacts: %w", err)
		}
	}

	files, total, err := checksumTree(stage, manifestName)
	if err != nil {
		return manifest, err
	}
	manifest = Manifest{
		FormatVersion: formatVersion,
		CreatedAt:     time.Now().UTC(),
		Database:      databaseName,
		Files:         files,
		TotalBytes:    total,
	}
	rawManifest, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return Manifest{}, fmt.Errorf("encode manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(stage, manifestName), append(rawManifest, '\n'), 0o640); err != nil {
		return Manifest{}, fmt.Errorf("write manifest: %w", err)
	}

	if err := writeArchive(ctx, stage, output); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func requiredFile(raw, label string) (string, error) {
	path := strings.TrimSpace(raw)
	if path == "" {
		return "", fmt.Errorf("%s is required", label)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", label, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", label, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s is not a regular file", label)
	}
	return abs, nil
}

func snapshotSQLite(ctx context.Context, source, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
		return fmt.Errorf("create database snapshot directory: %w", err)
	}
	conn, err := sql.Open("sqlite", "file:"+filepath.ToSlash(source)+"?mode=ro")
	if err != nil {
		return fmt.Errorf("open SQLite source: %w", err)
	}
	defer conn.Close()
	if err := conn.PingContext(ctx); err != nil {
		return fmt.Errorf("ping SQLite source: %w", err)
	}
	statement := "VACUUM INTO " + sqliteString(dest)
	if _, err := conn.ExecContext(ctx, statement); err != nil {
		return fmt.Errorf("snapshot SQLite database: %w", err)
	}
	return nil
}

func sqliteString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func copyTree(source, dest string) error {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil
	}
	info, err := os.Lstat(source)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("%s is not a directory", source)
	}
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dest, 0o750)
		}
		target := filepath.Join(dest, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o750)
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		return copyFile(path, target, 0o640)
	})
}

func copyFile(source, dest string, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(dest)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(dest)
		return closeErr
	}
	return nil
}

func checksumTree(root, exclude string) ([]FileRecord, int64, error) {
	var files []FileRecord
	var total int64
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !entry.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == exclude {
			return nil
		}
		record, err := checksumFile(path)
		if err != nil {
			return err
		}
		record.Path = rel
		files = append(files, record)
		total += record.Size
		return nil
	})
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, total, err
}

func checksumFile(path string) (FileRecord, error) {
	file, err := os.Open(path)
	if err != nil {
		return FileRecord{}, err
	}
	defer file.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return FileRecord{}, err
	}
	return FileRecord{Size: size, SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}

func writeArchive(ctx context.Context, stage, output string) error {
	tmp, err := os.CreateTemp(filepath.Dir(output), "."+filepath.Base(output)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create archive: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	gz := gzip.NewWriter(tmp)
	tw := tar.NewWriter(gz)
	err = filepath.WalkDir(stage, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(stage, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)
		if entry.IsDir() {
			header.Name += "/"
		}
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(tw, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	if err != nil {
		_ = tw.Close()
		_ = gz.Close()
		return fmt.Errorf("write archive: %w", err)
	}
	if err := tw.Close(); err != nil {
		_ = gz.Close()
		return fmt.Errorf("close tar: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("close gzip: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync archive: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close archive: %w", err)
	}
	if err := os.Rename(tmpName, output); err != nil {
		return fmt.Errorf("publish archive: %w", err)
	}
	return nil
}

func Restore(ctx context.Context, opts RestoreOptions) (RestoreResult, error) {
	var result RestoreResult
	archivePath := strings.TrimSpace(opts.ArchivePath)
	if archivePath == "" {
		return result, errors.New("archive path is required")
	}
	dbPath := strings.TrimSpace(opts.DatabasePath)
	if dbPath == "" {
		return result, errors.New("database path is required")
	}
	stage, err := os.MkdirTemp("", "quickdesk-restore-*")
	if err != nil {
		return result, fmt.Errorf("create restore staging directory: %w", err)
	}
	defer os.RemoveAll(stage)
	if err := extractArchive(ctx, archivePath, stage); err != nil {
		return result, err
	}
	manifest, err := readAndVerifyManifest(stage)
	if err != nil {
		return result, err
	}

	targets := []restoreTarget{
		{source: filepath.Join(stage, filepath.FromSlash(databaseName)), dest: dbPath, directory: false},
	}
	if strings.TrimSpace(opts.RecordingsDir) != "" {
		targets = append(targets, restoreTarget{
			source: filepath.Join(stage, "recordings"), dest: opts.RecordingsDir, directory: true,
		})
	}
	if strings.TrimSpace(opts.UpdatesDir) != "" {
		targets = append(targets, restoreTarget{
			source: filepath.Join(stage, "updates"), dest: opts.UpdatesDir, directory: true,
		})
	}
	for _, target := range targets {
		if !target.directory && target.source == filepath.Join(stage, filepath.FromSlash(databaseName)) {
			if _, err := os.Stat(target.source); err != nil {
				return result, fmt.Errorf("archive database snapshot is missing: %w", err)
			}
		}
		if target.directory {
			if _, err := os.Stat(target.source); os.IsNotExist(err) {
				continue
			}
		}
		if _, err := os.Stat(target.dest); err == nil && !opts.Force {
			return result, fmt.Errorf("%s already exists; pass --force to replace it", target.dest)
		} else if err != nil && !os.IsNotExist(err) {
			return result, fmt.Errorf("stat restore target %s: %w", target.dest, err)
		}
	}

	stamp := time.Now().UTC().Format("20060102T150405Z")
	type committedTarget struct {
		dest   string
		backup string
	}
	var committed []committedTarget
	for _, target := range targets {
		if _, err := os.Stat(target.source); os.IsNotExist(err) {
			continue
		}
		backupPath, err := replaceTarget(target, stamp)
		if err != nil {
			for i := len(committed) - 1; i >= 0; i-- {
				if rollbackErr := rollbackRestoreTarget(committed[i].dest, committed[i].backup); rollbackErr != nil {
					return result, fmt.Errorf("%w; rollback failed for %s: %v", err, committed[i].dest, rollbackErr)
				}
			}
			return result, err
		}
		if backupPath != "" {
			result.PreviousCopies = append(result.PreviousCopies, backupPath)
		}
		committed = append(committed, committedTarget{dest: target.dest, backup: backupPath})
	}
	result.RestoredFiles = int64(len(manifest.Files))
	result.RestoredBytes = manifest.TotalBytes
	return result, nil
}

func rollbackRestoreTarget(dest, backup string) error {
	if err := os.RemoveAll(dest); err != nil && !os.IsNotExist(err) {
		return err
	}
	if backup == "" {
		return nil
	}
	if err := os.Rename(backup, dest); err != nil {
		return err
	}
	return nil
}

type restoreTarget struct {
	source    string
	dest      string
	directory bool
}

func replaceTarget(target restoreTarget, stamp string) (string, error) {
	dest := filepath.Clean(target.dest)
	if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
		return "", fmt.Errorf("create restore target parent: %w", err)
	}
	staged := dest + ".restore-" + stamp
	_ = os.RemoveAll(staged)
	if target.directory {
		if err := copyTree(target.source, staged); err != nil {
			return "", fmt.Errorf("stage restored directory: %w", err)
		}
	} else {
		if err := copyFile(target.source, staged, 0o600); err != nil {
			return "", fmt.Errorf("stage restored database: %w", err)
		}
	}
	backupPath := ""
	if _, err := os.Stat(dest); err == nil {
		backupPath = dest + ".pre-restore-" + stamp
		_ = os.RemoveAll(backupPath)
		if err := os.Rename(dest, backupPath); err != nil {
			_ = os.RemoveAll(staged)
			return "", fmt.Errorf("move existing target aside: %w", err)
		}
	} else if !os.IsNotExist(err) {
		_ = os.RemoveAll(staged)
		return "", fmt.Errorf("stat existing target: %w", err)
	}
	if err := os.Rename(staged, dest); err != nil {
		if backupPath != "" {
			_ = os.Rename(backupPath, dest)
		}
		_ = os.RemoveAll(staged)
		return "", fmt.Errorf("install restored target: %w", err)
	}
	return backupPath, nil
}

func extractArchive(ctx context.Context, archivePath, dest string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("open gzip archive: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read archive: %w", err)
		}
		name, err := safeArchiveName(header.Name)
		if err != nil {
			return err
		}
		if name == "" {
			continue
		}
		target := filepath.Join(dest, filepath.FromSlash(name))
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o750); err != nil {
				return fmt.Errorf("create archive directory: %w", err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
				return fmt.Errorf("create archive file directory: %w", err)
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o640)
			if err != nil {
				return fmt.Errorf("create archive file: %w", err)
			}
			_, copyErr := io.Copy(out, tr)
			closeErr := out.Close()
			if copyErr != nil {
				return fmt.Errorf("extract archive file: %w", copyErr)
			}
			if closeErr != nil {
				return fmt.Errorf("close archive file: %w", closeErr)
			}
		default:
			return fmt.Errorf("archive contains unsupported entry %q", header.Name)
		}
	}
}

func safeArchiveName(raw string) (string, error) {
	name := strings.TrimSpace(filepath.ToSlash(raw))
	name = strings.TrimPrefix(name, "./")
	if name == "" || name == "." {
		return "", nil
	}
	if strings.HasPrefix(name, "/") || filepath.IsAbs(name) {
		return "", fmt.Errorf("archive contains an absolute path %q", raw)
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(name)))
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("archive entry escapes the restore root: %q", raw)
	}
	return clean, nil
}

func readAndVerifyManifest(stage string) (Manifest, error) {
	var manifest Manifest
	raw, err := os.ReadFile(filepath.Join(stage, manifestName))
	if err != nil {
		return manifest, fmt.Errorf("read backup manifest: %w", err)
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return manifest, fmt.Errorf("decode backup manifest: %w", err)
	}
	if manifest.FormatVersion != formatVersion {
		return manifest, fmt.Errorf("unsupported backup format version %d", manifest.FormatVersion)
	}
	if _, err := os.Stat(filepath.Join(stage, filepath.FromSlash(databaseName))); err != nil {
		return manifest, fmt.Errorf("backup manifest names a missing database snapshot: %w", err)
	}
	actual, total, err := checksumTree(stage, manifestName)
	if err != nil {
		return manifest, fmt.Errorf("checksum restored files: %w", err)
	}
	if total != manifest.TotalBytes || len(actual) != len(manifest.Files) {
		return manifest, errors.New("backup file set does not match its manifest")
	}
	for i := range manifest.Files {
		if actual[i] != manifest.Files[i] {
			return manifest, fmt.Errorf("backup checksum mismatch for %s", manifest.Files[i].Path)
		}
	}
	return manifest, nil
}
