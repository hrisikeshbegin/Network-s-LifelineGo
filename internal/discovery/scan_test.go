package discovery

import "testing"

func TestHostsInCIDR_ExcludesNetworkAndBroadcast(t *testing.T) {
	ips, err := hostsInCIDR("192.168.1.0/24")
	if err != nil {
		t.Fatalf("hostsInCIDR returned error: %v", err)
	}

	if len(ips) != 254 {
		t.Fatalf("got %d hosts, want 254", len(ips))
	}

	first, last := ips[0].String(), ips[len(ips)-1].String()
	if first != "192.168.1.1" {
		t.Errorf("first host = %s, want 192.168.1.1", first)
	}
	if last != "192.168.1.254" {
		t.Errorf("last host = %s, want 192.168.1.254", last)
	}

	for _, ip := range ips {
		s := ip.String()
		if s == "192.168.1.0" {
			t.Error("hostsInCIDR included the network address 192.168.1.0")
		}
		if s == "192.168.1.255" {
			t.Error("hostsInCIDR included the broadcast address 192.168.1.255")
		}
	}
}

func TestNmapHostLineRe(t *testing.T) {
	tests := []struct {
		name         string
		line         string
		wantMatch    bool
		wantIP       string
		wantHostname string
	}{
		{
			name:         "host with hostname",
			line:         "Host: 192.168.1.5 (myhost.local)\tStatus: Up",
			wantMatch:    true,
			wantIP:       "192.168.1.5",
			wantHostname: "myhost.local",
		},
		{
			name:         "host without hostname",
			line:         "Host: 192.168.1.10 ()\tStatus: Up",
			wantMatch:    true,
			wantIP:       "192.168.1.10",
			wantHostname: "",
		},
		{
			name:      "host down is not matched",
			line:      "Host: 192.168.1.20 ()\tStatus: Down",
			wantMatch: false,
		},
		{
			name:      "unrelated line",
			line:      "# Nmap 7.94 scan initiated Fri Jan  1 00:00:00 2026 as: nmap -sn -oG - 192.168.1.0/24",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := nmapHostLineRe.FindStringSubmatch(tt.line)

			if !tt.wantMatch {
				if m != nil {
					t.Fatalf("expected no match, got %v", m)
				}
				return
			}

			if m == nil {
				t.Fatalf("expected a match, got none")
			}
			if m[1] != tt.wantIP {
				t.Errorf("IP = %q, want %q", m[1], tt.wantIP)
			}
			if m[2] != tt.wantHostname {
				t.Errorf("hostname = %q, want %q", m[2], tt.wantHostname)
			}
		})
	}
}
