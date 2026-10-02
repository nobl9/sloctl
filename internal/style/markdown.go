package style

import (
	"fmt"
	"image/color"

	"charm.land/glamour/v2/ansi"
	"charm.land/glamour/v2/styles"
	"github.com/alecthomas/chroma/v2"
	chromastyles "github.com/alecthomas/chroma/v2/styles"
)

var (
	markdownSyntaxLight = newMarkdownSyntax("sloctl-help-light", themePalette(false))
	markdownSyntaxDark  = newMarkdownSyntax("sloctl-help-dark", themePalette(true))
)

func newMarkdownSyntax(name string, colors palette) *chroma.Style {
	accent, muted := hexColor(colors.accent), hexColor(colors.muted)
	return chromastyles.Register(chroma.MustNewStyle(name, chroma.StyleEntries{
		chroma.Comment:             muted + " italic",
		chroma.Keyword:             accent + " bold",
		chroma.Operator:            accent,
		chroma.Punctuation:         accent,
		chroma.NameVariable:        accent + " bold",
		chroma.NameAttribute:       accent,
		chroma.NameTag:             accent + " bold",
		chroma.LiteralNumber:       accent,
		chroma.LiteralString:       accent,
		chroma.LiteralStringEscape: accent + " bold",
		chroma.GenericDeleted:      hexColor(red),
		chroma.GenericInserted:     accent,
		chroma.GenericEmph:         "italic",
		chroma.GenericStrong:       "bold",
	}))
}

func hexColor(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}

// MarkdownTheme shares the interactive forms' accents and muted text colors.
func MarkdownTheme(isDark bool) ansi.StyleConfig {
	colors := themePalette(isDark)
	accent := hexColor(colors.accent)
	t := styles.ASCIIStyleConfig
	t.Document.Margin = new(uint(0))
	t.Heading.Color = new(accent)
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
	t.Code = ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Bold: new(true)}}
	// Preserve leading whitespace so copied heredoc delimiters remain valid.
	t.CodeBlock.Margin = new(uint(0))
	t.CodeBlock.Theme = markdownSyntaxLight.Name
	if isDark {
		t.CodeBlock.Theme = markdownSyntaxDark.Name
	}
	t.Link.Color = new(accent)
	t.Link.Underline = new(true)
	t.LinkText.Bold = new(true)
	return t
}
