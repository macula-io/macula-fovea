package core

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

const cellSkeleton = `id: %s
status: unassessed
owner: unassigned
threat:
  definition: ""

  manifestations: []
defense:
  detection: []
  countermeasures: []
  recovery: []
review_by: null
na_reason: null
notes: ""
`

// Init creates the full declared grid as unassessed skeletons.
func Init(dir string, stdout, stderr io.Writer) int {
	w := stdout
	h, f, err := LoadHeader(dir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if len(f) > 0 {
		fmt.Fprintln(stderr, "fovea init: fix the header findings first")
		printFindings(stderr, f)
		return 1
	}
	cellsDir := filepath.Join(dir, h.CellsDir)
	if err := os.MkdirAll(cellsDir, 0o755); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	made, skipped := 0, 0
	for _, id := range h.ExpectedCells() {
		// O_EXCL: init only ever adds files. A cell that exists but does not
		// parse is still someone's work and is never replaced by a skeleton.
		fh, err := os.OpenFile(filepath.Join(cellsDir, id+".yaml"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, fs.ErrExist) {
			skipped++
			continue
		}
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		_, werr := fmt.Fprintf(fh, cellSkeleton, id)
		if cerr := fh.Close(); werr == nil {
			werr = cerr
		}
		if werr != nil {
			fmt.Fprintln(stderr, werr)
			return 1
		}
		made++
	}
	fmt.Fprintf(w, "fovea init: created %d cell skeleton(s), %d already present\n", made, skipped)
	if made > 0 {
		fmt.Fprintln(w, "every skeleton starts as status: unassessed / owner: unassigned; both are lint errors until written and claimed.")
	}
	return 0
}
