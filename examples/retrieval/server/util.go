package main

import (
	"io"
	"os"
)

type fileSectionReadCloser struct {
	file          *os.File
	sectionReader *io.SectionReader
}

func (f *fileSectionReadCloser) Read(p []byte) (int, error) {
	n, err := f.sectionReader.Read(p)
	if err == io.EOF {
		return n, f.Close()
	}
	return n, err
}

func (f *fileSectionReadCloser) Close() error {
	return f.file.Close()
}

func newFileSectionReader(file *os.File, off int, n int) *fileSectionReadCloser {
	return &fileSectionReadCloser{file, io.NewSectionReader(file, int64(off), int64(n))}
}
