package mcp

import (
	"encoding/json"
	"net/http"
)

// HTTPHandler returns an http.Handler that processes JSON-RPC requests over HTTP.
// Supports both single Request and batched Request arrays in the body.
// This is a streamable-HTTP-style transport: each POST request returns
// the corresponding response(s); GET is used for keepalive.
func (s *Server) HTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// SSE/streaming keepalive; respond with empty 200.
			w.WriteHeader(http.StatusOK)
			return
		case http.MethodDelete:
			// Session termination; respond with 204.
			w.WriteHeader(http.StatusNoContent)
			return
		case http.MethodPost:
			s.handleHTTPPost(w, r)
			return
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
	})
}

func (s *Server) handleHTTPPost(w http.ResponseWriter, r *http.Request) {
	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeJSONRPCError(w, http.StatusBadRequest, ErrCodeParse, err.Error())
		return
	}

	// Try as single request first.
	trimmed := trimSpaceJSON(raw)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		var batch []Request
		if err := json.Unmarshal(raw, &batch); err != nil {
			writeJSONRPCError(w, http.StatusBadRequest, ErrCodeParse, err.Error())
			return
		}
		responses := make([]*Response, 0, len(batch))
		for i := range batch {
			resp := s.HandleRequest(r.Context(), &batch[i])
			if resp != nil {
				responses = append(responses, resp)
			}
		}
		if len(responses) == 0 {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(responses)
		return
	}

	var req Request
	if err := json.Unmarshal(raw, &req); err != nil {
		writeJSONRPCError(w, http.StatusBadRequest, ErrCodeParse, err.Error())
		return
	}
	resp := s.HandleRequest(r.Context(), &req)
	if resp == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func writeJSONRPCError(w http.ResponseWriter, status, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(&Response{
		JSONRPC: JSONRPCVersion,
		Error:   &RPCError{Code: code, Message: msg},
	})
}

func trimSpaceJSON(b []byte) []byte {
	for len(b) > 0 && (b[0] == ' ' || b[0] == '\t' || b[0] == '\n' || b[0] == '\r') {
		b = b[1:]
	}
	return b
}
