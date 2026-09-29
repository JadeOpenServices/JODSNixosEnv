package buildlimits

import (
	"slices"
	"testing"
)

func TestArgsFor(t *testing.T) {
	for _, tc := range []struct {
		memory uint64
		cpus   int
		want   []string
	}{
		{0, 8, nil},
		{32 * gib, 16, nil},
		{16 * gib, 16, nil},
		{8 * gib, 4, []string{"--max-jobs", "1", "--cores", "4"}},
		{7*gib + 800<<20, 4, []string{"--max-jobs", "1", "--cores", "3"}},
		{12 * gib, 2, []string{"--max-jobs", "1", "--cores", "2"}},
		{1 * gib, 8, []string{"--max-jobs", "1", "--cores", "1"}},
	} {
		if got := argsFor(tc.memory, tc.cpus); !slices.Equal(got, tc.want) {
			t.Errorf("argsFor(%d, %d) = %v, want %v", tc.memory, tc.cpus, got, tc.want)
		}
	}
}

func TestMemoryFromMeminfo(t *testing.T) {
	data := []byte("MemTotal:        8000000 kB\nMemFree:         100 kB\nSwapTotal:       2000000 kB\n")
	if got, want := memoryFromMeminfo(data), uint64(10000000)<<10; got != want {
		t.Fatalf("got %d, want %d", got, want)
	}
	// Live media: 12 GiB RAM, 8 GiB install swap, 7 GiB in tmpfs.
	live := []byte("MemTotal: 12582912 kB\nShmem: 7340032 kB\nSwapTotal: 8388608 kB\n")
	if got, want := memoryFromMeminfo(live), uint64(13)*gib; got != want {
		t.Fatalf("live media = %d, want %d", got, want)
	}
	if args := argsFor(memoryFromMeminfo(live), 4); len(args) == 0 {
		t.Fatal("live media with tmpfs-held RAM ran unlimited")
	}
	if got := memoryFromMeminfo([]byte("MemTotal: 100 kB\nShmem: 200 kB\n")); got != 1 {
		t.Fatalf("shmem above total = %d, want 1", got)
	}
	if got := memoryFromMeminfo([]byte("MemTotal: bogus kB\n")); got != 0 {
		t.Fatalf("malformed meminfo = %d, want 0", got)
	}
}
