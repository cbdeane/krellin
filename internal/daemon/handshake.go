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
}

type ConnectResponse struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
}

func ReadConnect(r io.Reader) (string, string, error) {
	var req ConnectRequest
	dec := json.NewDecoder(r)
	if err := dec.Decode(&req); err != nil {
		return "", "", err
	}
	if req.Type != "connect" {
		return "", "", fmt.Errorf("invalid connect request")
	}
	return req.SessionID, req.RepoRoot, nil
}

func WriteConnect(w io.Writer, sessionID string, repoRoot string) error {
	req := ConnectRequest{Type: "connect", SessionID: sessionID, RepoRoot: repoRoot}
	enc := json.NewEncoder(w)
	return enc.Encode(req)
}

func WriteConnectResponse(w io.Writer, sessionID string) error {
	resp := ConnectResponse{Type: "connected", SessionID: sessionID}
	enc := json.NewEncoder(w)
	return enc.Encode(resp)
}
