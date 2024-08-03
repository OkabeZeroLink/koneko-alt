package tui

import (
	"fmt"
	"log"
	"strings"

	"github.com/gdamore/tcell/v2"
	h "github.com/OkabeZeroLink/koneko-alt/helpers"
	t "github.com/OkabeZeroLink/koneko-alt/types"
	"github.com/rivo/tview"
)

var app = tview.NewApplication()

var nyaaDownload = "https://nyaa.land/download/"
var sukebeiDownload = "https://fap.iss.one/download/"

var query = ""
var category = 0
var filter = 0
var sortBy = 0
var markedTorrents []t.MarkedTorrent

var nyaaCategories = []string{
	"All",
	"Anime",
	"- Anime Music Video",
	"- English-translated",
	"- Non-English-translated",
	"- Raw",
	"Audio",
	"- Lossless",
	"- Lossy",
	"Literature",
	"- English-translated",
	"- Non-English-translated",
	"- Raw",
	"Live Action",
	"- English-translated",
	"- Idol/Promotional Video",
	"- Non-English-translated",
	"- Raw",
	"Pictures",
	"- Graphics",
	"- Photos",
	"Software",
	"- Applications",
	"- Games",
}

var sukebeiCategories = []string{
	"All",
	"Art",
	"- Anime",
	"- Doujinshi",
	"- Games",
	"- Manga",
	"- Pictures",
	"Real Life",
	"- Photobooks and Pictures",
	"- Videos",
}

var filters = []string{
	"No filter",
	"No remakes",
	"Trusted only",
}

var sortOptions = []string{
	"Date",
	"Downloads",
	"Size",
	"Seeders",
	"Leechers",
	"Comments",
}

var HelpText = ` Keybindings
 Press ESC to switch back
--------------------------------------------------------------------
| panel            | operation                | key                |
|------------------|--------------------------|--------------------|
| search           | navigate                 | Tab / Shift + Tab  |
| search           | focus results            | Esc                |
| results          | mark torrent             | Enter              |
| results          | download marked torrents | Ctrl + D           |
| results          | open marked torrents     | Ctrl + O           |
| results          | get marked torrent info  | Ctrl + I           |
| results          | move down                | j / ↓              |
| results          | move up                  | k / ↑              |
| results          | move to the top          | g / home           |
| results          | move to the bottom       | G / end            |
| results          | focus search             | Esc / Tab          |
| all              | open help                | Ctrl + J           |
| all              | exit                     | Ctrl + C           |
--------------------------------------------------------------------`

func Launch(provider string) {
	pages := tview.NewPages()
	app.SetRoot(pages, true).EnableMouse(true)

	info := setupInfoPage(pages)
	search := setupMainPage(pages, provider, info)
	help := setupHelpPage(pages)

	pages.AddPage("main", search, true, true)
	pages.AddPage("help", help, true, true)
	pages.AddPage("info", info, true, true)

	pages.SwitchToPage("main")
	app.SetFocus(search)
	app.Run()
}

