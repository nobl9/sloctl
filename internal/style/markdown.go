package style

import (
	"charm.land/glamour/v2/ansi"
	"charm.land/glamour/v2/styles"
)

// MarkdownTheme uses the terminal's text and background colors without querying it.
func MarkdownTheme() ansi.StyleConfig {
	t := styles.ASCIIStyleConfig
	t.Document.Margin = new(uint(0))
	t.Heading.Color = new(tealHex)
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
	t.Code = ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: new(pinkHex)}}
	t.CodeBlock.Chroma = &ansi.Chroma{
		Text:          ansi.StylePrimitive{},
		Comment:       ansi.StylePrimitive{Italic: new(true)},
		Keyword:       ansi.StylePrimitive{Color: new(tealHex)},
		Name:          ansi.StylePrimitive{},
		LiteralString: ansi.StylePrimitive{Color: new(pinkHex)},
	}
	t.Link.Underline = new(true)
	t.LinkText.Bold = new(true)
	return t
}
