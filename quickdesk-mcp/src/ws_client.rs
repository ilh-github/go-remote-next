use futures_util::{SinkExt, StreamExt};
use serde_json::Value;
use std::collections::HashMap;
use std::sync::atomic::{AtomicI64, Ordering};
use std::sync::Arc;
use std::time::Duration;
use tokio::sync::{oneshot, Mutex};
use tokio_tungstenite::{connect_async, tungstenite::Message};

use crate::event_bus::{Event, EventBus};

type PendingMap = HashMap<String, oneshot::Sender<Result<Value, String>>>;

/// Live connection state — replaced atomically on reconnect.
struct WsInner {
    sender: futures_util::stream::SplitSink<
        tokio_tungstenite::WebSocketStream<
            tokio_tungstenite::MaybeTlsStream<tokio::net::TcpStream>,
        >,
        Message,
    >,
    pending: Arc<Mutex<PendingMap>>,
    req_counter: u64,
}

/// Auto-reconnecting WebSocket client.
///
/// On every `request()` call, if the underlying connection is gone
/// (dropped by the remote or due to a send error) it transparently
/// reconnects before retrying the request once.
#[derive(Clone)]
pub struct WsClient {
    url: Arc<String>,
    token: Arc<Option<String>>,
    /// `None` means the connection is currently down.
    inner: Arc<Mutex<Option<WsInner>>>,
    event_bus: EventBus,
    last_control_plane_rev: Arc<AtomicI64>,
}

impl WsClient {
    pub async fn connect(
        url: &str,
        token: Option<&str>,
        event_bus: EventBus,
    ) -> Result<Self, String> {
        let client = Self {
            url: Arc::new(url.to_string()),
            token: Arc::new(token.map(str::to_string)),
            inner: Arc::new(Mutex::new(None)),
            event_bus,
            last_control_plane_rev: Arc::new(AtomicI64::new(0)),
        };
        client.do_connect().await?;
        Ok(client)
    }

    #[cfg(test)]
    pub(crate) fn disconnected() -> Self {
        Self {
            url: Arc::new("ws://127.0.0.1:1".to_string()),
            token: Arc::new(None),
            inner: Arc::new(Mutex::new(None)),
            event_bus: EventBus::new(16),
            last_control_plane_rev: Arc::new(AtomicI64::new(0)),
        }
    }

    /// Establish (or re-establish) the WebSocket connection.
    /// Must be called with `self.inner` lock NOT held.
    async fn do_connect(&self) -> Result<(), String> {
        let (ws_stream, _) = connect_async(self.url.as_str())
            .await
            .map_err(|e| format!("WebSocket connect failed: {e}"))?;

        let (write, read) = ws_stream.split();
        let pending: Arc<Mutex<PendingMap>> = Arc::new(Mutex::new(HashMap::new()));

        // Spawn the reader; it clears `inner` when the connection closes.
        tokio::spawn(Self::reader_loop(
            read,
            pending.clone(),
            self.inner.clone(),
            self.event_bus.clone(),
            self.last_control_plane_rev.clone(),
        ));

        let mut guard = self.inner.lock().await;
        *guard = Some(WsInner {
            sender: write,
            pending,
            req_counter: 0,
        });
        drop(guard);

        // Authenticate if a token was provided.
        if let Some(ref tok) = *self.token {
            self.authenticate(tok).await?;
        }

        tracing::info!("WsClient connected to {}", self.url);
        Ok(())
    }

    async fn authenticate(&self, token: &str) -> Result<(), String> {
        let since_server_rev = self.last_control_plane_rev.load(Ordering::Relaxed);
        let mut params = serde_json::json!({ "token": token });
        if since_server_rev > 0 {
            params["since_server_rev"] = serde_json::json!(since_server_rev);
        }
        let resp = self
            .try_request("auth", params, Duration::from_secs(10))
            .await?;
        if resp.get("authenticated").and_then(|v| v.as_bool()) == Some(true) {
            Ok(())
        } else {
            Err("Authentication failed".to_string())
        }
    }

