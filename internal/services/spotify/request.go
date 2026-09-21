package spotify

import (
	"context"
	"encoding/json"
	"fmt"

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
		return fmt.Errorf("%s returned %d", path, resp.StatusCode)
	}
	// Stream the body into the decoder instead of buffering the whole payload
	// alongside the decoded struct.
	return json.NewDecoder(resp.Body).Decode(out)
}
