package p2p

import (
	"testing"

	"github.com/pion/webrtc/v4"
)

func TestSignalRoundTrip(t *testing.T) {
	sd := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  "v=0\r\ns=-\r\n",
	}
	code := encode(&sd)
	decoded, err := decode(code)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if decoded.Type != sd.Type {
		t.Fatalf("type mismatch: got %v, want %v", decoded.Type, sd.Type)
	}
	if decoded.SDP != sd.SDP {
		t.Fatalf("sdp mismatch: got %q, want %q", decoded.SDP, sd.SDP)
	}
}
