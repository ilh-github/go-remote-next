mod config;
mod callback;
mod delegation;
mod fair_queue;
mod event_bus;
mod http_transport;
mod idempotency;
mod memory;
mod server;
mod trigger;
mod trust;
mod workflow;
mod workflow_job;
mod workflow_trigger;
mod ws_client;

use std::sync::Arc;

use clap::Parser;
use config::{AppConfig, TransportMode};
use event_bus::EventBus;
use rmcp::ServiceExt;
use rmcp::transport::stdio;
use server::SharedState;
use tracing_subscriber::EnvFilter;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    tracing_subscriber::fmt()
        .with_env_filter(EnvFilter::from_default_env())
        .with_writer(std::io::stderr)
        .init();

    let config = AppConfig::parse();

    tracing::info!("Connecting to QuickDesk at {}", config.ws_url);

    let event_bus = EventBus::new(500);

    let auth_token = config.auth_token();
    let ws = ws_client::WsClient::connect(&config.ws_url, auth_token, event_bus).await?;

    if let Some(ref devices) = config.allowed_devices {
        tracing::info!("Allowed devices: {:?}", devices);
    }
    if config.rate_limit > 0 {
        tracing::info!("Rate limit: {} requests/minute", config.rate_limit);
    }
    if config.session_timeout > 0 {
        tracing::info!("Session timeout: {}s", config.session_timeout);
    }

    let memory_store = memory::MemoryStore::new()
        .map_err(|e| format!("Failed to init memory store: {e}"))?;
    let workflow_store = workflow::WorkflowStore::new()
        .map_err(|e| format!("Failed to init workflow store: {e}"))?;
    let trust_engine = trust::TrustEngine::new()
        .map_err(|e| format!("Failed to init trust engine: {e}"))?;

    memory_store.cleanup_old_history(90);

    let shared_state = Arc::new(SharedState {
        memory: memory_store,
        workflow: workflow_store,
        trust: trust_engine,
    });

    {
        let state = shared_state.clone();
        let mut rx = ws.event_bus().subscribe();
        tokio::spawn(async move {
            while let Ok(ev) = rx.recv().await {
                match ev.event.as_str() {
                    "emergencyStopActivated" => {
                        let reason = ev.data.get("reason")
                            .and_then(|v| v.as_str())
                            .unwrap_or("gui");
                        state.trust.activate_emergency(reason);
                        tracing::info!("Emergency stop synced from GUI: {reason}");
                    }
                    "emergencyStopDeactivated" => {
                        state.trust.deactivate_emergency();
                        tracing::info!("Emergency stop deactivated from GUI");
                    }
                    _ => {}
                }
            }
        });
    }

    match config.transport {
        TransportMode::Stdio => {
            tracing::info!("Starting MCP server on stdio...");
            let mcp_server = server::QuickDeskMcpServer::new(
                ws,
                config.allowed_devices.unwrap_or_default(),
                shared_state,
            );
            let service = mcp_server.serve(stdio()).await?;
            service.waiting().await?;
        }
        TransportMode::Http => {
            http_transport::start_http(&config, ws, shared_state).await?;
        }
    }

    Ok(())
}
