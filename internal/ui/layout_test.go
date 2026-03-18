package ui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestBoundedContentWidth(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		total     int
		minWidth  int
		maxWidth  int
		gutter    int
		wantWidth int
	}{
		{
			name:      "caps at max width on wide terminals",
			total:     140,
			minWidth:  inputPanelMinWidth,
			maxWidth:  inputPanelMaxWidth,
			gutter:    inputPanelGutter,
			wantWidth: inputPanelMaxWidth,
		},
		{
			name:      "uses available width on medium terminals",
			total:     92,
			minWidth:  inputPanelMinWidth,
			maxWidth:  inputPanelMaxWidth,
			gutter:    inputPanelGutter,
			wantWidth: 78,
		},
		{
			name:      "drops below min width on narrow terminals",
			total:     68,
			minWidth:  inputPanelMinWidth,
			maxWidth:  inputPanelMaxWidth,
			gutter:    inputPanelGutter,
			wantWidth: 54,
		},
		{
			name:      "falls back to inner width when gutter is larger than the viewport",
			total:     12,
			minWidth:  inputPanelMinWidth,
			maxWidth:  inputPanelMaxWidth,
			gutter:    40,
			wantWidth: 8,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			gotWidth := boundedContentWidth(
				testCase.total,
				testCase.minWidth,
				testCase.maxWidth,
				testCase.gutter,
			)
			if gotWidth != testCase.wantWidth {
				t.Fatalf("boundedContentWidth() = %d, want %d", gotWidth, testCase.wantWidth)
			}
		})
	}
}

func TestInputTopPadding(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		totalHeight   int
		contentHeight int
		wantPadding   int
	}{
		{
			name:          "adds top padding on tall terminals",
			totalHeight:   40,
			contentHeight: 12,
			wantPadding:   5,
		},
		{
			name:          "keeps compact terminals flush",
			totalHeight:   16,
			contentHeight: 12,
			wantPadding:   0,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			gotPadding := inputTopPadding(testCase.totalHeight, testCase.contentHeight)
			if gotPadding != testCase.wantPadding {
				t.Fatalf("inputTopPadding() = %d, want %d", gotPadding, testCase.wantPadding)
			}
		})
	}
}

func TestBuildPanelTopBorderWidthMatchesBody(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		title      string
		innerWidth int
	}{
		{
			name:       "plain border keeps full width",
			title:      "",
			innerWidth: 72,
		},
		{
			name:       "titled border keeps full width",
			title:      accentTextStyle.Render(" Repository "),
			innerWidth: 72,
		},
		{
			name:       "short titled border keeps full width",
			title:      successText.Render(" Files "),
			innerWidth: 40,
		},
	}

	borderStyle := lipgloss.NewStyle().Foreground(colorAccent).Background(colorBG)

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := buildPanelTopBorder(testCase.title, testCase.innerWidth, borderStyle)
			wantWidth := testCase.innerWidth + 4
			if gotWidth := lipgloss.Width(got); gotWidth != wantWidth {
				t.Fatalf("buildPanelTopBorder() width = %d, want %d", gotWidth, wantWidth)
			}
		})
	}
}
