package monitor

import (
	"bufio"
	"io"
	"os"
	"strings"
	"time"
)

type TailableFile struct {
	file               *os.File
	reader             *bufio.Reader
	output             chan string
	pollIntervalMillis int64
}

func TailFile(filename string, output chan string, interval int64) (*TailableFile, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	// Seek to the end so we start from the freshest data.
	// It is possible to wind up in the middle of a record in a race condition.
	// This is okay (we'll drop it as malformed and start reading from the next full record).
	_, err = file.Seek(0, 2)
	if err != nil {
		return nil, err
	}
	return &TailableFile{
		file:               file,
		reader:             bufio.NewReader(file),
		output:             output,
		pollIntervalMillis: interval,
	}, nil
}

func (f *TailableFile) ContinuousRead() error {
	for {
		line, err := f.readLine()
		if err != nil {
			if err == io.EOF {
				// sleep and then retry
				stat, err := f.file.Stat()
				if err != nil {
					return err
				}
				mtime := stat.ModTime()
				for {
					// TODO(lizf): handle file removal by stating the filename rather than the existing handle.
					stat, err := f.file.Stat()
					if err != nil {
						return err
					}
					if mtime != stat.ModTime() {
						break
					}
					time.Sleep(time.Duration(f.pollIntervalMillis) * time.Millisecond)
				}
				// There is new work to do. TODO(lizf): handle file rotation/truncation.
				continue
			}
			// non-EOF errors are terminal.
			return err
		}
		f.output <- line
	}
}

func (f *TailableFile) readLine() (string, error) {
	line, err := f.reader.ReadString('\n')
	if err != nil {
		// Process the partial read anyways. The caller should read err to determine whether to proceed.
		return line, err
	}

	return strings.TrimSuffix(line, "\n"), nil
}

func (f *TailableFile) Close() {
	if f.file == nil {
		// noop on double Close()
		return
	}
	f.file.Close()
	f.file = nil
	close(f.output)
}
