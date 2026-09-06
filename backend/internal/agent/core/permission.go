package core

import (
	"fmt"
	"strings"
)

func allowCall(def ToolDefinition, call Call, settings Settings) error {
	if settings.Disabled {
		return fmt.Errorf("AI is disabled")
	}
	if settings.ReadOnly && def.Permission != PermissionRead {
		return fmt.Errorf("AI is read-only")
	}
	token := strings.TrimSpace(call.ConfirmTok)
	switch def.Permission {
	case PermissionRead:
		return nil
	case PermissionWritePreview:
		if call.Channel == ChannelMCP && token != "" && !settings.MCPWriteExposed {
			return fmt.Errorf("MCP write tools are not exposed")
		}
		// Chat-loop models must not apply; human confirm uses the action channel.
		if token != "" && call.Channel == ChannelChat && !settings.WriteEnabled && !settings.LowRiskDirectWrite {
			return fmt.Errorf("agent write permission is disabled")
		}
		return nil
	case PermissionWriteApply:
		if call.Channel == ChannelMCP && !settings.MCPWriteExposed {
			return fmt.Errorf("MCP write tools are not exposed")
		}
		if token == "" && !settings.LowRiskDirectWrite {
			return fmt.Errorf("%w", ErrConfirmRequired)
		}
		if call.Channel == ChannelChat && !settings.WriteEnabled && !settings.LowRiskDirectWrite {
			return fmt.Errorf("agent write permission is disabled")
		}
		return nil
	default:
		return fmt.Errorf("unknown permission %q", def.Permission)
	}
}
