// Helper functions for tests
package main
import (
	"fmt"
	"bytes"
	"io"
)

func mock_get_page_io(url string, access bool) io.Reader {
	return bytes.NewBufferString(fmt.Sprintf("{\"access\": %t, \"status\": \"yay\", \"message\": \"Some Message\"}", access))
}

func mock_get_page(url string, access bool) []byte {
	return bytes.NewBufferString(fmt.Sprintf("{\"access\": %t, \"status\": \"yay\", \"message\": \"Some Message\"}", access)).Bytes()
}
