package sshx

import "testing"

func TestWhitelistAllowed(t *testing.T) {
	whitelist := NewWhitelist([]string{"uptime", "df", "systemctl status", "docker ps", "tail"})

	cases := []struct {
		name    string
		command string
		allowed bool
	}{
		{"exact match", "uptime", true},
		{"prefix with args", "df -h", true},
		{"multi-word prefix", "systemctl status nginx", true},
		{"trailing spaces trimmed", "  tail -n 20 /var/log/syslog  ", true},
		{"unknown command", "rm -rf /", false},
		{"empty", "", false},
		{"empty after trim", "   ", false},
		// 绕过尝试：拼接/替换命令
		{"semicolon injection", "uptime; rm -rf /", false},
		{"and injection", "uptime && rm -rf /", false},
		{"pipe injection", "df | xargs rm", false},
		{"redirect", "uptime > /etc/passwd", false},
		{"backtick substitution", "uptime `rm -rf /`", false},
		{"dollar substitution", "uptime $(rm -rf /)", false},
		{"newline injection", "uptime\nrm -rf /", false},
		{"prefix must be word boundary", "dfx", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := whitelist.Allowed(tc.command); got != tc.allowed {
				t.Fatalf("Allowed(%q) = %v, want %v", tc.command, got, tc.allowed)
			}
		})
	}
}

func TestWhitelistNilAndEmpty(t *testing.T) {
	var nilWhitelist *Whitelist
	if nilWhitelist.Allowed("uptime") {
		t.Fatal("nil whitelist must reject everything")
	}
	if NewWhitelist(nil).Allowed("uptime") {
		t.Fatal("empty whitelist must reject everything")
	}
}

func TestWhitelistPrefixes(t *testing.T) {
	whitelist := NewWhitelist([]string{" uptime ", "", "df"})
	got := whitelist.Prefixes()
	if len(got) != 2 || got[0] != "uptime" || got[1] != "df" {
		t.Fatalf("unexpected prefixes: %v", got)
	}
}
