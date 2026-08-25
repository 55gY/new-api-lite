package common

import (
	"io"

	"github.com/55gY/new-api-lite/common"
)

// NewOutboundJSONBody wraps the already-marshaled upstream request body into a
// BodyStorage. When disk cache is enabled and the payload exceeds the configured
// threshold, the data is written to a temp file and the original []byte can be
// GC'd, significantly reducing the heap residency while waiting for the
// upstream provider to respond (the dominant cost for large base64 payloads).
//
// In memory mode the underlying memoryStorage reuses the same backing array,
// so this is equivalent to bytes.NewReader(data) in terms of memory usage.
//
// The caller MUST invoke closer.Close() once the upstream call has finished
// (typically via defer) to release the disk file / memory accounting.
//
// The returned primary reader and getBody factory each own only a child reader;
// callers must still close the returned root closer to release storage accounting
// and any temporary disk file. The size is propagated to http.Request.ContentLength
// because the type-erased reader prevents net/http from detecting it automatically.
func NewOutboundJSONBody(data []byte) (body io.Reader, size int64, getBody func() (io.ReadCloser, error), closer io.Closer, err error) {
	storage, err := common.CreateBodyStorage(data)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	reader, err := storage.NewReader()
	if err != nil {
		_ = storage.Close()
		return nil, 0, nil, nil, err
	}
	return reader, storage.Size(), storage.NewReader, storage, nil
}
