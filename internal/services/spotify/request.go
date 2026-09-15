package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/devgianlu/go-librespot/session"
)

// getJSON performs an authenticated spclient GET and decodes the JSON body.
func getJSON(ctx context.Context, sess *session.Session, path string, out any) error {
	resp, err := sess.Spclient().Request(ctx, "GET", path, nil, nil, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("%s returned %d", path, resp.StatusCode)
	}
	return json.Unmarshal(body, out)
}
