package handler

import "testing"

// QD-909 draws one line through the whole data-channel audit trail: report
// metadata, never content. The clipboard side holds it hard - there is no field
// for the text at all, and the comment on the model says why, because a row that
// could say what was copied would turn the trail into a credential store every
// later reader of the database inherits.
//
// The file side has the same intent and one hole in it. `filename` is bounded
// only by length, so a controller can report a full absolute path and the server
// stores it verbatim, and the client renders it verbatim. The result is a row
// that says not "a transfer happened" but "this user keeps
// /Users/someone/.ssh/id_rsa in this directory" - the directory layout of the
// controlled machine, which is exactly the kind of thing the clipboard rule
// exists to keep out of a table that outlives the session.
//
// The bound is on the *shape*, not the length: a name is a leaf. A name that
// carries a directory is a path, and a path is information about the machine
// rather than about the transfer.
func TestFileTransferAuditRejectsAPathAsFilename(t *testing.T) {
	cases := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{name: "a plain leaf is the normal case", filename: "report.pdf", wantErr: false},
		{name: "a leaf with spaces is still a leaf", filename: "Q3 report final.pdf", wantErr: false},
		{name: "an absolute posix path is a path", filename: "/Users/someone/.ssh/id_rsa", wantErr: true},
		{name: "a windows path is a path", filename: `C:\Users\someone\.ssh\id_rsa`, wantErr: true},
		{name: "a UNC path is a path", filename: `\\server\share\keys\id_rsa`, wantErr: true},
		{name: "a relative path is still a path", filename: "docs/../../../etc/passwd", wantErr: true},
		{name: "a trailing separator is a path", filename: "secrets/", wantErr: true},
		{name: "a bare drive letter is a path", filename: "C:", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateFileTransferAudit(fileTransferAuditRequest{
				DeviceID:   "123456789",
				Direction:  "download",
				Status:     "completed",
				TransferID: "transfer-1",
				Filename:   tc.filename,
			})
			if tc.wantErr && err == nil {
				t.Fatalf("filename %q was accepted; a path is information about the machine, not about the transfer", tc.filename)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("filename %q was rejected: %v", tc.filename, err)
			}
		})
	}
}

// The bound has to hold in the other direction too: a legitimate leaf must never
// be refused, or the guard is not a guard but a different bug. Names that look
// like paths but are not are the cases that decide it.
func TestFileTransferAuditStillAcceptsOrdinaryFilenames(t *testing.T) {
	ok := []string{
		"report.pdf",
		"Q3 report final.pdf",
		"archive.tar.gz",
		"notes (old).txt",
		"a",
		".gitignore",
		"file..name.txt",
	}
	for _, filename := range ok {
		if err := validateFileTransferAudit(fileTransferAuditRequest{
			DeviceID:   "123456789",
			Direction:  "upload",
			Status:     "completed",
			TransferID: "transfer-2",
			Filename:   filename,
		}); err != nil {
			t.Errorf("filename %q was rejected: %v", filename, err)
		}
	}
}
