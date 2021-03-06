package tuttobene

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/juju/errors"
	"github.com/sahilm/fuzzy"
	"github.com/shopspring/decimal"
	"github.com/tealeg/xlsx"
)

var Titles = map[MenuRowType]string{
	Primo:       "primi piatti",
	Secondo:     "secondi piatti",
	Contorno:    "contorni",
	Vegetariano: "piatti vegetariani",
	Frutta:      "frutta",
	Dolce:       "dolci",
	Panino:      "i nostri panini espressi",
}

// ParseMenuBytes takes io.ReaderAt of an XLSX file and returns a populated
// menu struct.
func ParseMenuBytes(bs []byte) (*Menu, error) {
	f, err := xlsx.OpenBinary(bs)
	if err != nil {
		return nil, errors.Annotate(err, "while opening binary")
	}

	if len(f.Sheet) == 0 {
		return nil, errors.New("no sheets in file")
	}

	// Menu is expected to be on the first sheet
	return ParseSheet(f.Sheets[0])
}

// ParseMenuFile takes the path to an XLSX file and returns a populated
// menu struct.
func ParseMenuFile(path string) (*Menu, error) {
	f, err := xlsx.OpenFile(path)
	if err != nil {
		return nil, errors.Annotatef(err, "while opening file %s", path)
	}

	if len(f.Sheet) == 0 {
		return nil, errors.New(fmt.Sprintf("no sheets in file %s", path))
	}

	// Menu is expected to be on the first sheet
	return ParseSheet(f.Sheets[0])
}

// ParseSheet takes an xlsx.Sheet and returns a populated menu struct.
func ParseSheet(s *xlsx.Sheet) (*Menu, error) {
	// attempt at having a sensible number of rows required in menu
	if len(s.Rows) < 12 {
		return nil, errors.New(fmt.Sprintf("not enough rows: %d", len(s.Rows)))
	}


	// Check tuttobene menu format (dishes in column 0 or 1)
	col := 0
	if len(s.Rows[0].Cells) >= 2 {
		sheetTitle := s.Rows[0].Cells[1].String()
		sheetTitle = strings.TrimSpace(sheetTitle)
		sheetTitle = strings.ToLower(sheetTitle)
		if sheetTitle == "tuttobene" {
			col = 1
		}
	}

	var nameCol, priceCol []string
	for _, r := range s.Rows {
		if len(r.Cells) >= col+1 {
			nameCol = append(nameCol, r.Cells[col].String())
		}
		if len(r.Cells) >= col+2 {
			priceCol = append(priceCol, r.Cells[col + 1].String())
		}
	}

	return ParseMenuCells(nameCol, priceCol)
}

func normalizeDish(r *MenuRow) *MenuRow {
	if r.Type == Contorno {
		tab := []struct {
			Find, Replace string
		}{
			{
				Find:    "grigliat",
				Replace: " alla griglia",
			},
			{
				Find:    "vapore",
				Replace: " al vapore",
			},
		}

		for _, t := range tab {
			if strings.HasPrefix(strings.ToLower(r.Content), t.Find) {
				l := strings.Split(r.Content, " ")
				if len(l) == 2 {
					dish := strings.Title(strings.ToLower(l[1]))
					r.Content = dish + t.Replace
				}
			}
		}
	}
	return r
}

