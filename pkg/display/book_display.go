package display

import (
	"fmt"
	"strings"
	"sync"

	"github.com/yuki-inoue-eng/lapuacore/domains/insights"
)

type BookEntry struct {
	Title string
	OB    insights.OrderBook
}

type BookDisplay struct {
	mu    sync.Mutex
	depth int
	books []BookEntry
}

func NewBookDisplay(depth int, books []BookEntry) *BookDisplay {
	return &BookDisplay{
		depth: depth,
		books: books,
	}
}

func (d *BookDisplay) Books() []BookEntry {
	return d.books
}

func (d *BookDisplay) Render() {
	d.mu.Lock()
	defer d.mu.Unlock()

	columns := make([][]string, len(d.books))
	widths := make([]int, len(d.books))
	maxRows := 0

	for i, entry := range d.books {
		columns[i], widths[i] = d.buildBookLines(entry.Title, entry.OB)
		if len(columns[i]) > maxRows {
			maxRows = len(columns[i])
		}
	}

	// Clear screen and move cursor to top-left
	fmt.Print("\033[2J\033[H")

	for row := 0; row < maxRows; row++ {
		for col := 0; col < len(d.books); col++ {
			line := ""
			if row < len(columns[col]) {
				line = columns[col][row]
			}
			if col < len(d.books)-1 {
				padding := widths[col] - visibleLen(line)
				if padding < 0 {
					padding = 0
				}
				fmt.Printf("%s%s  ", line, strings.Repeat(" ", padding))
			} else {
				fmt.Print(line)
			}
		}
		fmt.Println()
	}
}

func (d *BookDisplay) buildBookLines(title string, ob insights.OrderBook) ([]string, int) {
	bestAsk := ob.GetBestAsk()
	bestBid := ob.GetBestBid()
	if bestAsk == nil || bestBid == nil {
		line := title + ": waiting for data..."
		return []string{line}, len(line)
	}

	asks := ob.GetAsks(d.depth)
	bids := ob.GetBids(d.depth)

	// Build data lines first to measure max width
	var dataLines []string
	for i := len(asks) - 1; i >= 0; i-- {
		dataLines = append(dataLines, fmt.Sprintf("  \033[31mAsk %s: %s\033[0m",
			asks[i].Price.StringFixed(2), asks[i].Volume.String()))
	}
	for i := 0; i < len(bids); i++ {
		dataLines = append(dataLines, fmt.Sprintf("  \033[32mBid %s: %s\033[0m",
			bids[i].Price.StringFixed(2), bids[i].Volume.String()))
	}

	maxWidth := 0
	for _, line := range dataLines {
		if w := visibleLen(line); w > maxWidth {
			maxWidth = w
		}
	}

	// Header: 2-space indent + '=' padding to match data content width
	innerWidth := maxWidth - 2 // subtract the 2-space indent
	headerText := fmt.Sprintf(" %s ", title)
	padTotal := innerWidth - len(headerText)
	if padTotal < 0 {
		padTotal = 0
	}
	padLeft := padTotal / 2
	padRight := padTotal - padLeft
	header := "  " + strings.Repeat("=", padLeft) + headerText + strings.Repeat("=", padRight)

	// Spread: 2-space indent + '-' padding to match data content width
	spreadText := fmt.Sprintf(" spread(%s) ", bestAsk.Price.Sub(bestBid.Price).StringFixed(2))
	padTotal = innerWidth - len(spreadText)
	if padTotal < 0 {
		padTotal = 0
	}
	padLeft = padTotal / 2
	padRight = padTotal - padLeft
	spread := "  " + strings.Repeat("-", padLeft) + spreadText + strings.Repeat("-", padRight)

	var lines []string
	lines = append(lines, header)
	lines = append(lines, dataLines[:len(asks)]...)
	lines = append(lines, spread)
	lines = append(lines, dataLines[len(asks):]...)
	lines = append(lines, "")

	return lines, maxWidth
}

// visibleLen returns the visible length of a string, excluding ANSI escape codes.
func visibleLen(s string) int {
	visible := 0
	inEsc := false
	for _, r := range s {
		if r == '\033' {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		visible++
	}
	return visible
}
