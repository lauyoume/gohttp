package gohttp

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type MultipartStreamer struct {
	ContentType   string
	bodyBuffer    *bytes.Buffer
	bodyWriter    *multipart.Writer
	closeBuffer   *bytes.Buffer
	reader        io.Reader
	contentLength int64
}

// New initializes a new MultipartStreamer.
func NewMultiPartStreamer() (m *MultipartStreamer) {
	m = &MultipartStreamer{bodyBuffer: new(bytes.Buffer)}

	m.bodyWriter = multipart.NewWriter(m.bodyBuffer)
	boundary := m.bodyWriter.Boundary()
	m.ContentType = "multipart/form-data; boundary=" + boundary

	closeBoundary := fmt.Sprintf("\r\n--%s--\r\n", boundary)
	m.closeBuffer = bytes.NewBufferString(closeBoundary)

	return
}

// WriteFields writes multiple form fields to the multipart.Writer.
func (m *MultipartStreamer) WriteFields(fields url.Values) error {
	var err error

	for key, values := range fields {
		for _, value := range values {
			err = m.bodyWriter.WriteField(key, value)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// WriteReader adds an io.Reader to get the content of a file.  The reader is
// not accessed until the multipart.Reader is copied to some output writer.
// func (m *MultipartStreamer) WriteReader(key, filename string, size int64, reader io.Reader, ctype string) (err error) {
func (m *MultipartStreamer) WriteReader(f File) (err error) {
	m.reader = f.Reader
	m.contentLength = f.Len

	if f.ContentType == "" {
		_, err = m.bodyWriter.CreateFormFile(f.Fieldname, f.Filename)
	} else {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition",
			fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
				escapeQuotes(f.Fieldname), escapeQuotes(f.Filename)))
		h.Set("Content-Type", f.ContentType)
		m.bodyWriter.CreatePart(h)
	}
	return
}

// WriteFile is a shortcut for adding a local file as an io.Reader.
func (m *MultipartStreamer) WriteFile(key, filename string) error {
	fh, err := os.Open(filename)
	if err != nil {
		return err
	}

	stat, err := fh.Stat()
	if err != nil {
		return err
	}

	f := File{
		Fieldname: key,
		Filename:  filepath.Base(filename),
		Reader:    fh,
		Len:       stat.Size(),
	}
	return m.WriteReader(f)
}

// SetupRequest sets up the http.Request body, and some crucial HTTP headers.
func (m *MultipartStreamer) SetupRequest(req *http.Request) {
	req.Body = m.GetReader()
	req.Header.Set("Content-Type", m.ContentType)
	req.ContentLength = m.Len()
}

func (m *MultipartStreamer) Boundary() string {
	return m.bodyWriter.Boundary()
}

// Len calculates the byte size of the multipart content.
func (m *MultipartStreamer) Len() int64 {
	return m.contentLength + int64(m.bodyBuffer.Len()) + int64(m.closeBuffer.Len())
}

// GetReader gets an io.ReadCloser for passing to an http.Request.
func (m *MultipartStreamer) GetReader() io.ReadCloser {
	if m.reader == nil {
		reader := io.MultiReader(m.bodyBuffer, m.closeBuffer)
		return ioutil.NopCloser(reader)
	}
	reader := io.MultiReader(m.bodyBuffer, m.reader, m.closeBuffer)
	return ioutil.NopCloser(reader)
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}
