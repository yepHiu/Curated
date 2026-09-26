package server

import (
	"encoding/xml"
	"net/http"

	"curated-backend/internal/config"
	"curated-backend/internal/discovery"
)

type discoveryDescription struct {
	XMLName xml.Name        `xml:"root"`
	XMLNS   string          `xml:"xmlns,attr"`
	Major   int             `xml:"specVersion>major"`
	Minor   int             `xml:"specVersion>minor"`
	Device  discoveryDevice `xml:"device"`
}
type discoveryDevice struct {
	Type            string `xml:"deviceType"`
	Name            string `xml:"friendlyName"`
	Manufacturer    string `xml:"manufacturer"`
	Model           string `xml:"modelName"`
	UDN             string `xml:"UDN"`
	PresentationURL string `xml:"presentationURL"`
	ServiceType     string `xml:"serviceList>service>serviceType"`
}

func (h *Handler) handleDiscoveryDescription(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.LANEnabled || config.HTTPAddrIsLoopback(h.cfg.HttpAddr) || !h.cfg.DiscoveryOn() || h.cfg.ServerID == "" {
		http.NotFound(w, r)
		return
	}
	info := h.serverInfo()
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(xml.Header))
	_ = xml.NewEncoder(w).Encode(discoveryDescription{XMLNS: "urn:schemas-upnp-org:device-1-0", Major: 1, Device: discoveryDevice{
		Type: "urn:curated:device:LibraryServer:1", Name: info.Name, Manufacturer: "Curated", Model: "Curated Server", UDN: "uuid:" + info.ServerID, PresentationURL: "/", ServiceType: discovery.ServiceType,
	}})
}
