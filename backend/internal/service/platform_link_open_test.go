package service

import (
	"testing"

	"ppk/backend/internal/model"
)

func TestDecideLandingPlatformLinkOpenUsesConfiguredURLForSupportedPlatforms(t *testing.T) {
	tests := []struct {
		name         string
		platformCode string
		targetURL    string
	}{
		{name: "meituan", platformCode: "meituan", targetURL: "https://w.dianping.com/cube/evoke/meituan.html"},
		{name: "dianping", platformCode: "dianping", targetURL: "https://m.dianping.com/shop/123456"},
		{name: "douyin", platformCode: "douyin", targetURL: "https://www.douyin.com/poi/example"},
		{name: "xiaohongshu", platformCode: "xiaohongshu", targetURL: "https://www.xiaohongshu.com/explore/example"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decideLandingPlatformLinkOpen(model.StorePlatformLink{
				PlatformCode: tt.platformCode,
				TargetURL:    tt.targetURL,
			})

			if got.OpenMode != platformLinkOpenModeOfficial {
				t.Fatalf("open mode got %q, want %q", got.OpenMode, platformLinkOpenModeOfficial)
			}
			if got.OpenURL != tt.targetURL {
				t.Fatalf("open url got %q, want configured url %q", got.OpenURL, tt.targetURL)
			}
		})
	}
}

func TestDecideLandingPlatformLinkOpenFallsBackToAppSchemeOnlyWithoutConfiguredURL(t *testing.T) {
	got := decideLandingPlatformLinkOpen(model.StorePlatformLink{PlatformCode: "meituan"})

	if got.OpenMode != platformLinkOpenModeApp {
		t.Fatalf("open mode got %q, want %q", got.OpenMode, platformLinkOpenModeApp)
	}
	if got.OpenURL != "imeituan://" {
		t.Fatalf("open url got %q, want %q", got.OpenURL, "imeituan://")
	}
}

func TestDecideLandingPlatformLinkOpenUsesBackupURLForUnknownPlatform(t *testing.T) {
	backupURL := "https://example.com/backup"
	got := decideLandingPlatformLinkOpen(model.StorePlatformLink{
		PlatformCode: "other",
		BackupURL:    backupURL,
	})

	if got.OpenMode != platformLinkOpenModeOfficial {
		t.Fatalf("open mode got %q, want %q", got.OpenMode, platformLinkOpenModeOfficial)
	}
	if got.OpenURL != backupURL {
		t.Fatalf("open url got %q, want backup url %q", got.OpenURL, backupURL)
	}
}
