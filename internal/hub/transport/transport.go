package transport

import (
	"context"
	"errors"
	"fmt"

	"github.com/fxamacker/cbor/v2"
	"github.com/henrygd/beszel/internal/common"
	"github.com/henrygd/beszel/internal/entities/system"
)

// Transport defines the interface for hub-agent communication.
type Transport interface {
	Request(ctx context.Context, action common.WebSocketAction, req any, dest any) error
	IsConnected() bool
	Close()
}

// UnmarshalResponse unmarshals an AgentResponse into the destination type.
func UnmarshalResponse(resp common.AgentResponse, action common.WebSocketAction, dest any) error {
	if dest == nil {
		return errors.New("nil destination")
	}
	if len(resp.Data) > 0 {
		if err := cbor.Unmarshal(resp.Data, dest); err != nil {
			return fmt.Errorf("failed to unmarshal response data: %w", err)
		}
		return nil
	}
	return unmarshalLegacyResponse(resp, action, dest)
}

func unmarshalLegacyResponse(resp common.AgentResponse, action common.WebSocketAction, dest any) error {
	switch action {
	case common.GetData:
		d, ok := dest.(*system.CombinedData)
		if !ok {
			return fmt.Errorf("unexpected dest type for GetData: %T", dest)
		}
		if resp.SystemData == nil {
			return errors.New("no system data in response")
		}
		*d = *resp.SystemData
		return nil
	case common.CheckFingerprint:
		d, ok := dest.(*common.FingerprintResponse)
		if !ok {
			return fmt.Errorf("unexpected dest type for CheckFingerprint: %T", dest)
		}
		if resp.Fingerprint == nil {
			return errors.New("no fingerprint in response")
		}
		*d = *resp.Fingerprint
		return nil
	}
	return fmt.Errorf("unsupported action: %d", action)
}
