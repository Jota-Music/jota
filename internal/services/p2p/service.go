package p2p

import (
	"errors"
	"log"
	"strings"
	"sync"

	"github.com/pion/webrtc/v4"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	stunServer  = "stun:stun.l.google.com:19302"
	channelName = "jota-sync"
)

type Config struct {
	TURNURL  string
	TURNUser string
	TURNPass string
}

func servers(cfg Config) []webrtc.ICEServer {
	list := []webrtc.ICEServer{{URLs: []string{stunServer}}}
	if cfg.TURNURL == "" {
		return list
	}
	return append(list, webrtc.ICEServer{
		URLs:       strings.Split(cfg.TURNURL, ","),
		Username:   cfg.TURNUser,
		Credential: cfg.TURNPass,
	})
}

type role int

const (
	roleNone role = iota
	roleHost
	roleGuest
)

type Service struct {
	mu         sync.Mutex
	role       role
	pc         *webrtc.PeerConnection
	dc         *webrtc.DataChannel
	iceServers []webrtc.ICEServer
}

func New(cfg Config) *Service {
	return &Service{iceServers: servers(cfg)}
}

func (s *Service) StartHost() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.role != roleNone {
		return "", errors.New("already in a session")
	}

	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{ICEServers: s.iceServers})
	if err != nil {
		return "", err
	}

	s.pc = pc
	s.role = roleHost
	s.setupCallbacks(pc)

	dc, err := pc.CreateDataChannel(channelName, nil)
	if err != nil {
		s.resetLocked()
		return "", err
	}
	s.dc = dc
	s.setupDataChannel(dc)

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		s.resetLocked()
		return "", err
	}
	if err := pc.SetLocalDescription(offer); err != nil {
		s.resetLocked()
		return "", err
	}

	<-webrtc.GatheringCompletePromise(pc)
	return encode(pc.LocalDescription()), nil
}

func (s *Service) JoinGuest(offerCode string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.role != roleNone {
		return "", errors.New("already in a session")
	}

	offer, err := decode(offerCode)
	if err != nil {
		return "", err
	}

	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{ICEServers: s.iceServers})
	if err != nil {
		return "", err
	}

	s.pc = pc
	s.role = roleGuest
	s.setupCallbacks(pc)

	if err := pc.SetRemoteDescription(offer); err != nil {
		s.resetLocked()
		return "", err
	}

	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		s.mu.Lock()
		s.dc = dc
		s.mu.Unlock()
		s.setupDataChannel(dc)
	})

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		s.resetLocked()
		return "", err
	}
	if err := pc.SetLocalDescription(answer); err != nil {
		s.resetLocked()
		return "", err
	}

	<-webrtc.GatheringCompletePromise(pc)
	return encode(pc.LocalDescription()), nil
}

func (s *Service) AcceptAnswer(answerCode string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.role != roleHost {
		return errors.New("not the host")
	}

	answer, err := decode(answerCode)
	if err != nil {
		return err
	}

	return s.pc.SetRemoteDescription(answer)
}

func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resetLocked()
}

func (s *Service) Send(payload string) error {
	s.mu.Lock()
	dc := s.dc
	s.mu.Unlock()

	if dc == nil {
		log.Printf("p2p: send dc nil")
		return errors.New("not connected")
	}
	if err := dc.SendText(payload); err != nil {
		log.Printf("p2p: send error %v", err)
		return err
	}
	log.Printf("p2p: sent %d bytes", len(payload))
	return nil
}

func (s *Service) setupCallbacks(pc *webrtc.PeerConnection) {
	pc.OnConnectionStateChange(func(st webrtc.PeerConnectionState) {
		log.Printf("p2p: peer state %s", st.String())
		switch st {
		case webrtc.PeerConnectionStateFailed:
			emit("p2p:error", "connection failed")
			emit("p2p:closed")
			s.Stop()
		case webrtc.PeerConnectionStateClosed:
			emit("p2p:closed")
			s.Stop()
		}
	})
	pc.OnICEConnectionStateChange(func(st webrtc.ICEConnectionState) {
		if st == webrtc.ICEConnectionStateFailed {
			emit("p2p:error", "ice failed")
		}
	})
}

func (s *Service) setupDataChannel(dc *webrtc.DataChannel) {
	dc.OnOpen(func() {
		log.Printf("p2p: channel open")
		emit("p2p:connected")
	})
	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		log.Printf("p2p: message %s", string(msg.Data))
		emit("p2p:message", string(msg.Data))
	})
	dc.OnClose(func() {
		log.Printf("p2p: channel close")
		emit("p2p:closed")
	})
}

func (s *Service) resetLocked() {
	if s.pc != nil {
		s.pc.Close()
	}
	s.role = roleNone
	s.pc = nil
	s.dc = nil
}

func emit(name string, data ...any) {
	app := application.Get()
	if app == nil {
		return
	}
	app.Event.Emit(name, data...)
}
