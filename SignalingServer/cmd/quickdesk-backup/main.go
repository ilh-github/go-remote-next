package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"quickdesk/signaling/internal/backup"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "quickdesk-backup:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: quickdesk-backup <backup|restore> [flags]")
	}
	switch args[0] {
	case "backup":
		return runBackup(args[1:])
	case "restore":
		return runRestore(args[1:])
	default:
		return fmt.Errorf("unknown command %q; use backup or restore", args[0])
	}
}

func runBackup(args []string) error {
	flags := flag.NewFlagSet("backup", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	db := flags.String("db", "data/quickdesk.db", "SQLite database file")
	recordings := flags.String("recordings", "data/recordings", "recording artifact directory")
	updates := flags.String("updates", "data/updates", "update artifact directory")
	output := flags.String("output", "", "backup archive path")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *output == "" {
		return fmt.Errorf("backup requires -output")
	}
	manifest, err := backup.Create(context.Background(), backup.CreateOptions{
		DatabasePath:  *db,
		RecordingsDir: *recordings,
		UpdatesDir:    *updates,
		OutputPath:    *output,
	})
	if err != nil {
		return err
	}
	return writeJSON(manifest)
}

func runRestore(args []string) error {
	flags := flag.NewFlagSet("restore", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	archive := flags.String("archive", "", "backup archive path")
	db := flags.String("db", "data/quickdesk.db", "SQLite database destination")
	recordings := flags.String("recordings", "data/recordings", "recording artifact destination")
	updates := flags.String("updates", "data/updates", "update artifact destination")
	force := flags.Bool("force", false, "replace existing targets after moving them aside")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *archive == "" {
		return fmt.Errorf("restore requires -archive")
	}
	result, err := backup.Restore(context.Background(), backup.RestoreOptions{
		ArchivePath:   *archive,
		DatabasePath:  *db,
		RecordingsDir: *recordings,
		UpdatesDir:    *updates,
		Force:         *force,
	})
	if err != nil {
		return err
	}
	return writeJSON(result)
}

func writeJSON(value interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
