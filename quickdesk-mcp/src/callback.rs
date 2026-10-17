use std::time::{Duration, SystemTime, UNIX_EPOCH};

use http_body_util::Full;
use hyper::body::Bytes;
use hyper::Request;
use hyper_util::client::legacy::Client;
use hyper_util::rt::TokioExecutor;
use serde_json::Value;
use sha1::{Digest, Sha1};
use uuid::Uuid;

const HMAC_BLOCK_SIZE: usize = 64;

pub struct CallbackClient {
    client: Client<hyper_util::client::legacy::connect::HttpConnector, Full<Bytes>>,
    secret: Option<String>,
}

pub async fn deliver_workflow_callback(url: Option<&str>, event: serde_json::Value) {
    let Some(url) = url else {
        return;
    };
    let result = match CallbackClient::new() {
        Ok(client) => client.deliver(url, &event).await,
        Err(error) => Err(error),
    };
    if let Err(error) = result {
        tracing::warn!("workflow callback to {} failed: {}", url, error);
    }
}

impl CallbackClient {
    pub fn new() -> Result<Self, String> {
        let secret = std::env::var("QUICKDESK_CALLBACK_SECRET")
            .ok()
            .filter(|value| !value.is_empty());
        let client = Client::builder(TokioExecutor::new())
            .pool_idle_timeout(Duration::from_secs(30))
            .build_http();
        Ok(Self { client, secret })
    }

    pub async fn deliver(&self, url: &str, event: &Value) -> Result<(), String> {
        let secret = self
            .secret
            .as_deref()
            .ok_or_else(|| "QUICKDESK_CALLBACK_SECRET is not configured".to_string())?;
        let parsed: hyper::Uri = url
            .parse()
            .map_err(|error| format!("invalid callback URL: {error}"))?;
        if parsed.scheme_str() != Some("http") {
            return Err(
                "callback URL must use http; put HTTPS behind a local TLS reverse proxy"
                    .to_string(),
            );
        }

        let body = serde_json::to_vec(event)
            .map_err(|error| format!("serialize callback event: {error}"))?;
        let timestamp = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map_err(|error| format!("callback clock error: {error}"))?
            .as_secs()
            .to_string();
        let signature = sign(secret, &timestamp, &body);
        let event_name = event
            .get("event")
            .and_then(Value::as_str)
            .unwrap_or("workflow.run");

        let request = Request::builder()
            .method("POST")
            .uri(parsed)
            .header("Content-Type", "application/json")
            .header("X-QuickDesk-Event", event_name)
            .header("X-QuickDesk-Delivery-Id", Uuid::new_v4().to_string())
            .header("X-QuickDesk-Timestamp", timestamp)
            .header("X-QuickDesk-Signature", format!("hmac-sha1={signature}"))
            .body(Full::new(Bytes::from(body)))
            .map_err(|error| format!("build callback request: {error}"))?;
        let response = self
            .client
            .request(request)
            .await
            .map_err(|error| format!("callback request failed: {error}"))?;
        if !response.status().is_success() {
            return Err(format!("callback returned HTTP {}", response.status()));
        }
        Ok(())
    }
}

fn sign(secret: &str, timestamp: &str, body: &[u8]) -> String {
    let mut message = Vec::with_capacity(timestamp.len() + 1 + body.len());
    message.extend_from_slice(timestamp.as_bytes());
    message.push(b'.');
    message.extend_from_slice(body);
    hex(&hmac_sha1(secret.as_bytes(), &message))
}

fn hmac_sha1(key: &[u8], message: &[u8]) -> Vec<u8> {
    let key = if key.len() > HMAC_BLOCK_SIZE {
        Sha1::digest(key).to_vec()
    } else {
        key.to_vec()
    };
    let mut inner_pad = vec![0x36; HMAC_BLOCK_SIZE];
    let mut outer_pad = vec![0x5c; HMAC_BLOCK_SIZE];
    for (index, byte) in key.iter().enumerate() {
        inner_pad[index] ^= byte;
        outer_pad[index] ^= byte;
    }

    let mut inner = Sha1::new();
    inner.update(&inner_pad);
    inner.update(message);
    let inner_digest = inner.finalize();

    let mut outer = Sha1::new();
    outer.update(&outer_pad);
    outer.update(inner_digest);
    outer.finalize().to_vec()
}

fn hex(bytes: &[u8]) -> String {
    const HEX: &[u8; 16] = b"0123456789abcdef";
    let mut output = String::with_capacity(bytes.len() * 2);
    for byte in bytes {
        output.push(HEX[(byte >> 4) as usize] as char);
        output.push(HEX[(byte & 0x0f) as usize] as char);
    }
    output
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn hmac_sha1_matches_rfc_2202_case_1() {
        let digest = hmac_sha1(
            &[0x0b; 20],
            b"Hi There",
        );
        assert_eq!(
            hex(&digest),
            "b617318655057264e28bc0b6fb378c8ef146be00"
        );
    }
}
