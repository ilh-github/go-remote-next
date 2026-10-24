use clap::Parser;

#[derive(Clone, Debug, PartialEq, Eq)]
pub enum TransportMode {
    Stdio,
    Http,
}

impl std::str::FromStr for TransportMode {
    type Err = String;
    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "stdio" => Ok(TransportMode::Stdio),
            "http" => Ok(TransportMode::Http),
            other => Err(format!(
                "unknown transport mode '{}', expected 'stdio' or 'http'",
                other
            )),
        }
    }
}

impl std::fmt::Display for TransportMode {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            TransportMode::Stdio => write!(f, "stdio"),
            TransportMode::Http => write!(f, "http"),
        }
    }
}

#[derive(Parser, Clone)]
#[command(name = "quickdesk-mcp", about = "MCP bridge for QuickDesk remote desktop")]
pub struct AppConfig {
    /// Transport mode: "stdio" (default) or "http"
    #[arg(long, default_value = "stdio", env = "QUICKDESK_TRANSPORT")]
    pub transport: TransportMode,

    /// QuickDesk WebSocket server URL
    #[arg(long, default_value = "ws://127.0.0.1:9600")]
    pub ws_url: String,

    /// Authentication token for full-control access
    #[arg(long, env = "QUICKDESK_TOKEN")]
    pub token: Option<String>,

    /// Authentication token for read-only access (screenshot, status only — no input)
    #[arg(long, env = "QUICKDESK_READONLY_TOKEN")]
    pub readonly_token: Option<String>,

    /// Comma-separated list of allowed device IDs (restrict which devices AI can connect to)
    #[arg(long, env = "QUICKDESK_ALLOWED_DEVICES", value_delimiter = ',')]
    pub allowed_devices: Option<Vec<String>>,

    /// Maximum API requests per minute per client (0 = unlimited)
    #[arg(long, env = "QUICKDESK_RATE_LIMIT", default_value = "0")]
    pub rate_limit: i32,

    /// Session timeout in seconds (0 = no timeout)
    #[arg(long, env = "QUICKDESK_SESSION_TIMEOUT", default_value = "0")]
    pub session_timeout: i32,

    /// HTTP listen port (only used when transport=http)
    #[arg(long, default_value = "8080", env = "QUICKDESK_HTTP_PORT")]
    pub port: u16,

    /// HTTP listen host (only used when transport=http)
    #[arg(long, default_value = "127.0.0.1", env = "QUICKDESK_HTTP_HOST")]
    pub host: String,

    /// Allowed CORS origins, comma-separated (only used when transport=http, empty = no CORS headers)
    #[arg(long, env = "QUICKDESK_CORS_ORIGIN", value_delimiter = ',')]
    pub cors_origin: Option<Vec<String>>,

    /// Enable stateless HTTP mode (no session tracking). Default is stateful.
    #[arg(long, default_value = "false")]
    pub stateless: bool,

    /// Bearer token for the local business-workflow HTTP trigger. Empty disables the trigger API.
    #[arg(long, env = "QUICKDESK_TRIGGER_TOKEN")]
    pub trigger_token: Option<String>,

    /// Require an Owner-issued delegation grant for every workflow run. Off
    /// keeps the shared-token behaviour, where the token alone authorises any
    /// workflow the process already allows.
    #[arg(long, env = "QUICKDESK_DELEGATION_REQUIRED", default_value = "false")]
    pub delegation_required: bool,

    /// Path to a SQLite database shared by every workflow worker in this
    /// deployment. Empty keeps the local-file claim, which only coordinates
    /// processes on one machine: two workers on two hosts would each open their
    /// own claim file and both would drive the same run.
    #[arg(long, env = "QUICKDESK_WORKFLOW_CLAIM_DB")]
    pub workflow_claim_db: Option<String>,

    /// Comma-separated directories other machines submit workflow runs into.
    /// Empty keeps this worker watching only its own directory, so a run is
    /// executed by the process that accepted it. Watching a peer makes the run
    /// visible here; `workflow_claim_db` is what stops two machines that can
    /// both see it from driving it at once, so set that too.
    #[arg(long, env = "QUICKDESK_WORKFLOW_PEER_JOB_DIRS")]
    pub workflow_peer_job_dirs: Option<String>,
}

impl AppConfig {
    /// Get the authentication token to use for WebSocket connection.
    /// Prefers full-control token, falls back to readonly.
    pub fn auth_token(&self) -> Option<&str> {
        self.token.as_deref().or(self.readonly_token.as_deref())
    }
}
