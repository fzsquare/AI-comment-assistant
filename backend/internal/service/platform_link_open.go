package service

import (
	"strings"
	"sync"

	"ppk/backend/internal/model"
)

const (
	platformLinkOpenModeOfficial = "official_link"
	platformLinkOpenModeApp      = "app_link"
)

type LandingPlatformLink struct {
	ID           uint   `json:"id"`
	PlatformCode string `json:"platformCode"`
	PlatformName string `json:"platformName"`
	ButtonText   string `json:"buttonText"`
	TargetURL    string `json:"targetUrl"`
	BackupURL    string `json:"backupUrl"`
	SortNo       int    `json:"sortNo"`
	Status       int    `json:"status"`
	OpenMode     string `json:"openMode"`
	OpenURL      string `json:"openUrl"`
}

type landingPlatformLinkOpen struct {
	OpenMode string
	OpenURL  string
}

func buildLandingPlatformLinks(links []model.StorePlatformLink) []LandingPlatformLink {
	if len(links) == 0 {
		return nil
	}
	out := make([]LandingPlatformLink, len(links))
	var wg sync.WaitGroup
	wg.Add(len(links))
	for i, link := range links {
		i := i
		link := link
		go func() {
			defer wg.Done()
			out[i] = landingPlatformLinkFromModel(link)
		}()
	}
	wg.Wait()
	return out
}

func landingPlatformLinkFromModel(link model.StorePlatformLink) LandingPlatformLink {
	open := decideLandingPlatformLinkOpen(link)
	return LandingPlatformLink{
		ID:           link.ID,
		PlatformCode: link.PlatformCode,
		PlatformName: link.PlatformName,
		ButtonText:   link.ButtonText,
		TargetURL:    link.TargetURL,
		BackupURL:    link.BackupURL,
		SortNo:       link.SortNo,
		Status:       link.Status,
		OpenMode:     open.OpenMode,
		OpenURL:      open.OpenURL,
	}
}

func decideLandingPlatformLinkOpen(link model.StorePlatformLink) landingPlatformLinkOpen {
	targetURL := strings.TrimSpace(link.TargetURL)
	backupURL := strings.TrimSpace(link.BackupURL)
	if targetURL != "" {
		return landingPlatformLinkOpen{
			OpenMode: platformLinkOpenModeOfficial,
			OpenURL:  targetURL,
		}
	}
	if backupURL != "" {
		return landingPlatformLinkOpen{
			OpenMode: platformLinkOpenModeOfficial,
			OpenURL:  backupURL,
		}
	}
	if appURL := platformHomeAppOpenURL(link.PlatformCode); appURL != "" {
		return landingPlatformLinkOpen{
			OpenMode: platformLinkOpenModeApp,
			OpenURL:  appURL,
		}
	}
	return landingPlatformLinkOpen{
		OpenMode: platformLinkOpenModeOfficial,
		OpenURL:  "",
	}
}

func platformHomeAppOpenURL(platformCode string) string {
	switch strings.ToLower(strings.TrimSpace(platformCode)) {
	case "meituan":
		return "imeituan://"
	case "dianping":
		return "dianping://"
	case "douyin":
		return "snssdk1128://"
	case "xiaohongshu":
		return "xhsdiscover://"
	default:
		return ""
	}
}
