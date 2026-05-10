package watermark

import "sync"

// PartitionWatermark tracks the highest consecutively completed offset for a single partition.
// It is safe for concurrent use.
type PartitionWatermark struct {
	mu        sync.Mutex
	watermark int64
	completed map[int64]bool
}

func NewPartitionWatermark(startOffset int64) *PartitionWatermark {
	return &PartitionWatermark{
		watermark: startOffset,
		completed: make(map[int64]bool),
	}
}

// MarkDone marks offset as finished and returns (committableOffset, advanced).
// advanced is true when the watermark moved forward, meaning committableOffset is
// the new highest safe-to-commit Kafka offset.
func (pw *PartitionWatermark) MarkDone(offset int64) (int64, bool) {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	if offset > pw.watermark {
		pw.completed[offset] = true
		return pw.watermark - 1, false
	}

	if offset == pw.watermark {
		pw.watermark++
		for pw.completed[pw.watermark] {
			delete(pw.completed, pw.watermark)
			pw.watermark++
		}
		return pw.watermark - 1, true
	}

	return pw.watermark - 1, false
}

// Manager manages per-partition watermarks. It is safe for concurrent use.
type Manager struct {
	mu         sync.Mutex
	partitions map[int]*PartitionWatermark
}

func NewManager() *Manager {
	return &Manager{partitions: make(map[int]*PartitionWatermark)}
}

// Init registers a partition starting at startOffset. Subsequent calls for the same
// partition are no-ops, so it is safe to call on every fetched message.
func (m *Manager) Init(partition int, startOffset int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.partitions[partition]; !ok {
		m.partitions[partition] = NewPartitionWatermark(startOffset)
	}
}

// MarkDone delegates to the partition's watermark. Returns (-1, false) for unknown partitions.
func (m *Manager) MarkDone(partition int, offset int64) (int64, bool) {
	m.mu.Lock()
	pw := m.partitions[partition]
	m.mu.Unlock()
	if pw == nil {
		return -1, false
	}
	return pw.MarkDone(offset)
}
