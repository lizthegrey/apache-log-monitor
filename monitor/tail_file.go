package monitor

import ()

type TailableFile struct {
}

func TailFile(filename string) *TailableFile {
	return nil
}

func (f *TailableFile) ContinuousRead(output chan string) {
	output <- "foo"
	output <- "127.0.0.1 - frank [10/Oct/2000:13:55:36 -0700] \"GET /apache_pb.gif HTTP/1.0\" 200 2326"
	close(output)
}
