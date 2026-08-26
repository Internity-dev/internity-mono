// Package pdfgen renders documents with a pure-Go library (maroto) — no
// external binary dependency (unlike legacy's pdftk+dompdf combo), so it
// containerizes cleanly and needs nothing installed on the host.
package pdfgen

import (
	"fmt"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

type CertificateScoreLine struct {
	Name  string
	Score int
	Type  string
}

type CertificateData struct {
	CertificateNumber string
	StudentName       string
	NIS               string
	SchoolName        string
	CompanyName       string
	StartDate         string
	EndDate           string
	Predicate         string
	AverageScore      float64
	Scores            []CertificateScoreLine
}

// GenerateCertificate renders a one-page PDF: title, recipient, placement
// details, and a technical/non-technical score breakdown with the derived
// letter-grade predicate. Deterministic given the same data — that's what
// makes re-downloading idempotent (scoring.Service never regenerates the
// certificate_number, only re-renders from the current score snapshot).
func GenerateCertificate(data CertificateData) ([]byte, error) {
	cfg := config.NewBuilder().Build()
	m := maroto.New(cfg)

	center := func(size float64, style fontstyle.Type, value string) core.Row {
		return text.NewRow(size, value, props.Text{Size: size, Style: style, Align: align.Center})
	}

	m.AddRows(center(10, fontstyle.Normal, "CERTIFICATE OF INTERNSHIP COMPLETION"))
	m.AddRows(row.New(4))
	m.AddRows(center(6, fontstyle.Normal, data.CertificateNumber))
	m.AddRows(row.New(8))
	m.AddRows(center(6, fontstyle.Normal, "This certifies that"))
	m.AddRows(row.New(2))
	m.AddRows(center(9, fontstyle.Bold, data.StudentName))
	m.AddRows(row.New(2))
	m.AddRows(center(6, fontstyle.Normal, fmt.Sprintf("NIS %s, %s", data.NIS, data.SchoolName)))
	m.AddRows(row.New(6))
	m.AddRows(center(6, fontstyle.Normal, fmt.Sprintf("has completed an internship (PKL) at %s", data.CompanyName)))
	m.AddRows(center(6, fontstyle.Normal, fmt.Sprintf("from %s to %s", data.StartDate, data.EndDate)))
	m.AddRows(row.New(8))
	m.AddRows(center(7, fontstyle.Bold, fmt.Sprintf("Overall predicate: %s (%.1f)", data.Predicate, data.AverageScore)))
	m.AddRows(row.New(6))

	m.AddRows(row.New(6).Add(
		text.NewCol(6, "Category", props.Text{Size: 8, Style: fontstyle.Bold}),
		text.NewCol(3, "Type", props.Text{Size: 8, Style: fontstyle.Bold}),
		text.NewCol(3, "Score", props.Text{Size: 8, Style: fontstyle.Bold, Align: align.Right}),
	))
	for _, line := range data.Scores {
		m.AddRows(row.New(5).Add(
			text.NewCol(6, line.Name, props.Text{Size: 8}),
			text.NewCol(3, line.Type, props.Text{Size: 8}),
			text.NewCol(3, fmt.Sprintf("%d", line.Score), props.Text{Size: 8, Align: align.Right}),
		))
	}

	doc, err := m.Generate()
	if err != nil {
		return nil, err
	}
	return doc.GetBytes(), nil
}
