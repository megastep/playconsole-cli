package commands

import "testing"

func TestRootCmdExposesEditModeFlag(t *testing.T) {
	flag := GetRootCmd().PersistentFlags().Lookup("edit-mode")
	if flag == nil {
		t.Fatal("expected root command to expose --edit-mode")
	}
	if got, want := flag.DefValue, "live"; got != want {
		t.Fatalf("default edit-mode = %q, want %q", got, want)
	}
}
