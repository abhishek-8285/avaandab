package rag

import (
	"fmt"
	"strings"
)

// extractPDFText extracts text from a PDF file.
// Uses a simple approach: scans for PDF text streams.
// For production use, consider a dedicated PDF library.
func extractPDFText(data []byte) (string, error) {
	content := string(data)

	// Simple PDF text extraction: look for text between BT/ET markers
	// This handles basic PDFs but not encrypted or complex ones
	var builder strings.Builder

	// Split by BT (begin text) markers
	parts := strings.Split(content, "BT")
	for i := 1; i < len(parts); i++ {
		// Find the corresponding ET (end text)
		etIdx := strings.Index(parts[i], "ET")
		if etIdx == -1 {
			etIdx = len(parts[i])
		}
		textBlock := parts[i][:etIdx]

		// Extract text between parentheses (standard PDF text format)
		for {
			openParen := strings.Index(textBlock, "(")
			if openParen == -1 {
				break
			}
			closeParen := strings.Index(textBlock[openParen+1:], ")")
			if closeParen == -1 {
				break
			}
			text := textBlock[openParen+1 : openParen+1+closeParen]
			// Clean up PDF text encoding artifacts
			text = strings.ReplaceAll(text, "\\n", "\n")
			text = strings.ReplaceAll(text, "\\r", "")
			text = strings.ReplaceAll(text, "\\t", " ")
			builder.WriteString(text)
			builder.WriteString(" ")
			textBlock = textBlock[openParen+1+closeParen+1:]
		}
	}

	result := strings.TrimSpace(builder.String())
	if result == "" {
		return "", fmt.Errorf("no text found in PDF (may be scanned/image-based)")
	}

	return result, nil
}
