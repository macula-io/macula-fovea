package core

import (
	"fmt"
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
func Init(dir string) int {
	h, f, err := LoadHeader(dir)
	if err != nil {
		fmt.Println(err)
		return 1
	}
	if len(f) > 0 {
		fmt.Println("fovea init — header findings first:")
		printFindings(f)
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
			fmt.Println(err)
			return 1
		}
		made++
	}
	fmt.Printf("fovea init — created %d cell skeleton(s), %d already present\n", made, skipped)
	if made > 0 {
		fmt.Println("every skeleton starts as status: unassessed / owner: unassigned — both are lint errors until written and claimed.")
	}
	return 0
}
