use std::sync::Arc;

use axum::Router;
use rmcp::transport::streamable_http_server::{
    StreamableHttpServerConfig, StreamableHttpService,
    session::local::LocalSessionManager,
};
use tokio::io::AsyncReadExt;
use tokio::sync::Semaphore;
use tokio_util::sync::CancellationToken;
use tower_http::cors::{AllowHeaders, AllowMethods, CorsLayer};

use crate::config::AppConfig;
use crate::idempotency::WorkflowIdempotencyStore;
use crate::server::{QuickDeskMcpServer, SharedState};
use crate::trigger::TriggerRuntime;
use crate::workflow_job::WorkflowJobManager;
use crate::workflow_trigger::WorkflowTriggerStore;
use crate::ws_client::WsClient;

pub async fn start_http(
    config: &AppConfig,
    ws: WsClient,
    shared_state: Arc<SharedState>,
) -> Result<(), Box<dyn std::error::Error>> {
    let ct = CancellationToken::new();
    let allowed_devices = config.allowed_devices.clone().unwrap_or_default();
    let session_manager = Arc::new(LocalSessionManager::default());
    let mcp_ws = ws.clone();
    let mcp_shared_state = shared_state.clone();
    let mcp_allowed_devices = allowed_devices.clone();
    let workflow_triggers = WorkflowTriggerStore::new()?;
    let idempotency = WorkflowIdempotencyStore::new()?;
    let business_run_slots = Arc::new(Semaphore::new(1));
    let jobs = WorkflowJobManager::new(
        ct.child_token(),
        ws.clone(),
        shared_state.clone(),
        allowed_devices.clone(),
        business_run_slots.clone(),
    )?;

    let mcp_service: StreamableHttpService<QuickDeskMcpServer, LocalSessionManager> =
        StreamableHttpService::new(
            move || {
                Ok(QuickDeskMcpServer::new(
                    mcp_ws.clone(),
                    mcp_allowed_devices.clone(),
                    mcp_shared_state.clone(),
                ))
            },
            session_manager,
            StreamableHttpServerConfig {
                stateful_mode: !config.stateless,
                cancellation_token: ct.child_token(),
                ..Default::default()
            },
        );

    let trigger_runtime = TriggerRuntime {
        token: config.trigger_token.clone(),
        ws: ws.clone(),
        shared_state: shared_state.clone(),
        allowed_devices: allowed_devices.clone(),
        run_slots: business_run_slots.clone(),
        triggers: workflow_triggers.clone(),
        idempotency: idempotency.clone(),
        jobs: jobs.clone(),
    };
    let business_router = crate::trigger::router(trigger_runtime.clone());
    if config
        .trigger_token
        .as_deref()
        .is_some_and(|token| !token.is_empty())
    {
        crate::trigger::start_scheduler(ct.child_token(), trigger_runtime);
    }
    let app = Router::new()
        .route(
            "/health",
            axum::routing::get(|| async { "ok" }),
        )
        .nest_service("/mcp", mcp_service)
        .merge(business_router);

    // Apply CORS layer when cors_origin is configured
    let app = if let Some(origins) = &config.cors_origin {
        let allowed_origins: Vec<axum::http::HeaderValue> = origins
            .iter()
            .filter_map(|o| o.parse().ok())
            .collect();

        let cors = CorsLayer::new()
            .allow_origin(allowed_origins)
            .allow_methods(AllowMethods::any())
            .allow_headers(AllowHeaders::any());

        tracing::info!("CORS enabled for origins: {:?}", origins);
        app.layer(cors)
    } else {
        app.layer(CorsLayer::permissive())
    };

    let bind_addr = format!("{}:{}", config.host, config.port);
    let listener = tokio::net::TcpListener::bind(&bind_addr).await?;
    let local_addr = listener.local_addr()?;

    tracing::info!("MCP HTTP server listening on http://{}", local_addr);
    tracing::info!(
        "  MCP endpoint: http://{}/mcp",
        local_addr
    );
    tracing::info!(
        "  Health check: http://{}/health",
        local_addr
    );
    tracing::info!(
        "  Session mode: {}",
        if config.stateless { "stateless" } else { "stateful" }
    );
    if config
        .trigger_token
        .as_deref()
        .is_some_and(|token| !token.is_empty())
    {
        tracing::info!(
            "  Business workflow trigger: POST http://{}/v1/workflows/{{workflow_id}}/run",
            local_addr
        );
    }

    axum::serve(listener, app)
        .with_graceful_shutdown(async move {
            let ctrl_c = tokio::signal::ctrl_c();
            let stdin_eof = async {
                // Detect parent process closing our stdin pipe.
                // On Windows, QProcess::terminate() sends WM_CLOSE which
                // console apps can't receive. Instead, the parent closes
                // our stdin before calling terminate, and we detect the EOF.
                let mut stdin = tokio::io::stdin();
                let mut buf = [0u8; 1];
                loop {
                    match stdin.read(&mut buf).await {
                        Ok(0) | Err(_) => break, // EOF or error
                        Ok(_) => continue,        // ignore stray bytes
                    }
                }
            };
            tokio::select! {
                _ = ctrl_c => {},
                _ = stdin_eof => {},
            }
            tracing::info!("Shutting down HTTP server...");
            ct.cancel();
        })
        .await?;

    Ok(())
}
