package safety

import "testing"

func TestCheckAllowsNormalCommand(t *testing.T) {
	if err := Check("go test ./..."); err != nil {
		t.Fatalf("Check() error = %v", err)
	}
}

func TestCheckRejectsEmptyCommand(t *testing.T) {
	if err := Check("   "); err == nil {
		t.Fatal("Check() expected error for empty command")
	}
}

func TestCheckRejectsDangerousCommands(t *testing.T) {
	cases := []string{
		"rm -rf /",
		":(){ :|:& };:",
		"mkfs.ext4 /dev/sda",
		"dd if=/dev/zero of=/dev/sda",
		"shutdown -h now",
	}
	for _, cmd := range cases {
		if err := Check(cmd); err == nil {
			t.Fatalf("Check(%q) expected error", cmd)
		}
	}
}
