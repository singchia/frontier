package apis

// StreamSetupV1 opts a Service stream into the Frontier-to-Edge readiness RPC.
// It uses existing stream metadata/RPC packets; no bytes are added to raw IO.
const StreamSetupV1 = "frontier.stream-setup.v1"
const RPCStreamSetup = "__frontier_stream_setup_v1"

const (
	StreamReady          = "frontier.stream-setup.v1:ready"
	StreamEdgeOffline    = "frontier.stream-setup.v1:edge-not-online"
	StreamEdgeOpenFailed = "frontier.stream-setup.v1:edge-open-failed"
)