    pub async fn request(&self, method: &str, params: Value) -> Result<Value, String> {
        self.request_with_timeout(method, params, Duration::from_secs(30))
            .await
    }

    pub async fn request_with_timeout(
        &self,
        method: &str,
        params: Value,
        timeout: Duration,
    ) -> Result<Value, String> {
        match self.try_request(method, params.clone(), timeout).await {
            Ok(v) => Ok(v),
            Err(e) => {
                tracing::warn!(
                    "WsClient request '{}' failed ({}), reconnecting…",
                    method,
                    e
                );
                self.do_connect().await?;
                self.try_request(method, params, timeout).await
            }
        }
    }

    async fn try_request(
        &self,
        method: &str,
        params: Value,
        timeout: Duration,
    ) -> Result<Value, String> {
        let (id, rx) = {
            let mut guard = self.inner.lock().await;
            let inner = guard.as_mut().ok_or("Not connected")?;

            inner.req_counter += 1;
            let id = format!("req_{}", inner.req_counter);

            let (tx, rx) = oneshot::channel();
            inner.pending.lock().await.insert(id.clone(), tx);

            let msg = serde_json::json!({ "id": id, "method": method, "params": params });
            let text = serde_json::to_string(&msg).unwrap();

            inner
                .sender
                .send(Message::Text(text.into()))
                .await
                .map_err(|e| format!("WebSocket send failed: {e}"))?;

            (id, rx)
        };
        let _ = id;

        let secs = timeout.as_secs();
        match tokio::time::timeout(timeout, rx).await {
            Ok(Ok(v)) => v,
            Ok(Err(_)) => Err("Response channel closed".to_string()),
            Err(_) => Err(format!("WebSocket request timed out after {secs}s")),
        }
    }

    async fn reader_loop(
        mut read: futures_util::stream::SplitStream<
            tokio_tungstenite::WebSocketStream<
                tokio_tungstenite::MaybeTlsStream<tokio::net::TcpStream>,
            >,
        >,
        pending: Arc<Mutex<PendingMap>>,
        inner: Arc<Mutex<Option<WsInner>>>,
        event_bus: EventBus,
        last_control_plane_rev: Arc<AtomicI64>,
    ) {
        while let Some(msg) = read.next().await {
            let msg = match msg {
                Ok(Message::Text(t)) => t,
                Ok(Message::Close(_)) => break,
                Err(e) => {
                    tracing::warn!("WsClient reader error: {e}");
                    break;
                }
                _ => continue,
            };

            let parsed: Value = match serde_json::from_str(&msg) {
                Ok(v) => v,
                Err(_) => continue,
            };

            if let Some(id) = parsed.get("id").and_then(|v| v.as_str()) {
                let mut map = pending.lock().await;
                if let Some(tx) = map.remove(id) {
                    if let Some(err) = parsed.get("error") {
                        let msg = err
                            .get("message")
                            .and_then(|v| v.as_str())
                            .unwrap_or("Unknown error");
                        let _ = tx.send(Err(msg.to_string()));
                    } else if let Some(result) = parsed.get("result") {
                        let _ = tx.send(Ok(result.clone()));
                    } else {
                        let _ = tx.send(Ok(parsed));
                    }
                }
            } else if let Some(event_name) = parsed.get("event").and_then(|v| v.as_str()) {
                let data = parsed.get("data").cloned().unwrap_or(Value::Null);
                if event_name == "controlPlaneEvent" {
                    let server_rev = data
                        .get("server_rev")
                        .and_then(Value::as_i64)
                        .unwrap_or_default();
                    if server_rev > 0 {
                        last_control_plane_rev.fetch_max(server_rev, Ordering::Relaxed);
                    }
                }
                let timestamp = std::time::SystemTime::now()
                    .duration_since(std::time::UNIX_EPOCH)
                    .unwrap_or_default()
                    .as_millis() as u64;

                tracing::debug!("Received event: {} data={}", event_name, data);
                event_bus
                    .publish(Event {
                        event: event_name.to_string(),
                        data,
                        timestamp,
                    })
                    .await;
            }
        }

        // Connection closed — mark inner as None so the next request triggers reconnect.
        tracing::info!("WsClient connection closed, will reconnect on next request");
        // Fail any in-flight requests.
        let mut map = pending.lock().await;
        for (_, tx) in map.drain() {
            let _ = tx.send(Err("Connection closed".to_string()));
        }
        drop(map);
        *inner.lock().await = None;
    }

