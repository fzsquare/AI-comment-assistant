package service

import (
	"archive/zip"
	"bytes"
	"testing"
	"time"
)

func TestParseReviewCrawlXLSXKeepsRepeatedUsers(t *testing.T) {
	data := testReviewXLSX(t, [][]string{
		{"ID", "用户名", "评分", "时间", "内容"},
		{"100", "同一个用户", "50", "2026-07-01 10:00:00", "第一次评论内容很完整"},
		{"101", "同一个用户", "45", "2026-07-02 11:00:00", "第二次评论也要保留"},
	})

	rows, err := ParseReviewCrawlXLSX(data)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows got %d, want 2", len(rows))
	}
	if rows[0].SourceReviewRef != "100" || rows[1].SourceReviewRef != "101" {
		t.Fatalf("source refs got %q/%q", rows[0].SourceReviewRef, rows[1].SourceReviewRef)
	}
	if rows[0].UserName != rows[1].UserName {
		t.Fatalf("expected repeated username to be preserved, got %q/%q", rows[0].UserName, rows[1].UserName)
	}
	if rows[0].RatingNormalized == nil || *rows[0].RatingNormalized != 5 {
		t.Fatalf("rating normalized got %#v, want 5", rows[0].RatingNormalized)
	}
	want := time.Date(2026, 7, 2, 11, 0, 0, 0, time.Local)
	if rows[1].ReviewTime == nil || !rows[1].ReviewTime.Equal(want) {
		t.Fatalf("review time got %#v, want %v", rows[1].ReviewTime, want)
	}
}

func TestParseReviewCrawlXLSXRequiresExpectedHeaders(t *testing.T) {
	data := testReviewXLSX(t, [][]string{
		{"ID", "用户", "评分", "时间", "内容"},
		{"100", "张三", "50", "2026-07-01 10:00:00", "内容"},
	})

	if _, err := ParseReviewCrawlXLSX(data); err == nil {
		t.Fatal("expected header error")
	}
}

func testReviewXLSX(t *testing.T, rows [][]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	addZipFile(t, zw, "[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
  <Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
</Types>`)
	addZipFile(t, zw, "xl/workbook.xml", `<?xml version="1.0" encoding="UTF-8"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheets><sheet name="Sheet1" sheetId="1" r:id="rId1" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"/></sheets>
</workbook>`)
	addZipFile(t, zw, "xl/worksheets/sheet1.xml", testSheetXML(rows))
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}

func addZipFile(t *testing.T, zw *zip.Writer, name string, content string) {
	t.Helper()
	w, err := zw.Create(name)
	if err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	if _, err := w.Write([]byte(content)); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func testSheetXML(rows [][]string) string {
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	for rowIndex, row := range rows {
		b.WriteString(`<row>`)
		for colIndex, value := range row {
			cell := string(rune('A'+colIndex)) + string(rune('1'+rowIndex))
			b.WriteString(`<c r="` + cell + `" t="inlineStr"><is><t>`)
			b.WriteString(value)
			b.WriteString(`</t></is></c>`)
		}
		b.WriteString(`</row>`)
	}
	b.WriteString(`</sheetData></worksheet>`)
	return b.String()
}
