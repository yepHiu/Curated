import { describe, expect, it } from "vitest"
import { parseDiscoveryReply, serviceType } from "./discovery"
const id = "18a31111-36e0-424b-9f9f-d1e9cfb74ed2"
const reply = `HTTP/1.1 200 OK\r\nST: ${serviceType}\r\nUSN: uuid:${id}::${serviceType}\r\nLOCATION: http://192.168.1.2:8081/discovery/description.xml\r\nCACHE-CONTROL: max-age=120\r\n\r\n`
describe("SSDP candidate validation", () => {
  it("accepts a bounded response from the advertised host", () => {
    expect(parseDiscoveryReply(reply,"192.168.1.2")).toEqual({url:"http://192.168.1.2:8081",serverId:id,maxAge:120})
  })
  it("rejects foreign services, unrelated hosts, credentials, expired and ambiguous packets", () => {
    for(const packet of [reply.replace(serviceType,"ssdp:all"),reply.replace("http://192.168.1.2", "http://127.0.0.1"),reply.replace("http://", "http://u:p@"),reply.replace("max-age=120","max-age=0"),reply.replace("ST:",`ST: ${serviceType}\r\nST:`),reply.replace("description.xml", "../api/settings"),"x".repeat(2049)]) expect(parseDiscoveryReply(packet,"192.168.1.2")).toBeUndefined()
  })
})
