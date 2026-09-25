package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/devgianlu/go-librespot/session"
)

// getJSON performs an authenticated spclient GET and decodes the JSON body.
func getJSON(ctx context.Context, sess *session.Session, path string, out any) error {
	resp, err := sess.Spclient().Request(ctx, "GET", path, nil, nil, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// spclient answers with a JSON "error" body that says why, and without
		// it a rejected request is unreadable.
		reason, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return fmt.Errorf("%s returned %d: %s", path, resp.StatusCode, strings.TrimSpace(string(reason)))
	}
	// Stream the body into the decoder instead of buffering the whole payload
	// alongside the decoded struct.
	return json.NewDecoder(resp.Body).Decode(out)
}
