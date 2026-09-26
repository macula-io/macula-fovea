package core

import (
	"fmt"
	"io"
	"os"
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
		fmt.Fprintln(w, err)
		return 1
	}
	if len(f) > 0 {
		fmt.Fprintln(w, "fovea init — header findings first:")
		printFindings(w, f)
		return 1
	}
	cells, _, _ := LoadCells(dir, h)
	made, skipped := 0, 0
	for _, id := range h.ExpectedCells() {
		if _, ok := cells[id]; ok {
			skipped++
			continue
		}
		p := dir + "/" + h.CellsDir + id + ".yaml"
		if err := os.WriteFile(p, []byte(fmt.Sprintf(cellSkeleton, id)), 0o644); err != nil {
			fmt.Fprintln(w, err)
			return 1
		}
		made++
	}
	fmt.Fprintf(w, "fovea init — created %d cell skeleton(s), %d already present\n", made, skipped)
	if made > 0 {
		fmt.Fprintln(w, "every skeleton starts as status: unassessed / owner: unassigned — both are lint errors until written and claimed.")
	}
	return 0
}