func setupMainPage(p *tview.Pages, provider string, info *tview.TextView) *tview.Flex {
	table := tview.NewTable().
		SetSelectable(true, false)
	table.SetBorder(true)

	if provider == "nyaa" {
		table.SetTitle("nyaa.si")
	} else if provider == "sukebei" {
		table.SetTitle("sukebei.nyaa.si")
	}

	form := tview.NewForm().
		AddInputField("Query", "", 24, nil, func(text string) {
			query = text
		})
	if provider == "nyaa" {
		form.AddDropDown("Category", nyaaCategories, 0, func(option string, optionIndex int) {
			category = optionIndex
		})
	}
	if provider == "sukebei" {
		form.AddDropDown("Category", sukebeiCategories, 0, func(option string, optionIndex int) {
			category = optionIndex
		})
	}
	form.AddDropDown("Filter", filters, 0, func(option string, optionIndex int) {
		filter = optionIndex
	}).
		AddDropDown("Sort By", sortOptions, 0, func(option string, optionIndex int) {
			sortBy = optionIndex
		}).
		AddButton("Search", func() {
			UnmarkAll(table)
			c := ""
			torrents := ""
			s := h.ConvertSort(sortBy)
			f := h.ConvertFilter(filter)

			if provider == "nyaa" {
				c = h.ConvertNyaaCategory(category)
			} else if provider == "sukebei" {
				c = h.ConvertSukebeiCategory(category)
			}

			if provider == "nyaa" {
				torrents = fetchTorrents("nyaa", query, c, s, f)
			}
			if provider == "sukebei" {
				torrents = fetchTorrents("sukebei", query, c, s, f)
			}

			table.Clear()
			setTableData(table, torrents[:len(torrents)-1]) // remove last \n
			app.SetFocus(table)
			table.ScrollToBeginning()
		})

	form.SetBorder(true)
	form.SetHorizontal(true)

	form.SetCancelFunc(func() {
		app.SetFocus(table)
	})

	table.SetDoneFunc(func(key tcell.Key) {
		app.SetFocus(form)
	})

	table.SetSelectedFunc(func(row int, column int) {
		torrent := table.GetCell(row, 6)
		link := table.GetCell(row, 7)
		curColor := torrent.Color

		MarkTorrent(torrent, link, curColor, row)
		UnmarkTorrent(torrent, link, curColor)
	})

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlD {
			_, err := downloadTorrents(markedTorrents, provider)
			if err != nil {
				log.Fatal(err)
			}
			UnmarkAll(table)
		}
		if event.Key() == tcell.KeyCtrlO {
			err := openTorrents(provider)
			if err != nil {
				log.Fatal(err)
			}
			UnmarkAll(table)
		}
		return event
	})

	textView := tview.NewTextView().
		SetWordWrap(true).
		SetDynamicColors(true)

	fmt.Fprintf(textView, "%s ", `[#3fff33]Help: Ctrl + J[white]`)
	textView.SetBorder(true)

	flex := tview.NewFlex().
		AddItem(form, 5, 1, true).SetDirection(tview.FlexRow).
		AddItem(table, 0, 6, true).SetDirection(tview.FlexRow).
		AddItem(textView, 3, 1, true).SetDirection(tview.FlexRow)

	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlJ {
			p.SwitchToPage("help")
		}
		if event.Key() == tcell.KeyCtrlI && len(markedTorrents) == 1 {
			i := fetchTorrentInfo(markedTorrents[0].LinkCell.Text)
			info.SetText(i)
			p.SwitchToPage("info")
		}
		return event
	})

	return flex
}

func setupHelpPage(p *tview.Pages) *tview.TextView {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	go func() {
		for _, word := range strings.Split(HelpText, " ") {
			if word == "panel" {
				word = "[#2e64fe]panel[white]"
			}
			if word == "operation" {
				word = "[#2efe2e]operation[white]"
			}
			if word == "key" {
				word = "[#ffff00]key[white]"
			}
			fmt.Fprintf(textView, "%s ", word)
		}
	}()

	textView.SetBorder(true)

	textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyESC {
			p.SwitchToPage("main")
		}
		return event
	})

	return textView
}

func setupInfoPage(p *tview.Pages) *tview.TextView {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	textView.SetBorder(true)

	textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyESC {
			p.SwitchToPage("main")
		}
		return event
	})

	return textView
}

func setTableData(table *tview.Table, data string) {
	for row, line := range strings.Split(data, "\n") {
		for column, cell := range strings.Split(line, "{}") {
			textColor := tcell.ColorWhite
			bgColor := tcell.ColorBlack
			align := tview.AlignLeft

			if column == 0 {
				textColor = tcell.ColorYellow
			} else if column == 1 {
				textColor = tcell.ColorGreen
			} else if column == 2 {
				textColor = tcell.ColorRed
			} else if column == 3 {
				textColor = tcell.ColorMediumPurple
			} else if column == 4 {
				textColor = tcell.ColorLightPink
			} else if column == 5 {
				textColor = tcell.ColorLightCyan
			}

			if strings.Contains(line, "(trusted torrent)") && column == 6 {
				textColor = tcell.ColorGreen
			}

			tableCell := tview.NewTableCell(cell).
				SetTextColor(textColor).
				SetBackgroundColor(bgColor).
				SetAlign(align).
				SetSelectable(true)
			table.SetCell(row, column, tableCell)
		}
	}
}
