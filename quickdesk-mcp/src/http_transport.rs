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
use crate::delegation::DelegationStore;
use crate::distributed_claim::DistributedClaimStore;
use crate::idempotency::WorkflowIdempotencyStore;
use crate::server::{QuickDeskMcpServer, SharedState};
use crate::trigger::TriggerRuntime;
use crate::workflow_job::{parse_peer_dirs, WorkflowJobManager};
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
    let delegation = DelegationStore::new()?;
    let business_run_slots = Arc::new(Semaphore::new(1));
    let jobs = WorkflowJobManager::new(
        ct.child_token(),
        ws.clone(),
        shared_state.clone(),
        allowed_devices.clone(),
        business_run_slots.clone(),
    )?;
    // A shared claim database is what makes a multi-machine deployment safe:
    // without it every worker's claim lives on its own filesystem, and two of
    // them would each believe they own the same run.
    let jobs = match config.workflow_claim_db.as_deref().map(str::trim) {
        Some(path) if !path.is_empty() => {
            let claims = DistributedClaimStore::open(path)
                .map_err(|error| format!("cannot open workflow claim database {path}: {error}"))?;
            let worker_id = format!(
                "{}@{}",
                std::process::id(),
                hostname().unwrap_or_else(|| "unknown-host".to_string())
            );
            tracing::info!(
                "workflow execution claims are shared at {path}; this worker is {worker_id}"
            );
            jobs.with_distributed_claims(claims, worker_id)
        }
        _ => jobs,
    };
    // Peer directories turn a set of independent workers into a fleet: without
    // them a run sits in the queue of whichever process accepted it, so that
    // process being down strands the run. The claim store configured above is
    // what keeps watching a peer from turning into executing it twice, which is
    // why this is only worth setting together with it.
    let jobs = match config
        .workflow_peer_job_dirs
        .as_deref()
        .map(str::trim)
        .filter(|value| !value.is_empty())
    {
        Some(list) => {
            let dirs = parse_peer_dirs(list);
            tracing::info!(
                "workflow job directories watched for peers: {}",
                dirs.iter()
                    .map(|dir| dir.display().to_string())
                    .collect::<Vec<_>>()
                    .join(", ")
            );
            jobs.with_peer_job_dirs(dirs)
        }
        None => jobs,
    };

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
        delegation: delegation.clone(),
        delegation_required: config.delegation_required,
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

/// Best-effort host name, used only to make a worker id readable in the shared
/// claim store. `HOSTNAME` is what containers and systemd set; falling back to a
/// marker is fine because the process id already makes the id unique.
fn hostname() -> Option<String> {
    std::env::var("HOSTNAME")
        .ok()
        .map(|value| value.trim().to_string())
        .filter(|value| !value.is_empty())
}
