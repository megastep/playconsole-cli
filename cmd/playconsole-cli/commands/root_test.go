package commands

import (
	"testing"

	"github.com/spf13/viper"
)

func TestRootCmdExposesEditModeFlag(t *testing.T) {
	flag := GetRootCmd().PersistentFlags().Lookup("edit-mode")
	if flag == nil {
		t.Fatal("expected root command to expose --edit-mode")
	}
	if got, want := flag.DefValue, "live"; got != want {
		t.Fatalf("default edit-mode = %q, want %q", got, want)
	}
}

func TestRootCmdBindsEditModeFlagToViper(t *testing.T) {
	viper.Reset()
	t.Cleanup(func() {
		_ = GetRootCmd().PersistentFlags().Set("edit-mode", "live")
	})
	viper.BindPFlag("edit-mode", GetRootCmd().PersistentFlags().Lookup("edit-mode"))

	if err := GetRootCmd().PersistentFlags().Set("edit-mode", "stage"); err != nil {
		t.Fatalf("set edit-mode flag: %v", err)
	}

	if got, want := viper.GetString("edit-mode"), "stage"; got != want {
		t.Fatalf("viper edit-mode = %q, want %q", got, want)
	}
}
