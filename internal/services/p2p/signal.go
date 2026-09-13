package p2p

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"github.com/pion/webrtc/v4"
)

type signal struct {
	Type string `json:"type"`
	SDP  string `json:"sdp"`
}

func encode(sd *webrtc.SessionDescription) string {
	b, _ := json.Marshal(signal{Type: sd.Type.String(), SDP: sd.SDP})
	var buf bytes.Buffer
	w, _ := flate.NewWriter(&buf, flate.BestCompression)
	w.Write(b)
	w.Close()
	return base64.RawURLEncoding.EncodeToString(buf.Bytes())
}

func decode(s string) (webrtc.SessionDescription, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}
	r := flate.NewReader(bytes.NewReader(raw))
	defer r.Close()
	b, err := io.ReadAll(r)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}
	var sig signal
	if err := json.Unmarshal(b, &sig); err != nil {
		return webrtc.SessionDescription{}, err
	}
	t, ok := parseSDPType(sig.Type)
	if !ok {
		return webrtc.SessionDescription{}, fmt.Errorf("unknown sdp type: %s", sig.Type)
	}
	return webrtc.SessionDescription{Type: t, SDP: sig.SDP}, nil
}

func parseSDPType(s string) (webrtc.SDPType, bool) {
	switch s {
	case "offer":
		return webrtc.SDPTypeOffer, true
	case "answer":
		return webrtc.SDPTypeAnswer, true
	case "pranswer":
		return webrtc.SDPTypePranswer, true
	case "rollback":
		return webrtc.SDPTypeRollback, true
	}
	return webrtc.SDPType(0), false
}
