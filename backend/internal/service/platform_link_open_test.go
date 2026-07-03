package service

import (
	"testing"

	"ppk/backend/internal/model"
)

func TestDecideLandingPlatformLinkOpenUsesOfficialLinkWhenResolvedURLContainsPOIID(t *testing.T) {
	link := model.StorePlatformLink{
		TargetURL: "http://dpurl.cn/M5mipdSz",
	}
	resolvedURL := "https://w.dianping.com/cube/evoke/meituan.html?url=imeituan%3A%2F%2Fwww.meituan.com%2Fmrn%3Fmrn_biz%3Dmeishi%26mrn_entry%3Dfood-poi%26mrn_component%3Dfood-poi%26poiId%3D1953748828%26poiIdEncrypt%3Dabc123&utm_source=appshare"

	got := decideLandingPlatformLinkOpen(link, resolvedURL, nil)

	if got.OpenMode != platformLinkOpenModeOfficial {
		t.Fatalf("open mode got %q, want %q", got.OpenMode, platformLinkOpenModeOfficial)
	}
	if got.OpenURL != link.TargetURL {
		t.Fatalf("open url got %q, want %q", got.OpenURL, link.TargetURL)
	}
}

func TestDecideLandingPlatformLinkOpenUsesResolvedURLWhenNoPOIIDIsPresent(t *testing.T) {
	link := model.StorePlatformLink{
		TargetURL: "http://dpurl.cn/kTryCrPz",
	}
	resolvedURL := "https://www.meituan.com/cate/1357905152?utm_source=appshare&utm_fromapp=copy"

	got := decideLandingPlatformLinkOpen(link, resolvedURL, nil)

	if got.OpenMode != platformLinkOpenModeApp {
		t.Fatalf("open mode got %q, want %q", got.OpenMode, platformLinkOpenModeApp)
	}
	if got.OpenURL != resolvedURL {
		t.Fatalf("open url got %q, want %q", got.OpenURL, resolvedURL)
	}
}

func TestDecideLandingPlatformLinkOpenUsesResolvedURLWhenOnlyPOIIDEncryptIsPresent(t *testing.T) {
	link := model.StorePlatformLink{
		TargetURL: "http://dpurl.cn/onlyEncrypt",
	}
	resolvedURL := "https://w.dianping.com/cube/evoke/meituan.html?url=imeituan%3A%2F%2Fwww.meituan.com%2Fmrn%3FpoiIdEncrypt%3Dabc123&utm_source=appshare"

	got := decideLandingPlatformLinkOpen(link, resolvedURL, nil)

	if got.OpenMode != platformLinkOpenModeApp {
		t.Fatalf("open mode got %q, want %q", got.OpenMode, platformLinkOpenModeApp)
	}
	if got.OpenURL != resolvedURL {
		t.Fatalf("open url got %q, want %q", got.OpenURL, resolvedURL)
	}
}

func TestRedirectURLHasParamFindsNestedPOIIDBeforePOIIDEncrypt(t *testing.T) {
	resolvedURL := "https://w.dianping.com/cube/evoke/meituan.html?url=imeituan%3A%2F%2Fwww.meituan.com%2Fmrn%3Fmrn_biz%3Dmeishi%26poiId%3D1953748828%26poiIdEncrypt%3Dabc123&utm_source=appshare"

	if !redirectURLHasParam(resolvedURL, "poiId") {
		t.Fatalf("expected nested poiId to be detected")
	}
	if !redirectURLHasParam(resolvedURL, "poiIdEncrypt") {
		t.Fatalf("expected nested poiIdEncrypt to be detected")
	}
}