    pub fn event_bus(&self) -> &EventBus {
        &self.event_bus
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tokio_tungstenite::accept_async;

    #[tokio::test]
    async fn reconnect_auth_replays_since_last_control_plane_revision() {
        let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
        let address = listener.local_addr().unwrap();
        let (auth_tx, mut auth_rx) = tokio::sync::mpsc::unbounded_channel();

        let server = tokio::spawn(async move {
            for connection_index in 0..2 {
                let (stream, _) = listener.accept().await.unwrap();
                let mut socket = accept_async(stream).await.unwrap();
                let Some(Ok(Message::Text(auth_text))) = socket.next().await else {
                    panic!("expected auth request");
                };
                let auth: Value = serde_json::from_str(&auth_text).unwrap();
                let auth_id = auth.get("id").and_then(Value::as_str).unwrap().to_string();
                auth_tx.send(auth).unwrap();
                socket
                    .send(Message::Text(
                        serde_json::json!({
                            "id": auth_id,
                            "result": {"authenticated": true}
                        })
                        .to_string()
                        .into(),
                    ))
                    .await
                    .unwrap();

                if connection_index == 0 {
                    let Some(Ok(Message::Text(trigger_text))) = socket.next().await else {
                        panic!("expected trigger request");
                    };
                    let trigger: Value = serde_json::from_str(&trigger_text).unwrap();
                    let trigger_id = trigger.get("id").and_then(Value::as_str).unwrap();
                    socket
                        .send(Message::Text(
                            serde_json::json!({
                                "id": trigger_id,
                                "result": {"ok": true}
                            })
                            .to_string()
                            .into(),
                        ))
                        .await
                        .unwrap();
                    socket
                        .send(Message::Text(
                            serde_json::json!({
                                "event": "controlPlaneEvent",
                                "data": {
                                    "id": "evt-42",
                                    "type": "device.online.changed",
                                    "device_id": "device-1",
                                    "server_rev": 42,
                                    "data": {"online": true}
                                }
                            })
                            .to_string()
                            .into(),
                        ))
                        .await
                        .unwrap();
                    tokio::time::sleep(Duration::from_millis(50)).await;
                    socket.close(None).await.unwrap();
                } else {
                    let Some(Ok(Message::Text(ping_text))) = socket.next().await else {
                        panic!("expected ping request");
                    };
                    let ping: Value = serde_json::from_str(&ping_text).unwrap();
                    let ping_id = ping.get("id").and_then(Value::as_str).unwrap();
                    socket
                        .send(Message::Text(
                            serde_json::json!({
                                "id": ping_id,
                                "result": {"ok": true}
                            })
                            .to_string()
                            .into(),
                        ))
                        .await
                        .unwrap();
                }
            }
        });

        let client = WsClient::connect(
            &format!("ws://{address}"),
            Some("local-api-token"),
            EventBus::new(16),
        )
        .await
        .unwrap();
        let first_auth = tokio::time::timeout(Duration::from_secs(2), auth_rx.recv())
            .await
            .unwrap()
            .unwrap();
        assert!(first_auth["params"].get("since_server_rev").is_none());

        client
            .request("trigger", serde_json::json!({}))
            .await
            .unwrap();
        tokio::time::sleep(Duration::from_millis(100)).await;
        client.request("ping", serde_json::json!({})).await.unwrap();

        let second_auth = tokio::time::timeout(Duration::from_secs(2), auth_rx.recv())
            .await
            .unwrap()
            .unwrap();
        assert_eq!(second_auth["params"]["since_server_rev"], 42);
        server.await.unwrap();
    }
}
