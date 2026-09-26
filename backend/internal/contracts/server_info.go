package contracts

// ServerInfoDTO is the minimal public handshake; it contains no library paths
// or credentials. ProtocolVersion describes the connection contract, not the
// product release number.
type ServerInfoDTO struct {
	Product              string   `json:"product"`
	ServerID             string   `json:"serverId"`
	Name                 string   `json:"name"`
	Version              string   `json:"version"`
	ProtocolVersion      int      `json:"protocolVersion"`
	DesktopBridgeVersion int      `json:"desktopBridgeVersion"`
	Capabilities         []string `json:"capabilities"`
}
