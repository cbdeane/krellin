package daemon

import (
	"encoding/json"
	"fmt"
	"io"
)

type ConnectRequest struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	RepoRoot  string `json:"repo_root"`
	Subscribe bool   `json:"subscribe"`
}

type ConnectResponse struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
}

func ReadConnect(r io.Reader) (string, string, bool, error) {
	var req ConnectRequest
	dec := json.NewDecoder(r)
	if err := dec.Decode(&req); err != nil {
		return "", "", false, err
	}
	if req.Type != "connect" {
		return "", "", false, fmt.Errorf("invalid connect request")
	}
	return req.SessionID, req.RepoRoot, req.Subscribe, nil
}

func WriteConnect(w io.Writer, sessionID string, repoRoot string, subscribe bool) error {
	req := ConnectRequest{Type: "connect", SessionID: sessionID, RepoRoot: repoRoot, Subscribe: subscribe}
	enc := json.NewEncoder(w)
	return enc.Encode(req)
}

func WriteConnectResponse(w io.Writer, sessionID string) error {
	resp := ConnectResponse{Type: "connected", SessionID: sessionID}
	enc := json.NewEncoder(w)
	return enc.Encode(resp)
}
