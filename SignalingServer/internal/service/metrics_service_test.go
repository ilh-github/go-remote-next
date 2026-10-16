package service

import "testing"

func TestMetricsSnapshotCountsRequestsAndConnections(t *testing.T) {
	metrics := NewMetricsService()
	metrics.RecordHTTPRequest("/v1/devices")
	metrics.RecordHTTPRequest("/health") // not an API request
	metrics.MarkSignalConnected(SignalRoleHost, "device-1", "")
	metrics.MarkSignalConnected(SignalRoleClient, "device-1", "client-1")
	metrics.MarkEventConnected(7)

	snapshot := metrics.Snapshot()
	if snapshot.APIRequestsTotal != 1 || snapshot.APIRequestsToday != 1 {
		t.Fatalf("request counters = total %d, today %d; want 1, 1",
			snapshot.APIRequestsTotal, snapshot.APIRequestsToday)
	}
	if snapshot.SignalHosts != 1 || snapshot.SignalClients != 1 || snapshot.EventStreams != 1 {
		t.Fatalf("connections = hosts %d, clients %d, events %d; want 1, 1, 1",
			snapshot.SignalHosts, snapshot.SignalClients, snapshot.EventStreams)
	}
	if snapshot.WebSocketConnections != 3 {
		t.Fatalf("websocket connections = %d, want 3", snapshot.WebSocketConnections)
	}

	metrics.MarkSignalDisconnected(SignalRoleHost, "device-1", "")
	metrics.MarkSignalDisconnected(SignalRoleClient, "device-1", "client-1")
	metrics.MarkEventDisconnected(7)
	snapshot = metrics.Snapshot()
	if snapshot.WebSocketConnections != 0 {
		t.Fatalf("websocket connections after disconnect = %d, want 0",
			snapshot.WebSocketConnections)
	}
	if snapshot.APIRequestsTotal != 1 {
		t.Fatalf("total request counter reset with a live gauge: %d", snapshot.APIRequestsTotal)
	}
}
