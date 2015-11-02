package html2text

import (
	"bytes"
	"io"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var (
	spacingRe = regexp.MustCompile(`[ \r\n\t]+`)
	newlineRe = regexp.MustCompile(`\n\n+`)
)

type textifyTraverseCtx struct {
	Buf bytes.Buffer

	endsWithSpace bool
}

func (ctx *textifyTraverseCtx) Traverse(node *html.Node) error {
	switch node.Type {

	default:
		return ctx.TraverseChildren(node)

	case html.TextNode:
		data := strings.Trim(spacingRe.ReplaceAllString(node.Data, " "), " ")
		return ctx.Emit(data)

	case html.ElementNode:

		switch node.DataAtom {
		case atom.Br:
			return ctx.Emit("\n")

		case atom.Li:
			if err := ctx.Emit("* "); err != nil {
				return err
			}

			if err := ctx.TraverseChildren(node); err != nil {
				return err
			}

			return ctx.Emit("\n")

		case atom.A:
			if err := ctx.TraverseChildren(node); err != nil {
				return err
			}

			hrefLink := ""
			for _, attr := range node.Attr {
				if attr.Key != "href" {
					continue
				}

				attrVal := ctx.NormalizeHrefLink(attr.Val)
				if attrVal != "" {
					hrefLink = "( " + attrVal + " )"
				}
			}

			return ctx.Emit(hrefLink)

		case atom.P, atom.Ul:
			if err := ctx.Emit("\n\n"); err != nil {
				return err
			}

			if err := ctx.TraverseChildren(node); err != nil {
				return err
			}

			return ctx.Emit("\n\n")

		default:
			return ctx.TraverseChildren(node)
		}
	}
}

func (ctx *textifyTraverseCtx) TraverseChildren(node *html.Node) error {
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if err := ctx.Traverse(c); err != nil {
			return err
		}
	}

	return nil
}

func (ctx *textifyTraverseCtx) Emit(data string) error {
	if len(data) == 0 {
		return nil
	}

	runes := []rune(data)
	startsWithSpace := unicode.IsSpace(runes[0])
	if !startsWithSpace && !ctx.endsWithSpace {
		ctx.Buf.WriteByte(' ')
	}
	ctx.endsWithSpace = unicode.IsSpace(runes[len(runes)-1])

	_, err := ctx.Buf.WriteString(data)
	return err
}

func (ctx *textifyTraverseCtx) NormalizeHrefLink(link string) string {
	link = strings.TrimSpace(link)
	link = strings.TrimPrefix(link, "mailto:")
	return link
}

func FromReader(reader io.Reader) (string, error) {
	doc, err := html.Parse(reader)
	if err != nil {
		return "", err
	}

	ctx := textifyTraverseCtx{
		Buf: bytes.Buffer{},
	}
	if err = ctx.Traverse(doc); err != nil {
		return "", err
	}

	text := strings.TrimSpace(newlineRe.ReplaceAllString(
		strings.Replace(ctx.Buf.String(), "\n ", "\n", -1), "\n\n"))
	return text, nil
}

func FromString(input string) (string, error) {
	text, err := FromReader(strings.NewReader(input))
	if err != nil {
		return "", err
	}
	return text, nil
}
