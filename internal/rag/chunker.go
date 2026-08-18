package rag

import (
	"os"
	"path/filepath"
	"strings"
)

type Chunk struct {
	ID       string `json:"id"`
	Content  string `json:"content"`
	Source   string `json:"source"`
	LineFrom int    `json:"line_from"`
	LineTo   int    `json:"line_to"`
	ChunkIdx int    `json:"chunk_idx"`
}

type Chunker struct {
	ChunkSize    int
	ChunkOverlap int
}

func NewChunker(chunkSize, chunkOverlap int) *Chunker {
	if chunkSize <= 0 {
		chunkSize = 512
	}
	if chunkOverlap < 0 {
		chunkOverlap = 0
	}
	if chunkOverlap >= chunkSize {
		chunkOverlap = chunkSize / 2
	}
	return &Chunker{
		ChunkSize:    chunkSize,
		ChunkOverlap: chunkOverlap,
	}
}

func (c *Chunker) ChunkFile(filePath string) ([]Chunk, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	content := string(data)
	lines := strings.Split(content, "\n")
	source := strings.TrimPrefix(filePath, "./")

	var chunks []Chunk
	var currentChunk strings.Builder
	var chunkIdx int
	lineFrom := 1

	// Count characters as we go
	charCount := 0
	currentLine := 1

	for i, line := range lines {
		if charCount > 0 && currentChunk.Len() > 0 {
			currentChunk.WriteString("\n")
			charCount++
		}

		lineLen := len(line) + 1 // +1 for newline
		if charCount > 0 && charCount+len(line)+1 > c.ChunkSize && currentChunk.Len() > 0 {
			// Save current chunk
			chunks = append(chunks, Chunk{
				ID:       chunkID(source, chunkIdx),
				Content:  strings.TrimSpace(currentChunk.String()),
				Source:   source,
				LineFrom: lineFrom,
				LineTo:   currentLine - 1,
				ChunkIdx: chunkIdx,
			})
			chunkIdx++

			// Start new chunk with overlap
			overlapLines := c.extractOverlapLines(lines, i-1, c.ChunkOverlap)
			currentChunk.Reset()
			for _, ol := range overlapLines {
				if currentChunk.Len() > 0 {
					currentChunk.WriteString("\n")
				}
				currentChunk.WriteString(ol)
			}
			if len(overlapLines) > 0 {
				lineFrom = currentLine - len(overlapLines) + 1
			}
			charCount = currentChunk.Len()
		}

		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n")
			charCount++
		}
		currentChunk.WriteString(line)
		charCount += lineLen
		currentLine = i + 2 // 1-indexed, +1 because we're at end of this line

		if i == len(lines)-1 && currentChunk.Len() > 0 {
			chunks = append(chunks, Chunk{
				ID:       chunkID(source, chunkIdx),
				Content:  strings.TrimSpace(currentChunk.String()),
				Source:   source,
				LineFrom: lineFrom,
				LineTo:   currentLine - 1,
				ChunkIdx: chunkIdx,
			})
		}
	}

	return chunks, nil
}

func (c *Chunker) extractOverlapLines(lines []string, endIdx, maxOverlapChars int) []string {
	var result []string
	charCount := 0
	for i := endIdx; i >= 0 && charCount < maxOverlapChars; i-- {
		line := lines[i]
		result = append([]string{line}, result...)
		charCount += len(line) + 1
	}
	return result
}

func (c *Chunker) ChunkText(content, source string) []Chunk {
	lines := strings.Split(content, "\n")
	var chunks []Chunk
	var currentChunk strings.Builder
	var chunkIdx int
	lineFrom := 1
	currentLine := 1
	charCount := 0

	for i, line := range lines {
		lineLen := len(line) + 1
		if charCount > 0 && currentChunk.Len() > 0 && charCount+len(line)+1 > c.ChunkSize {
			chunks = append(chunks, Chunk{
				ID:       chunkID(source, chunkIdx),
				Content:  strings.TrimSpace(currentChunk.String()),
				Source:   source,
				LineFrom: lineFrom,
				LineTo:   currentLine - 1,
				ChunkIdx: chunkIdx,
			})
			chunkIdx++
			overlapLines := c.extractOverlapLines(lines, i-1, c.ChunkOverlap)
			currentChunk.Reset()
			for _, ol := range overlapLines {
				if currentChunk.Len() > 0 {
					currentChunk.WriteString("\n")
				}
				currentChunk.WriteString(ol)
			}
			if len(overlapLines) > 0 {
				lineFrom = currentLine - len(overlapLines) + 1
			}
			charCount = currentChunk.Len()
		}

		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n")
			charCount++
		}
		currentChunk.WriteString(line)
		charCount += lineLen
		currentLine = i + 2

		if i == len(lines)-1 && currentChunk.Len() > 0 {
			chunks = append(chunks, Chunk{
				ID:       chunkID(source, chunkIdx),
				Content:  strings.TrimSpace(currentChunk.String()),
				Source:   source,
				LineFrom: lineFrom,
				LineTo:   currentLine - 1,
				ChunkIdx: chunkIdx,
			})
		}
	}

	return chunks
}

func (c *Chunker) IndexDirectory(rootDir string, extensions []string) ([]Chunk, error) {
	var allChunks []Chunk
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			base := strings.ToLower(info.Name())
			if base == "node_modules" || base == ".git" || base == "vendor" || base == "bin" || base == "dist" || base == "test-results" || base == "tmp" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.TrimPrefix(filepath.Ext(path), ".")
		for _, allowed := range extensions {
			if ext == allowed {
				chunks, err := c.ChunkFile(path)
				if err != nil {
					return err
				}
				allChunks = append(allChunks, chunks...)
				break
			}
		}
		return nil
	})
	return allChunks, err
}

func chunkID(source string, idx int) string {
	return source + "#" + itoa(idx)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var buf [20]byte
	j := len(buf)
	for i > 0 {
		j--
		buf[j] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		j--
		buf[j] = '-'
	}
	return string(buf[j:])
}
