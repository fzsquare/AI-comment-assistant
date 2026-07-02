package service

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type ReviewCrawlImportRow struct {
	SourceReviewRef  string
	UserName         string
	RatingRaw        string
	RatingNormalized *float64
	ReviewTime       *time.Time
	Content          string
}

var expectedReviewCrawlHeaders = []string{"ID", "用户名", "评分", "时间", "内容"}

func ParseReviewCrawlXLSX(data []byte) ([]ReviewCrawlImportRow, error) {
	if len(data) == 0 {
		return nil, errors.New("empty xlsx")
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open xlsx: %w", err)
	}

	sharedStrings, err := readSharedStrings(reader)
	if err != nil {
		return nil, err
	}
	sheetData, err := readXLSXFile(reader, "xl/worksheets/sheet1.xml")
	if err != nil {
		return nil, err
	}

	rows, err := parseWorksheetRows(sheetData, sharedStrings)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errors.New("xlsx has no rows")
	}
	if err := validateReviewCrawlHeaders(rows[0]); err != nil {
		return nil, err
	}

	out := make([]ReviewCrawlImportRow, 0, len(rows)-1)
	for _, row := range rows[1:] {
		row = padCells(row, len(expectedReviewCrawlHeaders))
		if rowIsBlank(row) {
			continue
		}
		reviewTime := parseReviewTime(row[3])
		out = append(out, ReviewCrawlImportRow{
			SourceReviewRef:  strings.TrimSpace(row[0]),
			UserName:         strings.TrimSpace(row[1]),
			RatingRaw:        strings.TrimSpace(row[2]),
			RatingNormalized: normalizeRating(row[2]),
			ReviewTime:       reviewTime,
			Content:          strings.TrimSpace(row[4]),
		})
	}
	return out, nil
}

func readXLSXFile(reader *zip.Reader, name string) ([]byte, error) {
	for _, file := range reader.File {
		if file.Name != name {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return io.ReadAll(rc)
	}
	return nil, fmt.Errorf("xlsx missing %s", name)
}

type worksheetXML struct {
	Rows []worksheetRowXML `xml:"sheetData>row"`
}

type worksheetRowXML struct {
	Cells []worksheetCellXML `xml:"c"`
}

type worksheetCellXML struct {
	Ref       string `xml:"r,attr"`
	Type      string `xml:"t,attr"`
	Value     string `xml:"v"`
	InlineStr struct {
		Text string `xml:"t"`
	} `xml:"is"`
}

func parseWorksheetRows(data []byte, sharedStrings []string) ([][]string, error) {
	var sheet worksheetXML
	if err := xml.Unmarshal(data, &sheet); err != nil {
		return nil, fmt.Errorf("parse sheet: %w", err)
	}
	rows := make([][]string, 0, len(sheet.Rows))
	for _, row := range sheet.Rows {
		cells := []string{}
		for seq, cell := range row.Cells {
			index := cellColumnIndex(cell.Ref)
			if index < 0 {
				index = seq
			}
			for len(cells) <= index {
				cells = append(cells, "")
			}
			cells[index] = cellText(cell, sharedStrings)
		}
		rows = append(rows, cells)
	}
	return rows, nil
}

func cellText(cell worksheetCellXML, sharedStrings []string) string {
	switch cell.Type {
	case "inlineStr":
		return strings.TrimSpace(cell.InlineStr.Text)
	case "s":
		index, err := strconv.Atoi(strings.TrimSpace(cell.Value))
		if err == nil && index >= 0 && index < len(sharedStrings) {
			return strings.TrimSpace(sharedStrings[index])
		}
		return ""
	default:
		return strings.TrimSpace(cell.Value)
	}
}

type sharedStringsXML struct {
	Items []sharedStringItemXML `xml:"si"`
}

type sharedStringItemXML struct {
	Text string `xml:"t"`
	Runs []struct {
		Text string `xml:"t"`
	} `xml:"r"`
}

func readSharedStrings(reader *zip.Reader) ([]string, error) {
	data, err := readXLSXFile(reader, "xl/sharedStrings.xml")
	if err != nil {
		if strings.Contains(err.Error(), "missing") {
			return nil, nil
		}
		return nil, err
	}
	var parsed sharedStringsXML
	if err := xml.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("parse shared strings: %w", err)
	}
	values := make([]string, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		if item.Text != "" {
			values = append(values, item.Text)
			continue
		}
		var builder strings.Builder
		for _, run := range item.Runs {
			builder.WriteString(run.Text)
		}
		values = append(values, builder.String())
	}
	return values, nil
}

func cellColumnIndex(ref string) int {
	if ref == "" {
		return -1
	}
	column := 0
	seenLetter := false
	for _, r := range ref {
		if !unicode.IsLetter(r) {
			break
		}
		seenLetter = true
		column = column*26 + int(unicode.ToUpper(r)-'A'+1)
	}
	if !seenLetter {
		return -1
	}
	return column - 1
}

func validateReviewCrawlHeaders(headers []string) error {
	headers = padCells(headers, len(expectedReviewCrawlHeaders))
	for i, expected := range expectedReviewCrawlHeaders {
		if strings.TrimSpace(headers[i]) != expected {
			return fmt.Errorf("xlsx header %d got %q, want %q", i+1, headers[i], expected)
		}
	}
	return nil
}

func padCells(row []string, size int) []string {
	out := append([]string{}, row...)
	for len(out) < size {
		out = append(out, "")
	}
	return out
}

func rowIsBlank(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func parseReviewTime(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02"} {
		parsed, err := time.ParseInLocation(layout, raw, time.Local)
		if err == nil {
			return &parsed
		}
	}
	serial, err := strconv.ParseFloat(raw, 64)
	if err == nil && serial > 0 {
		base := time.Date(1899, 12, 30, 0, 0, 0, 0, time.Local)
		parsed := base.Add(time.Duration(serial * float64(24*time.Hour)))
		return &parsed
	}
	return nil
}

func normalizeRating(raw string) *float64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil
	}
	if value > 5 {
		value = value / 10
	}
	if value < 0 {
		value = 0
	}
	if value > 5 {
		value = 5
	}
	return &value
}
