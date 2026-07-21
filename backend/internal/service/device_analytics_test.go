package service

import "testing"

func TestDeviceCodeFromUserAgent(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		want      string
	}{
		{
			name:      "iphone mobile",
			userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 Mobile/15E148",
			want:      "iphone",
		},
		{
			name:      "huawei android",
			userAgent: "Mozilla/5.0 (Linux; Android 13; HUAWEI Mate 60 Pro) AppleWebKit/537.36 Chrome/124.0 Mobile Safari/537.36",
			want:      "huawei",
		},
		{
			name:      "xiaomi android",
			userAgent: "Mozilla/5.0 (Linux; Android 14; Xiaomi 14) AppleWebKit/537.36 Chrome/124.0 Mobile Safari/537.36",
			want:      "xiaomi",
		},
		{
			name:      "desktop",
			userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_5) AppleWebKit/537.36 Chrome/125.0 Safari/537.36",
			want:      "other",
		},
		{
			name:      "bot",
			userAgent: "Mozilla/5.0 Googlebot/2.1 (+http://www.google.com/bot.html)",
			want:      "bot",
		},
		{
			name:      "blank",
			userAgent: "",
			want:      "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DeviceCodeFromUserAgent(tt.userAgent); got != tt.want {
				t.Fatalf("DeviceCodeFromUserAgent() got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDeviceStatsFromUserAgents(t *testing.T) {
	stats := DeviceStatsFromUserAgents([]DeviceUserAgentCount{
		{UserAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) Mobile/15E148", Count: 3},
		{UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_5) AppleWebKit/537.36", Count: 1},
		{UserAgent: "", Count: 1},
	})

	if stats.TotalCount != 5 {
		t.Fatalf("total got %d, want 5", stats.TotalCount)
	}
	if len(stats.Items) != 3 {
		t.Fatalf("items got %d, want 3", len(stats.Items))
	}
	assertDeviceItem(t, stats.Items[0], "iphone", "苹果 iPhone", 3, 60)
	assertDeviceItem(t, stats.Items[1], "other", "其他设备", 1, 20)
	assertDeviceItem(t, stats.Items[2], "unknown", "未知", 1, 20)
}

func assertDeviceItem(t *testing.T, got DeviceBreakdownItem, code string, label string, count int64, percent float64) {
	t.Helper()
	if got.Code != code || got.Label != label || got.Count != count || got.Percent != percent {
		t.Fatalf("device item got %#v, want code=%s label=%s count=%d percent=%.1f", got, code, label, count, percent)
	}
}
