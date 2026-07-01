package supervisor

import (
	"testing"
	"time"
)

func TestRestartGuardCrashLoop(t *testing.T) {
	var g restartGuard
	window := 60 * time.Second
	burst := 3

	for i := 0; i < burst-1; i++ {
		if err := g.beforeRestart(burst, window, 0); err != nil {
			t.Fatalf("restart %d: %v", i, err)
		}
	}
	if err := g.beforeRestart(burst, window, 0); err == nil {
		t.Fatal("expected crash loop error")
	}
}

func TestRestartGuardMaxRestart(t *testing.T) {
	var g restartGuard
	if err := g.beforeRestart(10, time.Minute, 2); err != nil {
		t.Fatal(err)
	}
	if err := g.beforeRestart(10, time.Minute, 2); err != nil {
		t.Fatal(err)
	}
	if err := g.beforeRestart(10, time.Minute, 2); err == nil {
		t.Fatal("expected max restart error")
	}
}
