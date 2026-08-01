package tui

import (
	"fmt"

	"charm.land/lipgloss/v2"
	"github.com/fatih/color"
)

var (
	Plain   = func(a ...any) string { return fmt.Sprint(a...) }
	Red     = color.New(color.FgRed).SprintFunc()
	Blue    = color.New(color.FgBlue).SprintFunc()
	Cyan    = color.New(color.FgCyan).SprintFunc()
	Magenta = color.New(color.FgMagenta).SprintFunc()
	Yellow  = color.New(color.FgYellow).SprintFunc()
	White   = color.New(color.FgWhite).SprintFunc()
	Green   = color.New(color.FgGreen).SprintFunc()
	Gray    = color.New(color.FgWhite).SprintFunc() // FgWhite is standard gray on standard terminals
	Black   = color.New(color.FgBlack).SprintFunc()

	BgRed     = color.New(color.BgRed).SprintFunc()
	BgBlue    = color.New(color.BgBlue).SprintFunc()
	BgCyan    = color.New(color.BgCyan).SprintFunc()
	BgYellow  = color.New(color.BgYellow).SprintFunc()
	BgGreen   = color.New(color.BgGreen).SprintFunc()
	BgWhite   = color.New(color.BgWhite).SprintFunc()
	BgMagenta = color.New(color.BgMagenta).SprintFunc()

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
	quickSearchMaxResultDefault = 20

	DefaultYDivisor = 2

	DefaultScreenWidth = 80
	inputWidth         = 32
)
