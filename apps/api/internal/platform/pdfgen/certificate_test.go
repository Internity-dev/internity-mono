package pdfgen

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCertificate_ProducesValidPDF(t *testing.T) {
	data := CertificateData{
		CertificateNumber: "CERT-1-2026-0001",
		StudentName:       "Budi Santoso",
		NIS:               "1234567890",
		SchoolName:        "SMKN 1 Cibinong",
		CompanyName:       "PT Mumtaz Teknologi Indonesia",
		StartDate:         "2026-01-05",
		EndDate:           "2026-04-05",
		Predicate:         "A",
		AverageScore:      92.5,
		Scores: []CertificateScoreLine{
			{Name: "Kedisiplinan", Score: 90, Type: "non-teknis"},
			{Name: "Pemrograman", Score: 95, Type: "teknis"},
		},
	}

	bytesOut, err := GenerateCertificate(data)
	require.NoError(t, err)
	require.NotEmpty(t, bytesOut)
	assert.True(t, bytes.HasPrefix(bytesOut, []byte("%PDF")), "output should start with the PDF magic bytes")
	assert.Greater(t, len(bytesOut), 500, "a rendered certificate should be more than a trivially empty document")
}

func TestGenerateCertificate_NoScoreLines(t *testing.T) {
	// A certificate can still legitimately render with zero score lines
	// (e.g. an admin regenerating before scores are entered shouldn't panic).
	bytesOut, err := GenerateCertificate(CertificateData{
		CertificateNumber: "CERT-1-2026-0002",
		StudentName:       "Siti Aminah",
		NIS:               "0987654321",
		SchoolName:        "SMKN 1 Cibinong",
		CompanyName:       "PT Contoh Perusahaan",
		StartDate:         "2026-01-05",
		EndDate:           "2026-04-05",
		Predicate:         "-",
	})
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(bytesOut, []byte("%PDF")))
}