// ParseMenuCells takes a slice of strings and returns a populated menu struct.
func ParseMenuCells(nameCol []string, priceCol []string) (*Menu, error) {
	var (
		currentType MenuRowType
		menuRows    Menu
	)

	menuTitles, err := getMenuTitles(nameCol)
	if err != nil {
		return nil, fmt.Errorf("while getting menu titles: %v", err)
	}

	for idx, r := range nameCol {
		content, rowType, isTitle, isDailyProposal := parseRow(idx, normalizeSpaces(r), menuTitles)

		if isTitle {
			currentType = rowType
			continue
		}

		// Skip first empty rows/check menu date
		if currentType == Unknonwn {
			isDate, date := parseDate(normalizeSpaces(r))
			if isDate {
				menuRows.Date = date
			}
			continue
		}

		// Check if this is the end of the menu
		if currentType == Panino && rowType == Empty {
			break
		}
		if rowType == Empty {
			continue
		}

		price := parsePrice(priceCol, idx)
		// Handle "Pasta al ragù, pesto o pomodoro (sono sempre disponibili)"
		if strings.HasSuffix(content, "(sono sempre disponibili)") {

			menuRows.Add(&MenuRow{
				Content:         "Pasta al ragù",
				Type:            currentType,
				IsDailyProposal: false,
				Price: price,
			})

			menuRows.Add(&MenuRow{
				Content:         "Pasta al pesto",
				Type:            currentType,
				IsDailyProposal: false,
				Price: price,
			})

			menuRows.Add(&MenuRow{
				Content:         "Pasta al pomodoro",
				Type:            currentType,
				IsDailyProposal: false,
				Price: price,
			})

			continue
		}

		menuRows.Add(normalizeDish(&MenuRow{
			Content:         strings.TrimSpace(content),
			Type:            currentType,
			IsDailyProposal: isDailyProposal,
			Price: price,
		}))
	}

	if (menuRows.Date == time.Time{}) {
		loc, err := time.LoadLocation("Europe/Rome")
		if err != nil {
			log.Println("LoadLocation error: ", err)
			return nil, err
		}
		menuRows.Date = time.Now().In(loc)
	}

	return &menuRows, nil
}

var dailyProposalPrefixes = []string{
	"Proposta del giorno: ",
	"Prop. del giorno: ",
}

func parsePrice(priceCol []string, idx int) decimal.Decimal {
	if idx >= len(priceCol) {
		return decimal.Zero
	}
	price, err := decimal.NewFromString(strings.TrimSpace(strings.Replace(priceCol[idx], "€", "", -1)))
	if err != nil {
		return decimal.Zero
	}
	return price
}

func parseRow(idx int, content string, menuTitles map[int]MenuRowType) (string, MenuRowType, bool, bool) {
	var isDailyProposal bool

	if content == "" {
		return content, Empty, false, false
	}

	titleType, isTitle := menuTitles[idx]
	if isTitle {
		return content, titleType, isTitle, isDailyProposal
	}

	content, isDailyProposal = trimPrefixAll(content, dailyProposalPrefixes)

	return content, Unknonwn, isTitle, isDailyProposal
}

// trimPrefixAll tries to trim each prefix string in the order they are defined and returns
// the resulting strings and a bool value which is true if any of the prefixes was trimmed.
func trimPrefixAll(s string, prefixes []string) (string, bool) {
	var trimmed bool
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			s = strings.TrimPrefix(s, prefix)
			trimmed = true
		}
	}

	return s, trimmed
}

func normalizeSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// getMenuTitles returns a map of the row index for each of the sections found in the menu.
// Fuzzy matching is used to find the titles and some basic validation is done:
// - order: the titles are expected to be in the order in which the relative const enumeration is declared (see menu.go)
// - duplicates: if a duplicate title is found, an error is returned
//
// Note: it is not expected for all secitons to always be present i.e. if a section is missing, no error is thrown.
func getMenuTitles(rows []string) (map[int]MenuRowType, error) {
	var (
		menuTitlesRowIndexes = make(map[int]MenuRowType)
		lastTitleType        = Unknonwn
		currentIndex         int
	)

	for t, title := range Titles {
		results := fuzzy.Find(title, rows)
		if len(results) == 0 || results[0].Score < 0 {
			continue
		}

		if t < lastTitleType {
			return nil, errors.New(fmt.Sprintf("Unexpected title order (Found: %v after last: %v)", t, lastTitleType))
		}

		currentIndex = results[0].Index
		if _, found := menuTitlesRowIndexes[currentIndex]; found {
			return nil, errors.New(fmt.Sprintf("Unexptected title duplicate: %s", title))
		}

		// First match is always the title of a section (menu items may contain the same text)
		menuTitlesRowIndexes[results[0].Index] = t
	}

	return menuTitlesRowIndexes, nil
}
