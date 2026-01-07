package sync

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/Conflux-Chain/go-conflux-util/blockchain/sync/poll/testutil"
	"github.com/Conflux-Chain/go-conflux-util/blockchain/sync/process/db"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

type dummyDBOp struct{}

func (op dummyDBOp) Exec(tx *gorm.DB) error { return nil }

type testDBProcessor struct {
	batch   []testutil.Data
	batches [][]testutil.Data

	singles []testutil.Data

	drops [][]testutil.Data

	prev testutil.Data
}

func newTestDBProcessor() *testDBProcessor {
	return &testDBProcessor{}
}

func (p *testDBProcessor) requiresContinuous(data testutil.Data) {
	if len(p.prev.Hash) == 0 {
		return
	}

	if p.prev.Number+1 != data.Number {
		panic("Block number not in sequence")
	}

	if p.prev.Hash != data.ParentHash {
		panic("Block parent hash mismatch")
	}
}

func (p *testDBProcessor) Process(data testutil.Data) db.Operation {
	p.requiresContinuous(data)

	p.singles = append(p.singles, data)

	p.prev = data

	return dummyDBOp{}
}

func (p *testDBProcessor) Revert(data testutil.Data) db.Operation {
	if len(p.singles) == 0 {
		panic("No block to revert")
	}

	if data.Number < p.singles[0].Number {
		panic("Reverted block number too small")
	}

	if data.Number > p.singles[len(p.singles)-1].Number {
		panic("Reverted block number too large")
	}

	var ancestor int
	for p.singles[ancestor].Number != data.Number {
		ancestor++
	}

	p.drops = append(p.drops, slices.Clone(p.singles[ancestor:]))
	p.singles = p.singles[:ancestor]

	if ancestor == 0 {
		p.prev = testutil.Data{}
	} else {
		p.prev = p.singles[ancestor-1]
	}

	return dummyDBOp{}
}

func (p *testDBProcessor) BatchProcess(data testutil.Data) int {
	p.requiresContinuous(data)

	p.batch = append(p.batch, data)

	p.prev = data

	return len(p.batch)
}

func (p *testDBProcessor) BatchExec(tx *gorm.DB, createBatchSize int) error {
	return nil
}

func (p *testDBProcessor) BatchReset() {
	p.batches = append(p.batches, p.batch)
	p.batch = nil
}

func (p *testDBProcessor) assertData(t *testing.T, batches [][]uint64, singles []uint64, drops [][]uint64) {
	assert.Equal(t, batches, p.toNumberSlice2D(p.batches))
	assert.Nil(t, p.batch)
	assert.Equal(t, singles, p.toNumberSlice(p.singles))
	assert.Equal(t, drops, p.toNumberSlice2D(p.drops))
}

func (p *testDBProcessor) toNumberSlice(data []testutil.Data) []uint64 {
	if data == nil {
		return nil
	}

	result := make([]uint64, 0, len(data))

	for _, v := range data {
		result = append(result, v.Number)
	}

	return result
}

func (p *testDBProcessor) toNumberSlice2D(data [][]testutil.Data) [][]uint64 {
	if data == nil {
		return nil
	}

	result := make([][]uint64, 0, len(data))

	for _, v := range data {
		result = append(result, p.toNumberSlice(v))
	}

	return result
}

func (p *testDBProcessor) String() string {
	data := map[string]any{
		"batches": p.toNumberSlice2D(p.batches),
		"batch":   p.toNumberSlice(p.batch),
		"singles": p.toNumberSlice(p.singles),
		"drops":   p.toNumberSlice2D(p.drops),
		"prev":    p.prev.Number,
	}

	encoded, _ := json.MarshalIndent(data, "", "    ")

	return string(encoded)
}
