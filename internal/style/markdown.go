package style

import (
	"charm.land/glamour/v2/ansi"
	"charm.land/glamour/v2/styles"
	"github.com/alecthomas/chroma/v2"
	chromastyles "github.com/alecthomas/chroma/v2/styles"
)

// The terminal16 formatter maps these names to the terminal's ANSI palette.
var markdownSyntax = chromastyles.Register(chroma.MustNewStyle("sloctl-help", chroma.StyleEntries{
	chroma.Text:                "#ansidarkblue",
	chroma.Comment:             "italic",
	chroma.Keyword:             "#ansipurple",
	chroma.Operator:            "#ansipurple",
	chroma.Punctuation:         "#ansipurple",
	chroma.Name:                "#ansidarkblue",
	chroma.NameVariable:        "#ansiteal",
	chroma.NameAttribute:       "#ansiteal",
	chroma.NameTag:             "#ansiteal",
	chroma.LiteralNumber:       "#ansipurple",
	chroma.LiteralString:       "#ansidarkgreen",
	chroma.LiteralStringEscape: "#ansipurple",
	chroma.GenericDeleted:      "#ansidarkred",
	chroma.GenericInserted:     "#ansidarkgreen",
	chroma.GenericEmph:         "italic",
	chroma.GenericStrong:       "bold",
}))

// MarkdownTheme uses ANSI colors so help follows the terminal's own palette.
func MarkdownTheme() ansi.StyleConfig {
	t := styles.ASCIIStyleConfig
	t.Document.Margin = new(uint(0))
	t.Heading.Color = new("6")
	t.Heading.Bold = new(true)
	t.H1.Prefix = ""
	t.H2.Prefix = ""
	t.H3.Prefix = ""
	t.H4.Prefix = ""
	t.H5.Prefix = ""
	t.H6.Prefix = ""
	t.Strong = ansi.StylePrimitive{Bold: new(true)}
	t.Emph = ansi.StylePrimitive{Italic: new(true)}
	t.Strikethrough = ansi.StylePrimitive{CrossedOut: new(true)}
	t.Code = ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: new("6")}}
	t.CodeBlock.Theme = markdownSyntax.Name
	t.Link.Color = new("6")
	t.Link.Underline = new(true)
	t.LinkText.Bold = new(true)
	return t
}
