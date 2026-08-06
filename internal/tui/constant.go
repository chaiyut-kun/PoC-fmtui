package tui

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

func makeFg(c string) func(...any) string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(c))
	return func(a ...any) string { return style.Render(fmt.Sprint(a...)) }
}

func makeBg(c string) func(...any) string {
	style := lipgloss.NewStyle().Background(lipgloss.Color(c))
	return func(a ...any) string { return style.Render(fmt.Sprint(a...)) }
}

var (
	Plain   = func(a ...any) string { return fmt.Sprint(a...) }
	Red     = makeFg("9")
	Blue    = makeFg("12")
	Cyan    = makeFg("14")
	Magenta = makeFg("13")
	Yellow  = makeFg("11")
	White   = makeFg("15")
	Green   = makeFg("10")
	Gray    = makeFg("241")
	Black   = makeFg("0")
	Orange  = makeFg("208")

	BgRed     = makeBg("9")
	BgBlue    = makeBg("12")
	BgCyan    = makeBg("14")
	BgYellow  = makeBg("11")
	BgGreen   = makeBg("10")
	BgWhite   = makeBg("15")
	BgMagenta = makeBg("13")

	lightGray = lipgloss.Color("241")
	pink      = "205"
)

const ItemEndSpace = " "

const (
	CursorIcon = "▶"
	SearchIcon = "\ue68f"
	Marker     = "*"

	DirIcon     = "\uf4d3"
	FileIcon    = "\uf15b"
	GoExt       = "\ue65e"
	GoIcon      = "\ue627"
	JsIcon      = "\uf2ef"
	PyIcon      = "\ue235"
	GoMasCot    = "\ue627"
	MakeIcon    = "\ue673"
	BashIcon    = "\uebca"
	MdIcon      = "\ueeab"
	PdfIcon     = "\uf1c1"
	GitIcon     = "\ue709"
	JsonIcon    = "\ue60b"
	SymlinkIcon = "\uf44c"

	GitIgnoreIcon  = "\ue65d"
	TsIcon         = "\ue69d"
	PtIcon         = "\ue87b"
	JavaIcon       = "\uedaf"
	CssIcon        = "\ue749"
	HtmlIcon       = "\ue736"
	VueIcon        = "\ue6a0"
	SqliteIcon     = "\ue7c4"
	DockerFileIcon = "\uf21f"

	// Office
	TextIcon = "\ue64e"
	XlsIcon  = "\ue6a6"
	DocsIcon = "\ue6a5"
	CsvIcon  = "\ueefc"
	PptxIcon = "\uf1c4"

	// special
	CompressedIcon = "\uf410"

	// image / vdo
	ImageIcon = "\uf03e"
	VdoIcon   = "\uf52c"
	AudioIcon = "\uf1c7"

	// windows
	PsIcon = "\ue86c"
)

// dialog config
const (
	quickSearchWidthDefault     = 80
	quickSearchHalfWidth        = 40
	quickSearchMaxResultDefault = 10

	DefaultYDivisor = 2

	DefaultScreenWidth = 80
	inputWidth         = 32
)
