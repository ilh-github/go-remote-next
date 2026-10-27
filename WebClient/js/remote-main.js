/**
 * remote-main.js - 远程桌面页面入口
 *
 * 从 URL 参数获取连接信息，建立远程桌面会话
 * 由 index.html 通过 window.open 打开
 */

import { Session, SessionState } from './protocol/session.js';
import { DataChannelHandler } from './protocol/datachannel-handler.js';
import { MouseHandler } from './input/mouse-handler.js';
import { KeyboardHandler } from './input/keyboard-handler.js';
import { TouchHandler } from './input/touch-handler.js';
import { ClipboardHandler } from './input/clipboard-handler.js';
import { watchForRelease } from './input/visibilityRelease.js';
import { selectDisplayTrack } from './videoLayout.js';
import { TransferProgress, reconcileTotal } from './transferProgress.js';
import { CursorRenderer } from './ui/cursor-renderer.js';
import { MouseButton } from './protocol/protobuf-messages.js';
import { addTrackOnce } from './protocol/media-stream.js';
import { VideoStats } from './ui/video-stats.js';
import { FloatingToolbar } from './ui/floating-toolbar.js';
import { TabCoordinator } from './ui/tab-coordinator.js';
import { AnnotationModel, ANNOTATION_TOOL_PEN } from './ui/annotationModel.js';
import { AnnotationLayer } from './ui/annotationLayer.js';
import { AnnotationChannel } from './ui/annotationChannel.js';
import { ControlGrantState } from './ui/control-grant.js';
import { SessionMediaState } from './sessionMediaState.js';
import { IceConfigFetcher } from './ice-config-fetcher.js';
import { t, applyI18n, getLocale, setLocale } from './i18n.js';
import { userApi } from './api/user-api.js';
import { uploadRecordingArtifact } from './recording-upload.js';
import { SkillBridgeClient } from './protocol/skill-bridge-client.js';
import { RemoteFileManager } from './files/remote-file-manager.js';
import { FileManagerPanel } from './files/file-manager-panel.js';
import { SessionChatModel } from './chat/session-chat.js';
import { ChatPanel } from './chat/chat-panel.js';
import { describeAccessCodeFailure } from './auth/access-code-errors.js';
import {
    ReconnectExhaustedError,
    isReconnectableAccessFailure,
    nonRetryableReconnectError,
    runReconnectAttempts,
} from './signaling/reconnect.js';

class RemoteDesktopApp {
    constructor() {
        this.session = null;
        this._lastConnectFailure = null;
        this.dcHandler = null;
        this.mouseHandler = null;
        this.keyboardHandler = null;
        this.touchHandler = null;
        this.clipboardHandler = null;
        this._isMobile = this._detectMobile();
        this._connectStartTime = 0;
        this._deviceId = '';
        this._connectionRecorded = false;
        this.cursorRenderer = null;
        this.videoStats = null;
        this.floatingToolbar = null;
        this.skillBridge = null;
        this.fileManager = null;
        this.fileManagerPanel = null;
        this._supportsFileManager = false;
        this.chatModel = null;
        this.chatPanel = null;
        this._supportsChat = false;
        this.tabCoordinator = null;
        // QD-315: annotations live beside the video, not inside it, so the
        // decoder can keep redrawing without wiping them.
        this.annotationModel = null;
        this.annotationLayer = null;
        this.annotationChannel = null;
        this._annotating = false;
        this._annotationTool = ANNOTATION_TOOL_PEN;
        this._annotationPointerId = null;
        this.controlGrant = null;
        this._recordingUploading = false;
        this._fileTransferAudit = new Map();
        // Per-transfer progress, kept separately from the audit state because the
        // remembered total is what stops the bar jumping backwards.
        this._transferProgress = new Map();
        this._connectionConfig = null;
        this._reconnectTask = null;
        this._manualDisconnect = false;

        // Everything in here belongs to the session currently connected and is
        // dropped on teardown: the cached audio track, the received stream map
        // and the selected stream id are all issued per session.
        this._media = new SessionMediaState();
        this._remoteWidth = 0;
        this._remoteHeight = 0;
        this._currentDisplayList = [];
        this._activeDisplayIndex = 0;
        this._supportsVirtualDisplay = false;
        // Screen-wall embedding: a wall tile must never capture input (the
        // operator is watching several machines at once, not driving one) and
        // reports its state to the embedding page instead of drawing overlays.
        this._wallMode = false;
        // A wall tile can be explicitly armed for input from the wall page; the
        // wall only ever arms one tile at a time.
        this._inputArmed = false;
        this._wallAudioEnabled = false;
    }

    /**
     * 检测是否为移动设备（手机/平板）
     * 仅靠 touch 能力检测不可靠，Windows 桌面也可能报 maxTouchPoints > 0
     * 需要结合 UA、屏幕尺寸和指针能力综合判断
     */
    _detectMobile() {
        const ua = navigator.userAgent || '';
        // Check UA for common mobile identifiers
        if (/Android|iPhone|iPad|iPod/i.test(ua)) {
            // iPad with desktop UA: navigator.platform or maxTouchPoints
            return true;
        }
        // iPad on iOS 13+ uses desktop UA but has touch
        if (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1) {
            return true;
        }
        // Primary pointer is coarse (finger) and no fine pointer (mouse) available
        if (window.matchMedia) {
            const coarse = window.matchMedia('(pointer: coarse)').matches;
            const fine = window.matchMedia('(any-pointer: fine)').matches;
            if (coarse && !fine) {
                return true;
            }
        }
        // Small screen with touch capability is likely a phone
        if (navigator.maxTouchPoints > 0 && window.innerWidth <= 768) {
            return true;
        }
        return false;
    }

    async init() {
        this._initRemoteLangSelector();
        applyI18n();

        const params = new URLSearchParams(window.location.search);
        this._wallMode = params.get('wall') === '1';
        if (this._wallMode) {
            this._wallNotify('connecting');
            window.addEventListener('message', (event) => this._onWallMessage(event));
        }
        const serverUrl = params.get('server') || 'ws://localhost:8000';
        const deviceId = params.get('device');
        userApi.setBaseUrl(serverUrl);
        userApi.scheduleProactiveRefresh();
        // §2.18: URL carries a one-shot signal_token (key "st"), NOT the
        // plaintext access_code. The access_code is handed off via
        // sessionStorage (see src/utils/remoteLauncher.js) — same-origin,
        // never on the wire / browser history. SPAKE2 still needs the
        // access_code as the shared secret so the host can verify the
        // peer end-to-end (§2.6).
        const signalTokenFromUrl = params.get('st');
        const handoff = this._readHandoff(signalTokenFromUrl);
        let accessCode = handoff && handoff.access_code;
        const preferredVideoCodec = params.get('codec') || '';
        userApi.setBaseUrl(serverUrl);

        let signalToken = signalTokenFromUrl;
        let accessFailure = null;
        if (handoff && accessCode) {
            this._writeCurrentAccessCode(deviceId, accessCode);
        } else {
            accessCode = this._readCurrentAccessCode(deviceId);
            // On refresh, the URL's `st` is usually already consumed.
            // Re-verify the saved access_code to get a fresh one-shot
            // signal_token. As long as the access_code remains valid,
            // refresh can reconnect without asking the user to re-enter it.
            const issued = await this._issueFreshSignalToken(deviceId, accessCode, params);
            signalToken = issued.token;
            accessFailure = issued.failure;
        }

        if (!deviceId || !signalToken || !accessCode) {
            const message = deviceId && accessCode && accessFailure
                ? t(accessFailure.messageKey)
                : t('status.missingParams');
            this._log(message, 'error');
            this._setConnectionState('failed', message);
            return;
        }

        document.title = `QuickDesk - ${deviceId}`;
        document.getElementById('connDevice').textContent = deviceId;

        const remoteContainer = document.getElementById('remoteContainer');
        // A wall tile gets no toolbar at all: its chrome lives in the wall page.
        if (remoteContainer && !this._wallMode) {
            this.floatingToolbar = new FloatingToolbar(remoteContainer);
            this.floatingToolbar.addEventListener('action', (e) => this._handleToolbarAction(e.detail));
            this.floatingToolbar.addEventListener('settingChange', (e) => this._handleSettingChange(e.detail));
            this.floatingToolbar.setVisible(false);
        }

        if (this._isMobile) {
            this._setupMobileToolbar();
        }

        this._initLogDrawer();

        const statsOverlay = document.getElementById('statsOverlay');
        if (statsOverlay) {
            this.videoStats = new VideoStats(statsOverlay, null);
        }

        this._log(t('log.fetchingIce'));
        const fetcher = new IceConfigFetcher(serverUrl);
        const iceServers = await fetcher.getIceServers();
        this._log(t('log.iceServers', { count: iceServers.length }));

        await this._connect(serverUrl, deviceId, accessCode, signalToken, iceServers, preferredVideoCodec);
    }

    /**
     * Read the one-shot handoff envelope written by the launcher. Clears
     * it immediately so browser refresh won't accidentally reuse a
     * consumed signal_token (they're single-use server-side anyway, but
     * keep storage clean).
     */
    _readHandoff(signalToken) {
        if (!signalToken) return null;
        const key = `quickdesk_remote_handoff__${signalToken}`;
        try {
            const raw = sessionStorage.getItem(key);
            if (!raw) return null;
            const obj = JSON.parse(raw);
            try {
                sessionStorage.removeItem(key);
            } catch {}
            return obj;
        } catch {
            return null;
        }
    }

    _currentSessionKey(deviceId) {
        return `quickdesk_remote_session__${deviceId}`;
    }

    _readCurrentAccessCode(deviceId) {
        if (!deviceId) return '';
        try {
            const raw = sessionStorage.getItem(this._currentSessionKey(deviceId));
            if (!raw) return '';
            const obj = JSON.parse(raw);
            return obj && obj.device_id === deviceId ? (obj.access_code || '') : '';
        } catch {
            return '';
        }
    }

    _writeCurrentAccessCode(deviceId, accessCode) {
        if (!deviceId || !accessCode) return;
        try {
            sessionStorage.setItem(this._currentSessionKey(deviceId), JSON.stringify({
                device_id: deviceId,
                access_code: accessCode,
                updated_at: Date.now(),
            }));
        } catch {}
    }

    async _issueFreshSignalToken(deviceId, accessCode, params) {
        if (!deviceId || !accessCode) return { token: '', failure: null };
        try {
            const v = await userApi.verifyAccessCode(deviceId, accessCode);
            if (!v || !v.ok || !v.data || !v.data.signal_token) {
                const failure = describeAccessCodeFailure(v);
                this._log(
                    `access-code verify rejected: ${failure ? failure.code : 'unknown'}`,
                    'warning',
                );
                return { token: '', failure };
            }
            const signalToken = v.data.signal_token;
            try {
                params.set('st', signalToken);
                const newUrl = window.location.pathname + (params.toString() ? '?' + params.toString() : '');
                window.history.replaceState({}, document.title, newUrl);
            } catch {}
            return { token: signalToken, failure: null };
        } catch (e) {
            this._log(`verifyAccessCode error: ${e && e.message ? e.message : e}`, 'warning');
            return {
                token: '',
                failure: { code: 'ACCESS_VERIFY_FAILED', messageKey: 'status.accessVerifyFailed' },
            };
        }
    }

    _initRemoteLangSelector() {
        const select = document.getElementById('remoteLangSelect');
        if (!select) return;
        select.value = getLocale();
        select.addEventListener('change', () => {
            setLocale(select.value);
            applyI18n();
            if (this.floatingToolbar) {
                this.floatingToolbar.refreshI18n();
            }
            this.fileManagerPanel?.refreshI18n();
            this.chatPanel?.refreshI18n();
        });
    }

    async _connect(serverUrl, deviceId, accessCode, signalToken, iceServers, preferredVideoCodec) {
        this._log(t('log.connectingTo', { deviceId }));
        this._connectionConfig = {
            serverUrl,
            deviceId,
            accessCode,
            iceServers,
            preferredVideoCodec,
        };
        this._deviceId = deviceId;
        this._connectStartTime = Date.now();
        this._connectionRecorded = false;
        // Test hook only: shortens how long a silent link may stay silent. Unset
        // in production, where the transport's own default applies.
        const stalenessRaw = Number(new URLSearchParams(window.location.search).get('staleness'));
        const stalenessTimeoutMs = Number.isFinite(stalenessRaw) && stalenessRaw > 0
            ? stalenessRaw
            : undefined;
        this._setupTabCoordinator(deviceId);
        this._setupControlGrant();
        this._setupNetworkStateHandlers();
        this._setupVisibilityRelease();

        // Set up userApi base URL for connection recording
        userApi.setBaseUrl(serverUrl);
        if (preferredVideoCodec) {
            this._log(t('log.preferredCodec', { codec: preferredVideoCodec }));
        }

        try {
            const session = new Session({
                signalingUrl: serverUrl,
                iceServers,
                preferredVideoCodec,
                // Lets a test shorten the staleness window instead of waiting
                // 90 real seconds for the default to fire.
                stalenessTimeoutMs: stalenessTimeoutMs,
            });
            this.session = session;

            this.dcHandler = new DataChannelHandler();
            this._setupSkillBridge();

            session.addEventListener('stateChange', (e) => {
                if (this.session !== session) return;
                this._onSessionStateChange(e.detail);
            });

            session.addEventListener('log', (e) => {
                this._log(e.detail.message, e.detail.level);
            });

            session.addEventListener('track', (e) => {
                if (this.session !== session) return;
                this._onTrack(e.detail);
            });

            session.addEventListener('datachannel', (e) => {
                if (this.session !== session) return;
                this.dcHandler.handleDataChannel(e.detail.channel);
                if (!this.dcHandler._pc && session.pc) {
                    this.dcHandler.setPeerConnection(session.pc);
                }
            });

            session.addEventListener('signalingDisconnected', (e) => {
                if (this.session !== session) return;
                this._startReconnect(e.detail);
            });

            this.dcHandler.addEventListener('cursorShape', (e) => {
                if (this.cursorRenderer) this.cursorRenderer.updateCursor(e.detail);
            });

            this.dcHandler.addEventListener('videoLayout', (e) => {
                this._onVideoLayout(e.detail);
            });

            this.dcHandler.addEventListener('controlReady', () => {
                this._log(t('log.controlReady'));
                this._sendInitialConfig();
            });

            this.dcHandler.addEventListener('eventReady', () => {
                this._log(t('log.eventReady'));
            });

            this.dcHandler.addEventListener('capabilities', (e) => {
                this._log(t('log.hostCaps', { caps: e.detail.capabilities || '(empty)' }));
            });

            this.dcHandler.addEventListener('hostCapabilities', (e) => {
                const { supportsSendAttentionSequence, supportsLockWorkstation, supportsFileTransfer, supportsPrivacyScreen, supportsVirtualDisplay } = e.detail;
                this._log(t('log.negotiatedCaps', { sas: supportsSendAttentionSequence, lock: supportsLockWorkstation, ft: supportsFileTransfer }));
                this._supportsVirtualDisplay = supportsVirtualDisplay;
                if (this.floatingToolbar) {
                    this.floatingToolbar.setActionSupport(
                        supportsSendAttentionSequence, supportsLockWorkstation, supportsFileTransfer, supportsPrivacyScreen, supportsVirtualDisplay);
                }
                if (supportsVirtualDisplay) {
                    this._sendVirtualDisplayCommand({ action: 'query' });
                }
            });

            this.dcHandler.addEventListener('extensionMessage', (e) => {
                this._onExtensionMessage(e.detail);
            });

            this.dcHandler.addEventListener('fileTransferStarted', (e) => {
                this._fileTransferAudit.set(e.detail.transferId, {
                    filename: e.detail.filename,
                    bytesTransferred: 0,
                    totalBytes: Number(e.detail.totalBytes) || 0,
                });
                this._addTransferItem(e.detail.transferId, e.detail.filename, e.detail.totalBytes);
                // The total announced at the start, including zero: a zero here
                // says the file is empty, which is complete rather than 0%.
                const progress = new TransferProgress();
                progress.announceTotal(e.detail.totalBytes);
                this._transferProgress.set(e.detail.transferId, progress);
            });

            this.dcHandler.addEventListener('fileTransferProgress', (e) => {
                const state = this._fileTransferAudit.get(e.detail.transferId);
                if (state) {
                    state.bytesTransferred = Number(e.detail.bytesSent) || 0;
                    // A zero or missing total on a later event must not undo the
                    // one announced at the start, or the audit row and the bar
                    // disagree about how big this transfer is.
                    state.totalBytes = reconcileTotal(state.totalBytes, e.detail.totalBytes);
                }
                // The bar keeps its own remembered total for the same reason.
                const progress = this._transferProgress.get(e.detail.transferId)
                    || new TransferProgress();
                progress.update(e.detail.bytesSent, e.detail.totalBytes);
                this._transferProgress.set(e.detail.transferId, progress);
                this._updateTransferProgress(e.detail.transferId, progress.percent());
            });

            this.dcHandler.addEventListener('fileTransferComplete', (e) => {
                this._recordFileTransferAudit(
                    'upload', 'completed', e.detail.transferId, e.detail.filename);
                this._updateTransferStatus(e.detail.transferId, 'complete');
                this._log(t('log.uploadComplete', { filename: e.detail.filename }));
            });

            this.dcHandler.addEventListener('fileTransferError', (e) => {
                this._recordFileTransferAudit(
                    'upload', 'failed', e.detail.transferId, '', e.detail.errorMessage);
                this._updateTransferStatus(e.detail.transferId, 'error', e.detail.errorMessage);
                this._log(t('log.uploadFailed', { error: e.detail.errorMessage }));
            });

            this.dcHandler.addEventListener('fileDownloadStarted', (e) => {
                this._fileTransferAudit.set(e.detail.transferId, {
                    filename: e.detail.filename,
                    bytesTransferred: 0,
                    totalBytes: Number(e.detail.totalBytes) || 0,
                });
                this._addTransferItem(e.detail.transferId, e.detail.filename, e.detail.totalBytes, 'download');
                const progress = new TransferProgress();
                progress.announceTotal(e.detail.totalBytes);
                this._transferProgress.set(e.detail.transferId, progress);
                this._log(t('log.downloadStarted', { filename: e.detail.filename }));
            });

            this.dcHandler.addEventListener('fileDownloadProgress', (e) => {
                const state = this._fileTransferAudit.get(e.detail.transferId);
                if (state) {
                    state.bytesTransferred = Number(e.detail.bytesReceived) || 0;
                    // Same as the upload path: a zero or missing total must not
                    // undo the one announced at the start.
                    state.totalBytes = reconcileTotal(state.totalBytes, e.detail.totalBytes);
                }
                const progress = this._transferProgress.get(e.detail.transferId)
                    || new TransferProgress();
                progress.update(e.detail.bytesReceived, e.detail.totalBytes);
                this._transferProgress.set(e.detail.transferId, progress);
                this._updateTransferProgress(e.detail.transferId, progress.percent());
            });

            this.dcHandler.addEventListener('fileDownloadComplete', (e) => {
                this._recordFileTransferAudit(
                    'download', 'completed', e.detail.transferId, e.detail.filename);
                this._updateTransferStatus(e.detail.transferId, 'complete');
                this._log(t('log.downloadComplete', { filename: e.detail.filename }));
                if (e.detail.blob && !e.detail.streaming) {
                    this._triggerBrowserDownload(e.detail.blob, e.detail.filename);
                }
            });

            this.dcHandler.addEventListener('fileDownloadError', (e) => {
                this._recordFileTransferAudit(
                    'download', 'failed', e.detail.transferId, '', e.detail.errorMessage);
                this._updateTransferStatus(e.detail.transferId, 'error', e.detail.errorMessage);
                this._log(t('log.downloadFailed', { error: e.detail.errorMessage }));
            });

            await this.session.connect(deviceId, accessCode, signalToken);
            this._lastConnectFailure = null;
            return true;

        } catch (error) {
            this._log(t('log.connectFailed', { error: error.message }), 'error');
            this._setConnectionState('failed', error.message);
            // Kept for the caller to rethrow: _connect answers false rather
            // than propagating, so a non-retryable reason would otherwise be
            // flattened into a generic failure the retry loop treats as
            // retryable.
            this._lastConnectFailure = error;
            return false;
        }
    }

    _startReconnect(detail = {}) {
        if (this._manualDisconnect || this._reconnectTask) return
        this._log(t('log.reconnecting', {
            code: detail.code || '',
            reason: detail.reason || '',
        }), 'warning')

        this._reconnectTask = runReconnectAttempts(
            () => this._attemptReconnect(),
            {
                onAttempt: ({ attempt, total }) => {
                    const text = t('status.reconnecting', { attempt, total })
                    this._setConnectionState('reconnecting', text)
                    this._log(t('log.reconnectAttempt', { attempt, total }))
                },
                onFailure: ({ attempt, total, error, retryable }) => {
                    this._log(t('log.reconnectFailed', {
                        attempt,
                        total,
                        error: error?.message || String(error),
                    }), retryable ? 'warning' : 'error')
                },
            },
        ).catch((error) => {
            const accessMessageKey = error instanceof ReconnectExhaustedError
                && error.retryable === false
                ? error.cause?.messageKey
                : ''
            this._setConnectionState(
                'failed',
                t(accessMessageKey || 'status.reconnectExhausted'),
            )
            this._recordConnectionResult(
                'failed',
                error?.code || (error?.retryable === false
                    ? 'reconnect_rejected'
                    : 'reconnect_exhausted'),
            )
        }).finally(() => {
            this._reconnectTask = null
        })
    }

    async _attemptReconnect() {
        const config = this._connectionConfig
        if (!config) throw new Error('missing connection config for reconnect')

        const issued = await this._issueFreshSignalToken(
            config.deviceId,
            config.accessCode,
            new URLSearchParams(window.location.search),
        )
        if (!issued.token) {
            const failure = issued.failure || {
                code: 'ACCESS_VERIFY_FAILED',
                messageKey: 'status.accessVerifyFailed',
            }
            if (!isReconnectableAccessFailure(failure)) {
                throw nonRetryableReconnectError(failure.messageKey, failure)
            }
            const error = new Error(failure.messageKey)
            error.code = failure.code
            error.messageKey = failure.messageKey
            throw error
        }

        this._resetSessionForReconnect()
        const connected = await this._connect(
            config.serverUrl,
            config.deviceId,
            config.accessCode,
            issued.token,
            config.iceServers,
            config.preferredVideoCodec,
        )
        if (!connected) {
            // _connect swallows the error and answers false, so the reason it
            // failed would otherwise be lost here - including a non-retryable
            // one, which must stop the run instead of spending the whole budget
            // rediscovering the same rejection four times.
            const failure = this._lastConnectFailure
            this._lastConnectFailure = null
            if (failure && failure.retryable === false) {
                // Carry the server's code so the status pill can name the
                // rejection rather than show the generic exhausted message. An
                // operator told "the network kept failing" about a credential
                // the server refused will wait for a link that was never broken.
                const rejected = new Error(failure.message)
                rejected.retryable = false
                rejected.code = failure.code
                rejected.messageKey = `status.reason.${failure.code}`
                throw rejected
            }
            throw new Error('reconnect attempt failed')
        }
    }

    _resetSessionForReconnect() {
        this._cleanupInputHandlers()
        // The captured resolution belongs to the session that just ended. The new
        // one may attach to a different monitor, and a click mapped through the
        // old resolution lands at a fraction of where the operator aimed.
        this._remoteWidth = 0
        this._remoteHeight = 0
        // The media state belongs to the session that just ended. The audio track
        // is the one that bites: it is cached separately and merged into every
        // combined stream built afterwards, so a still-live track from the old
        // session would play under the new session's video.
        this._media.reset()
        const session = this.session
        this.session = null
        if (session) {
            try { session.disconnect() } catch { /* already closed */ }
        }
        this.dcHandler = null
    }

    _onSessionStateChange(detail) {
        const { oldState, newState, reason } = detail;
        this._log(t('log.stateChange', { from: oldState, to: newState }));

        switch (newState) {
            case SessionState.CONNECTED:
                this._setConnectionState('connected', t('status.connected'));
                this._recordConnectionResult('success');
                if (this.floatingToolbar) this.floatingToolbar.setVisible(true);
                if (this.videoStats) this.videoStats.setPeerConnection(this.session.pc);
                this._startConnBarStats();
                if (window.opener && !window.opener.closed) {
                    try {
                        window.opener.postMessage({
                            type: 'quickdesk-connected',
                            deviceId: this.session.deviceId,
                            serverUrl: this.session.signalingUrl,
                        }, '*');
                    } catch (e) { /* cross-origin */ }
                }
                break;
            case SessionState.FAILED: {
                // §2.15 surfaces precise reason codes from signaling:
                //   HOST_OFFLINE       — peer never came online
                //   PEER_DISCONNECTED  — peer dropped mid-session
                //   TOKEN_INVALID/AUTH_INVALID — signal_token rejected
                // Fall back to the generic "connection failed" message
                // for WebRTC/ICE-level failures that have no reason
                // code (oniceconnectionstatechange → failed etc.).
                const reasonKey = reason ? `status.reason.${reason}` : '';
                const reasonText = reasonKey ? t(reasonKey) : '';
                const failMsg = (reasonText && reasonText !== reasonKey)
                    ? reasonText
                    : t('status.failed');
                this._setConnectionState('failed', failMsg);
                this._recordConnectionResult('failed', reason || 'Connection failed');
                this.tabCoordinator?.releaseAll();
                break;
            }
            case SessionState.CLOSED:
                this._setConnectionState('failed', t('status.closed'));
                this._recordConnectionResult('success');
                this.tabCoordinator?.releaseAll();
                break;
        }
    }

    // QD-315: one canvas over the video, one model behind it, one channel to the
    // other viewers of this device. All three are created once per session.
    _setupAnnotations() {
        if (this.annotationLayer) return;
        const container = document.getElementById('videoContainer');
        const video = document.getElementById('remoteVideo');
        if (!container || !video) return;

        const canvas = document.createElement('canvas');
        canvas.id = 'annotationCanvas';
        canvas.className = 'annotation-canvas';
        container.appendChild(canvas);

        this.annotationLayer = new AnnotationLayer(canvas);
        this.annotationModel = new AnnotationModel();
        this.annotationChannel = new AnnotationChannel({
            deviceId: this._deviceId || '',
            // The viewer identity is what makes "clear mine" possible, and the
            // tab id keeps two tabs of the same account from colliding.
            viewerId: this.tabCoordinator?.tabId || `viewer-${Math.random().toString(36).slice(2)}`,
            onStroke: (stroke) => {
                // The channel outlives a teardown by a message in flight, so
                // every handler checks the stack is still there.
                if (this.annotationModel?.add(stroke)) this._renderAnnotations();
            },
            onRemoved: (id) => {
                if (this.annotationModel?.remove(id)) this._renderAnnotations();
            },
            onAuthorCleared: (authorId) => {
                if ((this.annotationModel?.clearAuthor(authorId) ?? 0) > 0) this._renderAnnotations();
            },
        });
        this.annotationChannel.start();

        // Pointer events go to the canvas only while annotating, so the normal
        // control path is untouched when the feature is off.
        canvas.addEventListener('pointerdown', (e) => this._onAnnotationPointerDown(e));
        canvas.addEventListener('pointermove', (e) => this._onAnnotationPointerMove(e));
        canvas.addEventListener('pointerup', (e) => this._onAnnotationPointerUp(e));
        canvas.addEventListener('pointercancel', (e) => this._onAnnotationPointerUp(e));
        globalThis.addEventListener('resize', () => this._renderAnnotations());
    }

    _renderAnnotations() {
        if (!this.annotationLayer || !this.annotationModel) return;
        this.annotationLayer.render(this.annotationModel.strokes);
    }

    _teardownAnnotations() {
        if (this.annotationChannel) {
            this.annotationChannel.destroy();
            this.annotationChannel = null;
        }
        this.annotationModel = null;
        this.annotationLayer = null;
        this._annotating = false;
        this._annotationPointerId = null;
        const canvas = document.getElementById('annotationCanvas');
        canvas?.remove();
    }

    _toggleAnnotations() {
        this._annotating = !this._annotating;
        const canvas = document.getElementById('annotationCanvas');
        if (canvas) {
            // `pointer-events` is what decides whether a drag reaches the canvas
            // or falls through to the video and becomes remote input.
            canvas.classList.toggle('active', this._annotating);
        }
        if (this.floatingToolbar) this.floatingToolbar.setAnnotating(this._annotating);
        if (!this._annotating) {
            this._annotationPointerId = null;
            this.annotationModel.end();
            this._renderAnnotations();
        }
        this._log(t(this._annotating ? 'log.annotateOn' : 'log.annotateOff'));
    }

    _annotationPoint(event) {
        const canvas = document.getElementById('annotationCanvas');
        if (!canvas) return null;
        const rect = canvas.getBoundingClientRect();
        const local = this.annotationLayer.toRemote(
            event.clientX - rect.left,
            event.clientY - rect.top,
        );
        return local;
    }

    _onAnnotationPointerDown(event) {
        if (!this._annotating || this._annotationPointerId !== null) return;
        this._annotationPointerId = event.pointerId;
        const point = this._annotationPoint(event);
        if (!point) return;
        const id = this.annotationChannel.nextStrokeId();
        this.annotationModel.begin({
            id,
            tool: this._annotationTool,
            authorId: this.annotationChannel.viewerId,
            x: point.x,
            y: point.y,
        });
        this._renderAnnotations();
        // Capturing the pointer is what keeps a stroke alive when the cursor
        // leaves the canvas mid-drag.
        event.currentTarget.setPointerCapture?.(event.pointerId);
    }

    _onAnnotationPointerMove(event) {
        if (!this._annotating || event.pointerId !== this._annotationPointerId) return;
        const point = this._annotationPoint(event);
        if (!point) return;
        this.annotationModel.extend(point.x, point.y);
        this._renderAnnotations();
    }

    _onAnnotationPointerUp(event) {
        if (event.pointerId !== this._annotationPointerId) return;
        this._annotationPointerId = null;
        const stroke = this.annotationModel.end();
        if (!stroke) return;
        // Only a finished stroke is shared. Publishing on every move would put a
        // message on the channel per pointer event for a line nobody has
        // finished drawing yet.
        this.annotationChannel.publishAdd(stroke);
        this._renderAnnotations();
    }

    /** Clears only this viewer's strokes, leaving everybody else's intact. */
    _clearMyAnnotations() {
        const removed = this.annotationModel.clearAuthor(this.annotationChannel.viewerId);
        this.annotationChannel.publishClearAuthor();
        this._renderAnnotations();
        this._log(t('log.annotateCleared', { count: removed }));
    }

    _setupTabCoordinator(deviceId) {
        if (this.tabCoordinator || !deviceId) return;

        this.tabCoordinator = new TabCoordinator({ deviceId });
        this.tabCoordinator.addEventListener('revoked', (event) => {
            this._onTabResourceRevoked(event.detail);
        });
        this.tabCoordinator.start();

        window.addEventListener('beforeunload', () => {
            this.tabCoordinator?.releaseAll();
        }, { once: true });
    }

    // A process that loses the network gets no socket event to react to: the
    // connection does not close, it just stops working, and the only signal is
    // the browser's own offline state. Without this the client keeps showing
    // "Connected" over a link that is gone, and the operator finds out by
    // typing into a machine that is not receiving anything.
    _setupNetworkStateHandlers() {
        window.addEventListener('offline', () => {
            if (this._manualDisconnect) return;
            this._log(t('log.networkOffline'), 'warning');
            this._setConnectionState('reconnecting', t('log.networkOffline'));
            // Close the socket so the reconnect path has something to react to.
            // Leaving it open means the client waits for a close that will not
            // arrive until the network is already back.
            try { this.session?.disconnect(); } catch { /* already closing */ }
        });

        window.addEventListener('online', () => {
            if (this._manualDisconnect || this._reconnectTask) return;
            // Coming back is the trigger, but only if there is something to
            // restore: a page that was never connected must not start a
            // reconnect it has no config for.
            if (!this._connectionConfig) return;
            this._log(t('log.networkOnline'), 'info');
            this._startReconnect({ code: 'network_restored', reason: 'online' });
        });
    }

    // The host keeps its own record of which keys and buttons are down, and none
    // of the ways an operator stops paying attention produce a blur on the
    // element that receives keys: switching tabs, switching applications,
    // minimising and locking the screen all leave the host holding whatever was
    // down. The Qt client guards all four; this is the web client's equivalent.
    _setupVisibilityRelease() {
        if (this._detachVisibilityRelease) return;
        this._detachVisibilityRelease = watchForRelease(window, () => {
            // Releasing the handlers already sends releaseAllInput, which is the
            // part that reaches the host.
            this._cleanupInputHandlers();
            this._log(t('log.inputReleased'), 'info');
        });
    }

    // QD-314: the server's answer on who may drive this machine. It is wired to
    // the signaling socket rather than the DataChannel because that is the only
    // place the answer can come from - the input itself is peer to peer, so no
    // middlebox can drop it on this client's behalf.
    _setupControlGrant() {
        if (this.controlGrant) return;
        this.controlGrant = new ControlGrantState();
        this.controlGrant.addEventListener('revoked', (event) => {
            this._onControlRevoked(event.detail);
        });
        this.session?.addEventListener('jsonMessage', (event) => {
            this.controlGrant.handleMessage(event.detail);
        });
    }

    // Losing control means this tab stops sending input, exactly as it already
    // does when another same-origin tab takes over.
    _onControlRevoked(detail) {
        if (!detail) return;
        this._cleanupInputHandlers();
        this._log(t('log.controlTakenOver', { holder: detail.holder || '' }), 'warning');
        this._showTabControlNotice();
    }

    _setupSkillBridge() {
        if (this.skillBridge) return;
        this.skillBridge = new SkillBridgeClient(
            (data) => this.dcHandler?.sendExtensionMessage('qd-skill-bridge', data),
            { idPrefix: `web-${this._deviceId || 'device'}` },
        );
        this.fileManager = new RemoteFileManager(this.skillBridge);
        this.chatModel = new SessionChatModel({
            send: message => this.skillBridge.sendRaw(message),
        });
        this.skillBridge.addEventListener('message', event => {
            this.chatModel.handleMessage(event.detail);
        });
        this.chatModel.addEventListener('messageAdded', event => {
            if (!this._wallMode && this.chatPanel?.isOpen) {
                this.chatModel.markRead();
            }
            const message = event.detail;
            this._log(t('log.chatMessage', {
                sender: message.outgoing ? t('chat.you') : t('chat.host'),
            }));
        });
        this.chatModel.addEventListener('unreadChanged', event => {
            this.floatingToolbar?.updateChatUnread(event.detail.count);
        });
        this.chatModel.addEventListener('error', event => {
            this._log(event.detail.message, 'warning');
        });

        const container = document.getElementById('remoteContainer');
        if (container && !this._wallMode) {
            this.chatPanel = new ChatPanel(container, this.chatModel);
        }
    }

    _onTabResourceRevoked(detail) {
        if (!detail) return;

        if (detail.resource === 'control') {
            this._cleanupInputHandlers();
            this._log(t('log.tabControlRevoked'), 'warning');
            this._showTabControlNotice();
            return;
        }

        if (detail.resource === 'audio') {
            if (this.floatingToolbar) {
                this.floatingToolbar.settings.audioEnabled = false;
                this.floatingToolbar.refreshI18n();
            }
            this.dcHandler?.sendAudioControl({ enable: false });
            this._log(t('log.tabAudioRevoked'), 'warning');
        }
    }

    _showTabControlNotice() {
        if (this._wallMode) return;
        const overlay = document.getElementById('unmuteOverlay');
        if (!overlay) return;
        overlay.textContent = t('overlay.controlTaken');
        overlay.style.display = 'block';
        overlay.onclick = () => {
            overlay.onclick = null;
            overlay.style.display = 'none';
            this._setupInputHandlers();
        };
    }

    _recordConnectionResult(status, errorMsg) {
        if (this._connectionRecorded && status === 'success') return; // Don't double-record success
        if (!userApi.isLoggedIn() || !this._deviceId) return;
        this._connectionRecorded = true;

        const duration = Math.round((Date.now() - this._connectStartTime) / 1000);
        userApi.recordConnection(this._deviceId, duration, status, errorMsg || '').catch(() => {});
    }

    _recordFileTransferAudit(direction, status, transferId, filename, errorMessage = '') {
        const state = this._fileTransferAudit.get(transferId);
        this._fileTransferAudit.delete(transferId);
        if (!userApi.isLoggedIn() || !this._deviceId || !transferId) return;

        const bytesTransferred = Math.max(0, Math.trunc(Number(state?.bytesTransferred) || 0));
        const totalBytes = Math.max(0, Math.trunc(Number(state?.totalBytes) || 0));
        userApi.recordFileTransferAudit({
            deviceId: this._deviceId,
            transferId,
            direction,
            status,
            filename: filename || state?.filename || '',
            bytesTransferred,
            totalBytes,
            error: errorMessage || '',
        }).catch(() => {});
    }

    _onTrack(detail) {
        const { track, streams } = detail;
        const streamId = (streams && streams.length > 0) ? streams[0].id : '';
        this._log(t('log.trackReceived', { kind: track.kind, stream: streamId }));

        if (track.kind === 'video' && streams && streams.length > 0) {
            this._media.addStream(streams[0].id, streams[0]);
        }

        if (track.kind === 'audio') {
            this._log(t('log.audioTrack', { enabled: track.enabled, muted: track.muted, state: track.readyState }));
            this._media.setAudioTrack(track);
            const video = document.getElementById('remoteVideo');
            if (video?.srcObject && addTrackOnce(video.srcObject, track)) {
                this._log(t('log.audioMerged'));
            }
            return;
        }

        if (track.kind === 'video' && this._media.selectedStreamId && streamId !== this._media.selectedStreamId) {
            this._log(t('log.ignoreStream', { stream: streamId }));
            return;
        }

        if (track.kind === 'video') {
            this._setupVideoPlayback(track, streams);
        }
    }

    _setupVideoPlayback(track, streams) {
        const video = document.getElementById('remoteVideo');
        if (!video) return;

        const combinedStream = new MediaStream();
        const audioTrack = this._media.usableAudioTrack();
        if (audioTrack) {
            if (addTrackOnce(combinedStream, audioTrack)) {
                this._log(t('log.audioMerged'));
            }
        }
        combinedStream.addTrack(track);
        video.srcObject = combinedStream;
        if (this._wallMode) {
            video.muted = !this._wallAudioEnabled;
        }

        video.play().catch(e => {
            this._log(t('log.autoplayFailed', { error: e.message }), 'warning');
            video.muted = true;
            video.play().then(() => {
                this._showClickToUnmute(video);
            }).catch(() => {
                this._showClickToPlay(video);
            });
        });

        video.onloadedmetadata = () => {
            this._log(t('log.videoResolution', { width: video.videoWidth, height: video.videoHeight }));
            document.getElementById('noVideo')?.style.setProperty('display', 'none');
            if (this.floatingToolbar) {
                this.floatingToolbar.setRemoteResolution(video.videoWidth, video.videoHeight);
            }
            // QD-315: strokes are stored in remote coordinates, so a change of
            // captured resolution has to reach the layer too - otherwise every
            // existing stroke is scaled by the old factor and lands in the wrong
            // place on the desktop.
            if (this.annotationLayer) {
                this.annotationLayer.setRemoteResolution(video.videoWidth, video.videoHeight);
                this.annotationLayer.resize();
                this._renderAnnotations();
            }
            this._wallNotify('live', `${video.videoWidth}x${video.videoHeight}`);
            // Input stays off in wall mode: a tile is a monitor, not a console.
            if (!this._wallMode) {
                this._setupInputHandlers();
            } else if (this._inputArmed) {
                // The wall asked for input before the media was ready.
                this._setupInputHandlers();
            }
        };
    }

    _showClickToUnmute(video) {
        // A wall tile is muted by design; an overlay per tile would just be noise.
        if (this._wallMode) return;
        const overlay = document.getElementById('unmuteOverlay');
        if (!overlay) return;
        overlay.textContent = t('overlay.clickToUnmute');
        overlay.style.display = 'block';
        const handler = () => {
            video.muted = false;
            overlay.style.display = 'none';
            overlay.removeEventListener('click', handler);
            document.removeEventListener('click', handler);
        };
        overlay.addEventListener('click', handler);
        document.addEventListener('click', handler, { once: true });
    }

    _showClickToPlay(video) {
        if (this._wallMode) return;
        const overlay = document.getElementById('unmuteOverlay');
        if (!overlay) return;
        overlay.textContent = t('overlay.clickToPlay');
        overlay.style.display = 'block';
        const handler = () => {
            video.muted = false;
            video.play().catch(() => {});
            overlay.style.display = 'none';
            overlay.removeEventListener('click', handler);
            document.removeEventListener('click', handler);
        };
        overlay.addEventListener('click', handler);
        document.addEventListener('click', handler, { once: true });
    }

    _onVideoLayout(layout) {
        if (!layout.videoTracks || layout.videoTracks.length === 0) return;

        // The host prepends legacy capture/full-desktop tracks without a media
        // stream id. Only per-display tracks can be selected or rendered.
        const displays = layout.videoTracks.filter(track => track.mediaStreamId);
        this._log(t('log.videoLayout', { count: displays.length }));

        this._currentDisplayList = displays;

        // An empty layout is a state, not a non-event: the display strip has to
        // stop showing monitors the host no longer reports. Returning before
        // this point left the previous list standing, and clicking one of its
        // entries selected a stream id the host will never send.
        if (this.floatingToolbar) {
            this.floatingToolbar.updateDisplayList(displays, -1);
        }

        let selectedTrack = null;
        let selectedIndex = 0;

        const selection = selectDisplayTrack(displays, {
            selectedStreamId: this._media.selectedStreamId,
            primaryScreenId: layout.primaryScreenId,
        });
        selectedTrack = selection.track;
        selectedIndex = selection.index;

        // Nothing selectable: the strip has already been told, and there is no
        // monitor to point the resolution or the video element at.
        if (!selectedTrack) return;

        this._log(t('log.selectMonitor', { width: selectedTrack.width, height: selectedTrack.height }));
        this._remoteWidth = selectedTrack.width;
        this._remoteHeight = selectedTrack.height;
        // QD-315: strokes are stored in remote coordinates, so the layer needs
        // the captured resolution before it can place any of them.
        if (this.annotationLayer) {
            this.annotationLayer.setRemoteResolution(selectedTrack.width, selectedTrack.height);
            this._renderAnnotations();
        }
        this._activeDisplayIndex = selectedIndex;

        const resEl = document.getElementById('connResolution');
        if (resEl) resEl.textContent = `${selectedTrack.width}x${selectedTrack.height}`;

        if (selectedTrack.mediaStreamId) {
            this._media.selectStream(selectedTrack.mediaStreamId);
            {
                const targetStream = this._media.selectedStream();
                if (targetStream) {
                    const video = document.getElementById('remoteVideo');
                    if (video && video.srcObject !== targetStream) {
                        video.srcObject = targetStream;
                    }
                }
            }
        }

        if (this.mouseHandler) {
            this.mouseHandler.setRemoteResolution(selectedTrack.width, selectedTrack.height);
        }
        if (this.touchHandler) {
            this.touchHandler.setRemoteResolution(selectedTrack.width, selectedTrack.height);
        }

        if (this.floatingToolbar) {
            // Same reason as the display switch above: "original" tracks the
            // monitor being viewed, and this is the other path that changes it.
            this.floatingToolbar.setRemoteResolution(selectedTrack.width, selectedTrack.height);
            this.floatingToolbar.updateDisplayList(displays, selectedIndex);
        }
    }

    _setupInputHandlers() {
        const video = document.getElementById('remoteVideo');
        const videoContainer = document.getElementById('videoContainer');
        if (!video || !this.dcHandler) return;

        // QD-314: exclusive control means this tab is not the one driving the
        // machine. The answer arrives on the signaling socket, which can beat
        // the media, so it is enforced here rather than only on revoke.
        if (this.controlGrant && !this.controlGrant.canControl) return;

        this._cleanupInputHandlers();

        // Before the first track arrives there is no honest answer, and a guess is
        // worse than nothing: 1920x1080 is a plausible desktop, so nothing looks
        // broken while every click is quietly wrong. Leaving the handlers
        // unconfigured makes the mapping return null, which drops the event
        // instead of injecting it at the wrong place. `onloadedmetadata` and the
        // display-switch path both set the real resolution and re-run this.
        const w = this._remoteWidth || 0;
        const h = this._remoteHeight || 0;
        if (!w || !h) return;

        if (this._isMobile) {
            this.touchHandler = new TouchHandler(videoContainer || video, video, this.dcHandler);
            this.touchHandler.setRemoteResolution(w, h);
            this.touchHandler.enable();
        } else {
            this.mouseHandler = new MouseHandler(video, this.dcHandler, videoContainer || video);
            this.mouseHandler.setRemoteResolution(w, h);
            this.mouseHandler.enable();

            this.keyboardHandler = new KeyboardHandler(videoContainer || video, this.dcHandler);
            this.keyboardHandler.enable();
        }

        // QD-909: the device id is what ties an audit row to a machine, so the
        // handler needs it to report clipboard syncs at all.
        this.clipboardHandler = new ClipboardHandler(this.dcHandler, {
            deviceId: this._deviceId || '',
        });
        this.clipboardHandler.enable();

        this.tabCoordinator?.claim('control');

        this.cursorRenderer = new CursorRenderer(videoContainer || video);

        this._setupDragDrop(videoContainer || video);
        this._setupPasteUpload(videoContainer || video);

        if (this._isMobile) {
            this._enableMobileKeyboard();
        } else {
            (videoContainer || video).focus();
        }
    }

    _cleanupInputHandlers() {
        if (this.mouseHandler) { this.mouseHandler.destroy(); this.mouseHandler = null; }
        if (this.keyboardHandler) { this.keyboardHandler.destroy(); this.keyboardHandler = null; }
        if (this.touchHandler) { this.touchHandler.destroy(); this.touchHandler = null; }
        this._mobileKbInput = null;
        if (this.clipboardHandler) { this.clipboardHandler.destroy(); this.clipboardHandler = null; }
        if (this.cursorRenderer) { this.cursorRenderer.destroy(); this.cursorRenderer = null; }

        // The host keeps its own record of which keys and buttons are down, and
        // it has no way to notice that this controller is gone. Destroying the
        // handlers drops the key-up events that were still pending, so the host
        // is left holding them: the next person at the machine types into
        // whatever modifier was stuck. The Qt client guards the same exposure
        // with releaseAllInput on focus loss and teardown.
        if (this.dcHandler) {
            try {
                this.dcHandler.sendAction('releaseAllInput');
            } catch (_) {
                // A channel that is already closed has nothing to release.
            }
        }
    }

    _sendInitialConfig() {
        this.dcHandler.sendCapabilities(
            'sendAttentionSequenceAction lockWorkstationAction fileTransfer privacyScreen virtualDisplay');
        this.dcHandler.sendAudioControl({
            enable: this._wallMode ? this._wallAudioEnabled : true,
        });
        if (!this._wallMode) {
            this.tabCoordinator?.claim('audio');
        }
        this._log(t('log.sentConfig'));
    }

    _handleToolbarAction(detail) {
        switch (detail.action) {
            case 'disconnect':
                this._disconnect();
                break;
            case 'fitWindow': {
                const video = document.getElementById('remoteVideo');
                if (video) video.style.objectFit = 'contain';
                break;
            }
            case 'screenshot':
                this._screenshot();
                break;
            case 'toggleLogs':
                this._toggleLogDrawer();
                break;
            case 'sendAttentionSequence':
                if (this.dcHandler) {
                    this.dcHandler.sendAction('sendAttentionSequence');
                    this._log(t('log.sentCAD'));
                }
                break;
            case 'lockWorkstation':
                if (this.dcHandler) {
                    this.dcHandler.sendAction('lockWorkstation');
                    this._log(t('log.sentLock'));
                }
                break;
            case 'enablePrivacyScreen':
                if (this.dcHandler) {
                    this.dcHandler.sendAction('enablePrivacyScreen');
                    this._log(t('log.privacyScreenEnabled'));
                }
                break;
            case 'disablePrivacyScreen':
                if (this.dcHandler) {
                    this.dcHandler.sendAction('disablePrivacyScreen');
                    this._log(t('log.privacyScreenDisabled'));
                }
                break;
            case 'uploadFile':
                this._triggerFileUpload();
                break;
            case 'downloadFile':
                this._onDownloadFromHost();
                break;
            case 'showTransfers':
                this._toggleTransferPanel();
                break;
            case 'fileManager':
                this._openFileManager();
                break;
            case 'sessionChat':
                this._openSessionChat();
                break;
            case 'toggleAnnotate':
                this._setupAnnotations();
                this._toggleAnnotations();
                break;
            case 'selectDisplay':
                this._selectDisplay(detail.index);
                break;
            case 'createVirtualDisplay':
                this._sendVirtualDisplayCommand({
                    action: 'create',
                    width: detail.width,
                    height: detail.height,
                    refreshRate: detail.refreshRate,
                });
                break;
            case 'removeVirtualDisplay':
                this._sendVirtualDisplayCommand({
                    action: 'remove',
                    index: detail.index,
                });
                break;
            case 'removeAllVirtualDisplays':
                this._sendVirtualDisplayCommand({ action: 'removeAll' });
                break;
        }
    }

    // ==================== Display Switching ====================

    _selectDisplay(index) {
        if (!this._currentDisplayList || index < 0 || index >= this._currentDisplayList.length) return;

        const track = this._currentDisplayList[index];
        if (!track.mediaStreamId) return;

        this._media.selectStream(track.mediaStreamId);
        this._activeDisplayIndex = index;
        this._remoteWidth = track.width;
        this._remoteHeight = track.height;

        {
            const targetStream = this._media.selectedStream();
            if (targetStream) {
                const video = document.getElementById('remoteVideo');
                if (video) {
                    const combinedStream = new MediaStream();
                    const audioTrack = this._media.usableAudioTrack();
                    if (audioTrack) {
                        combinedStream.addTrack(audioTrack);
                    }
                    for (const vt of targetStream.getVideoTracks()) {
                        combinedStream.addTrack(vt);
                    }
                    video.srcObject = combinedStream;
                }
            }
        }

        if (this.mouseHandler) this.mouseHandler.setRemoteResolution(track.width, track.height);
        if (this.touchHandler) this.touchHandler.setRemoteResolution(track.width, track.height);

        // The toolbar's "original" entry and its resolution readout both follow
        // the monitor being viewed. `onloadedmetadata` only fires for the first
        // stream, so without this the toolbar keeps reporting the first monitor
        // after a switch.
        if (this.floatingToolbar) {
            this.floatingToolbar.setRemoteResolution(track.width, track.height);
        }

        const resEl = document.getElementById('connResolution');
        if (resEl) resEl.textContent = `${track.width}x${track.height}`;

        if (this.floatingToolbar) {
            this.floatingToolbar.updateDisplayList(this._currentDisplayList, index);
        }

        this._log(t('log.switchDisplay', { index }));
    }

    // ==================== Virtual Display ====================

    _sendVirtualDisplayCommand(command) {
        if (!this.dcHandler || !this._supportsVirtualDisplay) return;
        this.dcHandler.sendExtensionMessage('qd-virtual-display', JSON.stringify(command));
    }

    _onExtensionMessage(msg) {
        if (!msg) return;
        if (msg.type === 'qd-skill-bridge') {
            let skillMessage;
            try {
                skillMessage = JSON.parse(msg.data);
            } catch {
                this._log(t('log.skillBridgeInvalidJson'), 'warning');
                return;
            }
            this.skillBridge?.handleMessage(skillMessage);
            this._updateFileManagerSupport();
            this._updateChatSupport();
            return;
        }
        if (msg.type !== 'qd-virtual-display') return;

        let state;
        try {
            state = JSON.parse(msg.data);
        } catch (e) {
            return;
        }

        const type = state.type || '';
        if (type === 'created') {
            this._log(t('log.vdCreated', { index: state.index, resolution: `${state.width}x${state.height}` }));
            this._sendVirtualDisplayCommand({ action: 'query' });
        } else if (type === 'removed') {
            this._log(t('log.vdRemoved', { index: state.index }));
            this._sendVirtualDisplayCommand({ action: 'query' });
        } else if (type === 'removedAll') {
            this._log(t('log.vdRemovedAll'));
            if (this.floatingToolbar) this.floatingToolbar.updateVirtualDisplays([]);
        } else if (type === 'state') {
            const displays = state.displays || [];
            const active = displays.filter(d => d.active);
            this._log(t('log.vdQuery', { count: active.length }));
            if (this.floatingToolbar) this.floatingToolbar.updateVirtualDisplays(displays);
        } else if (type === 'error') {
            this._log(t('log.vdError', { message: state.message || 'Unknown' }), 'warning');
        }
    }

    _updateFileManagerSupport() {
        const requiredTools = [
            'get_default_directory',
            'list_directory',
            'create_directory',
            'move_file',
            'trash_path',
            'read_file_chunk',
            'write_file_chunk',
        ];
        const supported = Boolean(
            this.skillBridge
            && requiredTools.every(tool => this.skillBridge.supportsTool(tool))
        );
        if (supported === this._supportsFileManager) return;
        this._supportsFileManager = supported;
        this.floatingToolbar?.setFileManagerSupport(supported);
        this._log(supported
            ? t('log.fileManagerReady')
            : t('log.fileManagerUnavailable'), supported ? 'info' : 'warning');
        if (!supported) this.fileManagerPanel?.close();
    }

    _updateChatSupport() {
        const supported = Boolean(
            this.skillBridge
            && this.skillBridge.supportsTool('session_chat')
        );
        if (supported === this._supportsChat) return;
        this._supportsChat = supported;
        this.floatingToolbar?.setChatSupport(supported);
        if (!supported) this.chatPanel?.close();
    }

    _openFileManager() {
        if (!this._supportsFileManager || this._wallMode) return;
        if (!this.fileManagerPanel) {
            const container = document.getElementById('remoteContainer');
            if (!container || !this.fileManager) return;
            this.fileManagerPanel = new FileManagerPanel(container, this.fileManager);
        }
        this.fileManagerPanel.open();
    }

    _openSessionChat() {
        if (!this._supportsChat || this._wallMode || !this.chatPanel) return;
        this.chatPanel.open();
    }

    // ==================== File Transfer ====================

    _setupDragDrop(target) {
        if (!target) return;
        let dragCounter = 0;

        target.addEventListener('dragenter', (e) => {
            e.preventDefault();
            e.stopPropagation();
            dragCounter++;
            if (dragCounter === 1) this._showDropOverlay(target);
        });

        target.addEventListener('dragover', (e) => {
            e.preventDefault();
            e.stopPropagation();
        });

        target.addEventListener('dragleave', (e) => {
            e.preventDefault();
            e.stopPropagation();
            dragCounter--;
            if (dragCounter <= 0) {
                dragCounter = 0;
                this._hideDropOverlay(target);
            }
        });

        target.addEventListener('drop', (e) => {
            e.preventDefault();
            e.stopPropagation();
            dragCounter = 0;
            this._hideDropOverlay(target);
            if (e.dataTransfer?.files?.length > 0) {
                this._uploadFiles(e.dataTransfer.files);
            }
        });
    }

    _showDropOverlay(target) {
        if (this._dropOverlay) return;
        const overlay = document.createElement('div');
        overlay.className = 'drop-overlay';
        overlay.innerHTML = `<div class="drop-overlay-content">
            <div class="drop-icon">📤</div>
            <div class="drop-text">${t('drop.text')}</div>
        </div>`;
        target.style.position = 'relative';
        target.appendChild(overlay);
        this._dropOverlay = overlay;
    }

    _hideDropOverlay(target) {
        if (this._dropOverlay) {
            this._dropOverlay.remove();
            this._dropOverlay = null;
        }
    }

    _uploadFiles(fileList) {
        if (!this.dcHandler || !fileList || fileList.length === 0) return;
        for (const file of fileList) {
            this._log(t('log.uploadingFile', { name: file.name, size: file.size }));
            this.dcHandler.startFileUpload(file);
        }
        this._showTransferPanel();
    }

    _setupPasteUpload(target) {
        if (!target) return;
        target.addEventListener('paste', (e) => {
            const files = e.clipboardData?.files;
            if (files && files.length > 0) {
                e.preventDefault();
                this._uploadFiles(files);
            }
        });
    }

    async _onDownloadFromHost() {
        if (!this.dcHandler) return;

        let fileHandle = null;
        if (window.showSaveFilePicker) {
            try {
                fileHandle = await window.showSaveFilePicker({
                    suggestedName: 'download',
                });
            } catch (err) {
                if (err.name === 'AbortError') {
                    this._log(t('log.downloadCancelled'));
                    return;
                }
                this._log(t('log.savePickerError', { error: err.message }), 'warning');
            }
        }

        this.dcHandler.startFileDownload(fileHandle);
        this._log(fileHandle
            ? t('log.downloadStreaming')
            : t('log.downloadBuffered'));
    }

    _triggerFileUpload() {
        const input = document.createElement('input');
        input.type = 'file';
        input.multiple = true;
        input.style.display = 'none';
        input.addEventListener('change', () => {
            this._uploadFiles(input.files);
            input.remove();
        });
        document.body.appendChild(input);
        input.click();
    }

    // ==================== Transfer Panel ====================

    _ensureTransferPanel() {
        if (this._transferPanel) return;

        const panel = document.createElement('div');
        panel.className = 'transfer-panel';
        panel.innerHTML = `
            <div class="transfer-panel-header">
                <span class="transfer-panel-title">File Transfers</span>
                <span class="transfer-panel-close" title="Close">&times;</span>
            </div>
            <div class="transfer-panel-list"></div>
            <div class="transfer-panel-footer">
                <button class="transfer-panel-clear">Clear Completed</button>
            </div>
        `;
        panel.style.display = 'none';

        panel.querySelector('.transfer-panel-close').addEventListener('click', () => {
            this._hideTransferPanel();
        });
        panel.querySelector('.transfer-panel-clear').addEventListener('click', () => {
            this._clearCompletedTransfers();
        });

        const container = document.getElementById('remoteContainer') || document.body;
        container.appendChild(panel);
        this._transferPanel = panel;
        this._transferItems = new Map();
    }

    _showTransferPanel() {
        this._ensureTransferPanel();
        this._transferPanel.style.display = 'flex';
    }

    _hideTransferPanel() {
        if (this._transferPanel) {
            this._transferPanel.style.display = 'none';
        }
    }

    _toggleTransferPanel() {
        this._ensureTransferPanel();
        if (this._transferPanel.style.display === 'none') {
            this._showTransferPanel();
        } else {
            this._hideTransferPanel();
        }
    }

    _addTransferItem(transferId, filename, totalBytes, direction = 'upload') {
        this._ensureTransferPanel();
        const list = this._transferPanel.querySelector('.transfer-panel-list');
        const item = document.createElement('div');
        item.className = 'transfer-item';
        item.dataset.transferId = transferId;
        item.dataset.direction = direction;
        const dirIcon = direction === 'download' ? '📥' : '📤';
        item.innerHTML = `
            <div class="transfer-item-row">
                <span class="transfer-item-icon">${dirIcon}</span>
                <span class="transfer-item-name" title="${filename}">${filename}</span>
                <span class="transfer-item-pct">0%</span>
                <span class="transfer-item-cancel" title="Cancel">&times;</span>
            </div>
            <div class="transfer-item-bar"><div class="transfer-item-fill"></div></div>
        `;
        item.querySelector('.transfer-item-cancel').addEventListener('click', () => {
            if (this.dcHandler) {
                if (direction === 'download') {
                    this.dcHandler.cancelFileDownload(transferId);
                } else {
                    this.dcHandler.cancelFileUpload(transferId);
                }
            }
        });
        list.appendChild(item);
        this._transferItems.set(transferId, item);
        this._updateActiveCount();
        this._showTransferPanel();
    }

    /**
     * @param {string} transferId
     * @param {number} percent already-clamped 0-100
     */
    _updateTransferProgress(transferId, percent) {
        const item = this._transferItems?.get(transferId);
        if (!item) return;
        const pctEl = item.querySelector('.transfer-item-pct');
        if (pctEl) pctEl.textContent = `${pct}%`;
        const fill = item.querySelector('.transfer-item-fill');
        if (fill) fill.style.width = `${pct}%`;
    }

    _updateTransferStatus(transferId, status, errorMessage) {
        const item = this._transferItems?.get(transferId);
        if (!item) return;

        const pctEl = item.querySelector('.transfer-item-pct');
        const cancelEl = item.querySelector('.transfer-item-cancel');
        const fill = item.querySelector('.transfer-item-fill');
        const iconEl = item.querySelector('.transfer-item-icon');

        if (status === 'complete') {
            if (pctEl) pctEl.textContent = '✅';
            if (cancelEl) cancelEl.style.display = 'none';
            if (fill) { fill.style.width = '100%'; fill.classList.add('complete'); }
            if (iconEl) iconEl.textContent = '✅';
            item.classList.add('transfer-complete');
        } else {
            const msg = errorMessage || 'Failed';
            if (pctEl) pctEl.textContent = '❌';
            if (cancelEl) cancelEl.style.display = 'none';
            if (fill) fill.classList.add('error');
            if (iconEl) iconEl.textContent = '❌';
            item.classList.add('transfer-error');
            const errEl = document.createElement('div');
            errEl.className = 'transfer-item-error';
            errEl.textContent = msg;
            item.appendChild(errEl);
        }
        this._updateActiveCount();
    }

    _clearCompletedTransfers() {
        if (!this._transferItems) return;
        for (const [id, item] of this._transferItems) {
            if (item.classList.contains('transfer-complete') || item.classList.contains('transfer-error')) {
                item.remove();
                this._transferItems.delete(id);
            }
        }
        this._updateActiveCount();
    }

    _updateActiveCount() {
        if (!this._transferItems || !this.floatingToolbar) return;
        let count = 0;
        for (const [, item] of this._transferItems) {
            if (!item.classList.contains('transfer-complete') && !item.classList.contains('transfer-error')) {
                count++;
            }
        }
        this.floatingToolbar.updateTransferCount(count);
    }

    _triggerBrowserDownload(blob, filename) {
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        a.style.display = 'none';
        document.body.appendChild(a);
        a.click();
        setTimeout(() => {
            document.body.removeChild(a);
            URL.revokeObjectURL(url);
        }, 100);
    }

    _handleSettingChange(detail) {
        if (!this.dcHandler) return;

        switch (detail.setting) {
            case 'framerate':
                this.dcHandler.sendVideoControl({ enable: true, targetFramerate: detail.value });
                this._log(t('log.targetFps', { fps: detail.value }));
                break;
            case 'framerateBoost': {
                const boostConfig = {
                    off: { enabled: false },
                    office: { enabled: true, captureIntervalMs: 30, boostDurationMs: 300 },
                    gaming: { enabled: true, captureIntervalMs: 15, boostDurationMs: 500 },
                };
                this.dcHandler.sendVideoControl({
                    framerateBoost: boostConfig[detail.value] || { enabled: false },
                });
                this._log(t('log.boostMode', { mode: detail.value }));
                break;
            }
            case 'bitrate':
                this.dcHandler.sendPeerConnectionParameters({
                    preferredMinBitrateBps: detail.value,
                });
                this._log(t('log.minBitrate', { bitrate: Math.round(detail.value / (1024 * 1024)) }));
                break;
            case 'resolution': {
                let width, height;
                if (detail.value === 'original') {
                    const original = this.floatingToolbar?._originalResolution;
                    if (original && original.width > 0) {
                        width = original.width;
                        height = original.height;
                    } else return;
                } else {
                    const parts = detail.value.split('x');
                    width = parseInt(parts[0]);
                    height = parseInt(parts[1]);
                }
                this.dcHandler.sendClientResolution({ widthPixels: width, heightPixels: height, xDpi: 96, yDpi: 96 });
                this._log(t('log.resolution', { width, height }));
                break;
            }
            case 'audio':
                if (detail.value) {
                    this.tabCoordinator?.claim('audio');
                } else {
                    this.tabCoordinator?.release('audio');
                }
                this.dcHandler.sendAudioControl({ enable: detail.value });
                this._log(t('log.audioToggle', { state: detail.value ? t('log.audioOn') : t('log.audioOff') }));
                break;
            case 'stats':
                if (this.videoStats) {
                    detail.value ? this.videoStats.show() : this.videoStats.hide();
                }
                break;
        }
    }

    _screenshot() {
        const video = document.getElementById('remoteVideo');
        if (!video || !video.videoWidth) {
            this._log(t('log.noVideoScreenshot'), 'warning');
            return;
        }
        const canvas = document.createElement('canvas');
        canvas.width = video.videoWidth;
        canvas.height = video.videoHeight;
        canvas.getContext('2d').drawImage(video, 0, 0);
        canvas.toBlob((blob) => {
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `quickdesk_screenshot_${Date.now()}.png`;
            a.click();
            URL.revokeObjectURL(url);
            this._log(t('log.screenshotSaved'));
        }, 'image/png');
    }

    _disconnect() {
        this._manualDisconnect = true
        // Release held input while the event data channel is still writable.
        this._cleanupInputHandlers();
        if (this._detachVisibilityRelease) {
            this._detachVisibilityRelease();
            this._detachVisibilityRelease = null;
        }
        // The session is over, so its media state goes with it - the same reason
        // as the reconnect path.
        this._media.reset();
        this._transferProgress.clear();
        this.skillBridge?.close();
        this.chatPanel?.close();
        this.tabCoordinator?.releaseAll();
        // QD-315: annotations belong to this session. Leaving the strokes on a
        // canvas that outlives the connection would show marks over a screen
        // that no longer exists - on the mobile path the window stays alive
        // after `history.back()`.
        this._teardownAnnotations();
        if (this.session) {
            this.session.disconnect();
        }
        this._setConnectionState('failed', t('status.disconnected'));
        setTimeout(() => {
            if (this._isMobile) {
                if (window.history.length > 1) {
                    window.history.back();
                } else {
                    window.location.href = 'index.html';
                }
            } else {
                window.close();
            }
        }, 500);
    }

    // Wall tiles report their state to the embedding page, which owns the
    // per-tile chrome (name, status badge, actions) - the tile itself stays bare.
    _wallNotify(state, detail = '') {
        if (!this._wallMode || !window.parent || window.parent === window) return;
        const device = this._deviceId
            || new URLSearchParams(window.location.search).get('device')
            || '';
        try {
            window.parent.postMessage(
                { type: 'quickdesk-wall', device, state, detail },
                window.location.origin,
            );
        } catch { /* not embedded in a wall */ }
    }

    // The wall page arms exactly one tile when the operator wants to drive it.
    // Locked tiles stay input-free, so a wall can never send a stray click to a
    // machine nobody is looking at.
    _onWallMessage(event) {
        if (!this._wallMode || event.origin !== window.location.origin) return;
        const data = event.data;
        if (!data || !data.type) return;
        const device = new URLSearchParams(window.location.search).get('device') || '';
        if (data.device && data.device !== device) return;
        if (data.type === 'quickdesk-wall-control') {
            this._setWallInput(!!data.armed);
        } else if (data.type === 'quickdesk-wall-audio') {
            this._setWallAudio(!!data.enabled);
        }
    }

    _setWallInput(armed) {
        if (!this._wallMode || this._inputArmed === armed) return;
        this._inputArmed = armed;
        document.body.classList.toggle('qd-wall-armed', armed);
        if (armed) {
            // Before the media is up there is nothing to attach handlers to; the
            // metadata callback re-applies the armed state.
            if (this.dcHandler) this._setupInputHandlers();
        } else {
            this._cleanupInputHandlers();
            this.tabCoordinator?.release('control');
        }
    }

    _setWallAudio(enabled) {
        if (!this._wallMode || this._wallAudioEnabled === enabled) return;
        this._wallAudioEnabled = enabled;
        this.dcHandler?.sendAudioControl({ enable: enabled });
        const video = document.getElementById('remoteVideo');
        if (!video) return;
        video.muted = !enabled;
        if (enabled) {
            video.play().catch((error) => {
                this._log(t('log.autoplayFailed', { error: error.message }), 'warning');
            });
        }
    }

    _setConnectionState(state, text) {
        if (this._wallMode) {
            this._wallNotify(state === 'failed' ? 'error' : state, text || '');
        }
        const dot = document.getElementById('connDot');
        const statusEl = document.getElementById('connStatus');
        if (dot) {
            dot.className = 'conn-dot';
            if (state === 'connected') dot.classList.add('connected');
            else if (state === 'failed') dot.classList.add('failed');
        }
        if (statusEl) statusEl.textContent = text;
        if (state !== 'connected') {
            this._stopConnBarStats();
            const routeEl = document.getElementById('connRoute');
            const pingEl = document.getElementById('connPing');
            if (routeEl) routeEl.style.display = 'none';
            if (pingEl) pingEl.style.display = 'none';
        }
    }

    _startConnBarStats() {
        if (this._connBarInterval) return;
        this._connBarInterval = setInterval(() => this._updateConnBarStats(), 2000);
        this._updateConnBarStats();
    }

    _stopConnBarStats() {
        if (this._connBarInterval) {
            clearInterval(this._connBarInterval);
            this._connBarInterval = null;
        }
    }

    async _updateConnBarStats() {
        if (!this.session?.pc || this.session.pc.connectionState === 'closed') return;
        try {
            const stats = await this.session.pc.getStats();
            let rtt = 0;
            let localId = null;
            let remoteId = null;

            stats.forEach(report => {
                if (report.type === 'candidate-pair' && report.state === 'succeeded') {
                    if (!localId || report.nominated) {
                        rtt = Math.round((report.currentRoundTripTime || 0) * 1000);
                        localId = report.localCandidateId;
                        remoteId = report.remoteCandidateId;
                    }
                }
            });

            let localType = '';
            let remoteType = '';
            if (localId) {
                const lc = stats.get(localId);
                if (lc) localType = lc.candidateType || '';
            }
            if (remoteId) {
                const rc = stats.get(remoteId);
                if (rc) remoteType = rc.candidateType || '';
            }

            // Route type
            const routeEl = document.getElementById('connRoute');
            const routeDot = document.getElementById('connRouteDot');
            const routeLabel = document.getElementById('connRouteLabel');
            if (routeEl && localType) {
                let label, color;
                if (localType === 'relay' || remoteType === 'relay') {
                    label = 'Relay'; color = '#FFA726';
                } else if (localType === 'srflx' || localType === 'prflx' ||
                           remoteType === 'srflx' || remoteType === 'prflx') {
                    label = 'STUN'; color = '#66BB6A';
                } else {
                    label = 'P2P'; color = '#66BB6A';
                }
                routeDot.style.background = color;
                routeLabel.textContent = label;
                routeEl.style.display = 'inline-flex';
            }

            // Ping
            const pingEl = document.getElementById('connPing');
            const pingDot = document.getElementById('connPingDot');
            const pingValue = document.getElementById('connPingValue');
            if (pingEl && rtt > 0) {
                const pingColor = rtt < 50 ? '#4caf50' : rtt < 100 ? '#ffc107' : '#f44336';
                pingDot.style.background = pingColor;
                pingValue.textContent = rtt + ' ms';
                pingEl.style.display = 'inline-flex';
            }
        } catch (e) {
            // ignore
        }
    }

    _initLogDrawer() {
        document.getElementById('logCloseBtn')?.addEventListener('click', () => this._toggleLogDrawer(false));
        document.getElementById('logClearBtn')?.addEventListener('click', () => {
            const container = document.getElementById('logContainer');
            if (container) container.innerHTML = '';
        });
    }

    _toggleLogDrawer(forceState) {
        const drawer = document.getElementById('logDrawer');
        if (!drawer) return;
        const open = forceState !== undefined ? forceState : !drawer.classList.contains('open');
        drawer.classList.toggle('open', open);
        if (open) {
            const container = document.getElementById('logContainer');
            if (container) container.scrollTop = container.scrollHeight;
        }
    }

    _setupMobileToolbar() {
        const toolbar = document.getElementById('mobileToolbar');
        if (!toolbar) return;
        toolbar.style.display = 'block';

        const keyInput = document.getElementById('mobileKeyInput');

        document.getElementById('btnKeyboard')?.addEventListener('click', () => {
            if (!keyInput) return;
            keyInput.style.pointerEvents = 'auto';
            keyInput.focus();
            keyInput.style.pointerEvents = 'none';
        });

        document.getElementById('btnRightClick')?.addEventListener('click', () => {
            if (!this.touchHandler || !this.dcHandler) return;
            const x = Math.round(this.touchHandler._cursorX);
            const y = Math.round(this.touchHandler._cursorY);
            this.dcHandler.sendMouseEvent({ x, y, button: MouseButton.BUTTON_RIGHT, buttonDown: true });
            this.dcHandler.sendMouseEvent({ x, y, button: MouseButton.BUTTON_RIGHT, buttonDown: false });
        });

        document.getElementById('btnLogs')?.addEventListener('click', () => {
            this._toggleLogDrawer();
        });

        document.getElementById('btnZoomReset')?.addEventListener('click', () => {
            if (this.touchHandler) this.touchHandler.resetZoom();
        });

        document.getElementById('btnFullscreen')?.addEventListener('click', () => {
            if (document.fullscreenElement) {
                document.exitFullscreen();
            } else {
                document.documentElement.requestFullscreen?.();
            }
        });

        // QD-315: the desktop floating menu is not created in mobile layout, so
        // annotation needs its own entry here or the feature is unreachable on a
        // phone. The same toggle drives both.
        const annotateButton = document.getElementById('btnAnnotate');
        annotateButton?.addEventListener('click', () => {
            this._setupAnnotations();
            this._toggleAnnotations();
            annotateButton.classList.toggle('active', this._annotating);
        });

        document.getElementById('btnDisconnect')?.addEventListener('click', () => {
            this._disconnect();
        });
    }

    _enableMobileKeyboard() {
        const keyInput = document.getElementById('mobileKeyInput');
        if (!keyInput || !this.dcHandler || this._mobileKbInput) return;
        this._mobileKbInput = keyInput;
        this._isComposing = false;
        this._inputTimer = null;
        this._lastSentValue = '';

        keyInput.addEventListener('compositionstart', () => {
            this._isComposing = true;
        });

        keyInput.addEventListener('compositionend', (e) => {
            this._isComposing = false;
            clearTimeout(this._inputTimer);
            keyInput.value = '';
            if (e.data) {
                this._sendTextAsKeys(e.data);
            }
        });

        keyInput.addEventListener('input', () => {
            if (this._isComposing) return;
            clearTimeout(this._inputTimer);
            this._inputTimer = setTimeout(() => {
                const text = keyInput.value;
                if (!text) return;
                keyInput.value = '';
                this._sendTextAsKeys(text);
            }, 60);
        });

        keyInput.addEventListener('keydown', (e) => {
            if (e.key === 'Backspace' || e.key === 'Enter') {
                e.preventDefault();
                const usb = e.key === 'Backspace' ? 0x07002A : 0x070028;
                this.dcHandler.sendKeyEvent({ pressed: true, usbKeycode: usb });
                this.dcHandler.sendKeyEvent({ pressed: false, usbKeycode: usb });
            }
        });

        this._setupKeyboardResize();
    }

    _sendTextAsKeys(text) {
        let asciiBuffer = '';
        for (const ch of text) {
            if (CHAR_TO_USB[ch] || CHAR_TO_USB[ch.toLowerCase()]) {
                asciiBuffer += ch;
            } else {
                if (asciiBuffer) {
                    for (const c of asciiBuffer) this._sendCharAsKey(c);
                    asciiBuffer = '';
                }
                // Non-ASCII (中文等): use TextEvent protocol
                this.dcHandler.sendTextEvent(ch);
            }
        }
        if (asciiBuffer) {
            for (const c of asciiBuffer) this._sendCharAsKey(c);
        }
    }

    _setupKeyboardResize() {
        const vv = window.visualViewport;
        if (!vv) return;
        let lastHeight = vv.height;

        vv.addEventListener('resize', () => {
            const h = vv.height;
            if (Math.abs(h - lastHeight) < 50) return;
            lastHeight = h;
            document.body.style.height = `${h}px`;
            requestAnimationFrame(() => {
                window.scrollTo(0, 0);
                if (this.touchHandler?.isZoomed) {
                    this.touchHandler._autoPanToFollow();
                }
            });
        });
    }

    _sendCharAsKey(ch) {
        const usb = CHAR_TO_USB[ch] || CHAR_TO_USB[ch.toLowerCase()];
        if (!usb) return;
        const needShift = (ch >= 'A' && ch <= 'Z') || SHIFT_CHARS.has(ch);
        const SHIFT_USB = 0x0700E1;
        if (needShift) this.dcHandler.sendKeyEvent({ pressed: true, usbKeycode: SHIFT_USB });
        this.dcHandler.sendKeyEvent({ pressed: true, usbKeycode: usb });
        this.dcHandler.sendKeyEvent({ pressed: false, usbKeycode: usb });
        if (needShift) this.dcHandler.sendKeyEvent({ pressed: false, usbKeycode: SHIFT_USB });
    }

    // ==================== Session recording ====================
    //
    // Phase 1 records on the controller: the decoded remote stream is already
    // attached to #remoteVideo, so MediaRecorder can capture exactly what the
    // operator sees without touching the remoting engine. Uploading the result
    // creates a `session_recordings` row and then posts the bytes, which is the
    // same pair of calls a native recorder would make.

    _recordingApiBase() {
        let base = (localStorage.getItem('quickdesk_signaling_url') || '').replace(/\/+$/, '');
        if (base.startsWith('wss://')) base = base.replace(/^wss:\/\//, 'https://');
        else if (base.startsWith('ws://')) base = base.replace(/^ws:\/\//, 'http://');
        return base;
    }

    _recordingAccessToken() {
        return localStorage.getItem('quickdesk_user_access_token') || '';
    }

    _pickRecordingMime() {
        // Chrome 126+ can mux MP4 (H.264) directly; WebM is the fallback.
        const candidates = [
            'video/mp4;codecs=avc1.42E01E',
            'video/mp4',
            'video/webm;codecs=vp9',
            'video/webm;codecs=vp8',
            'video/webm',
        ];
        for (const type of candidates) {
            if (typeof MediaRecorder !== 'undefined' && MediaRecorder.isTypeSupported(type)) return type;
        }
        return '';
    }

    startRecording() {
        if (this._recorder) return true;
        if (this._recordingUploading) {
            this._log(t('log.recordUploadInProgress'), 'error');
            return false;
        }
        const video = document.getElementById('remoteVideo');
        const stream = video && video.srcObject;
        if (!stream) {
            this._log(t('log.recordNoStream'), 'error');
            return false;
        }
        const mimeType = this._pickRecordingMime();
        if (!mimeType) {
            this._log(t('log.recordUnsupported'), 'error');
            return false;
        }
        this._recordChunks = [];
        try {
            this._recorder = new MediaRecorder(stream, { mimeType, videoBitsPerSecond: 4_000_000 });
        } catch (err) {
            this._log(`record: ${err.message}`, 'error');
            this._recorder = null;
            return false;
        }
        this._recorder.ondataavailable = (e) => {
            if (e.data && e.data.size > 0) this._recordChunks.push(e.data);
        };
        // Grab the poster now, while the element still holds a live frame.
        // Capturing at stop time would be unreliable: the decoder may have been
        // torn down and the canvas would come back blank.
        this._recordPoster = this._captureRecordingPoster();
        this._recorder.start(1000);
        this._recordStartedAt = Date.now();
        this._setRecordButtonState(true);
        this._log(t('log.recordStarted'));
        return true;
    }

    // A preview is a thumbnail, not a screenshot: cap the long edge so a 4K
    // session does not produce a 4K preview that must be re-encoded on every
    // recordings-list render.
    _captureRecordingPoster() {
        const video = document.getElementById('remoteVideo');
        if (!video || !video.videoWidth || !video.videoHeight) return Promise.resolve(null);
        const scale = Math.min(1, 640 / Math.max(video.videoWidth, video.videoHeight));
        const canvas = document.createElement('canvas');
        canvas.width = Math.max(1, Math.round(video.videoWidth * scale));
        canvas.height = Math.max(1, Math.round(video.videoHeight * scale));
        const ctx = canvas.getContext('2d');
        if (!ctx) return Promise.resolve(null);
        try {
            ctx.drawImage(video, 0, 0, canvas.width, canvas.height);
        } catch (err) {
            console.warn('[RemoteDesktop] poster capture failed', err);
            return Promise.resolve(null);
        }
        return new Promise((resolve) => {
            try {
                canvas.toBlob((blob) => resolve(blob), 'image/jpeg', 0.82);
            } catch (err) {
                console.warn('[RemoteDesktop] poster encode failed', err);
                resolve(null);
            }
        });
    }

    async stopRecording() {
        const recorder = this._recorder;
        if (!recorder) return null;
        const stopped = new Promise((resolve) => { recorder.onstop = resolve; });
        recorder.stop();
        await stopped;
        const mimeType = recorder.mimeType || 'video/webm';
        const blob = new Blob(this._recordChunks, { type: mimeType });
        const durationMs = Date.now() - this._recordStartedAt;
        const poster = this._recordPoster;
        this._recorder = null;
        this._recordChunks = [];
        this._recordPoster = null;
        this._setRecordButtonState(false);
        this._log(t('log.recordStopped', { seconds: Math.round(durationMs / 1000), kb: Math.round(blob.size / 1024) }));
        return { blob, mimeType, durationMs, poster };
    }

    async toggleRecording() {
        if (this._recorder) {
            const result = await this.stopRecording();
            if (result && result.blob.size > 0) {
                await this._uploadRecording(result);
            } else if (result) {
                this._log(t('log.recordEmpty'), 'error');
            }
            return;
        }
        this.startRecording();
    }

    async _uploadRecording({ blob, mimeType, durationMs, poster }) {
        const token = this._recordingAccessToken();
        const base = this._recordingApiBase();
        if (!token || !base) {
            this._log(t('log.recordNoSession'), 'error');
            return null;
        }
        const deviceId = new URLSearchParams(window.location.search).get('device') || '';
        const startedAt = new Date(Date.now() - durationMs).toISOString();
        const endedAt = new Date().toISOString();
        const ext = mimeType.includes('mp4') ? 'mp4' : 'webm';
        const headers = { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` };
        // Tracked outside the try so the catch can repair the index when the
        // upload dies at the transport layer (an aborted fetch throws instead
        // of resolving with a non-2xx status).
        let rowId = null;
        this._recordingUploading = true;
        this._setRecordingUploadProgress(0);
        let lastLoggedPercent = 0;
        try {
            const created = await fetch(`${base}/v1/me/recordings`, {
                method: 'POST',
                headers,
                body: JSON.stringify({
                    device_id: deviceId,
                    device_name: '',
                    session_id: `web-${Date.now()}`,
                    // Starts as "recording": the upload below decides whether it
                    // becomes "completed" (server flips it) or "failed".
                    status: 'recording',
                    format: ext,
                    duration_seconds: Math.max(1, Math.round(durationMs / 1000)),
                    started_at: startedAt,
                    ended_at: endedAt,
                }),
            });
            if (!created.ok) throw new Error(`create failed: HTTP ${created.status}`);
            const row = await created.json();
            rowId = row.id;

            const saved = await uploadRecordingArtifact({
                url: `${base}/v1/me/recordings/${row.id}/artifact`,
                token,
                blob,
                ext,
                onProgress: ({ percent }) => {
                    this._setRecordingUploadProgress(percent);
                    const bucket = Math.floor(percent / 10) * 10;
                    if (bucket > lastLoggedPercent) {
                        lastLoggedPercent = bucket;
                        this._log(t('log.recordUploadProgress', { percent: bucket }));
                    }
                },
                onRetry: ({ attempt }) => {
                    this._log(t('log.recordRetry', { label: 'artifact', attempt }));
                },
            });
            this._log(t('log.recordUploaded', { id: saved.id, kb: Math.round((saved.size_bytes || 0) / 1024) }));
            // Best effort from here down: the recording is already stored, so a
            // missing preview is a cosmetic loss, not a failed capture.
            await this._uploadThumbnail(row.id, token, base, poster);
            return saved;
        } catch (err) {
            if (rowId && err.status !== 200) await this._markRecordingFailed(rowId, token, base);
            this._log(t('log.recordUploadFailed', { error: err.message }), 'error');
            return null;
        } finally {
            this._recordingUploading = false;
            this._setRecordButtonState(false);
        }
    }

    // POST a multipart body, retrying only what a retry can fix: transport
    // failures and 5xx/429. A 4xx is the server rejecting this payload, so the
    // loop stops immediately instead of hammering it.
    async _postWithRetry(url, token, makeBody, label) {
        const backoffMs = [0, 800, 2000, 5000];
        let lastError = null;
        for (let attempt = 0; attempt < backoffMs.length; attempt++) {
            if (backoffMs[attempt] > 0) {
                await new Promise((resolve) => setTimeout(resolve, backoffMs[attempt]));
                this._log(t('log.recordRetry', { label, attempt: attempt + 1 }));
            }
            let response;
            try {
                response = await fetch(url, {
                    method: 'POST',
                    headers: { Authorization: `Bearer ${token}` },
                    body: makeBody(),
                });
            } catch (err) {
                lastError = err;
                continue;
            }
            if (response.ok) return response;
            lastError = new Error(`${label} upload failed: HTTP ${response.status}`);
            if (response.status < 500 && response.status !== 429) throw lastError;
        }
        throw lastError || new Error(`${label} upload failed`);
    }

    async _uploadThumbnail(recordingId, token, base, posterPromise) {
        try {
            let poster = null;
            try {
                poster = await posterPromise;
            } catch {
                poster = null;
            }
            if (!poster || poster.size === 0) return false;
            await this._postWithRetry(
                `${base}/v1/me/recordings/${recordingId}/thumbnail`,
                token,
                () => {
                    const form = new FormData();
                    form.append('file', poster, 'thumb.jpg');
                    return form;
                },
                'thumbnail',
            );
            return true;
        } catch (err) {
            this._log(t('log.recordThumbFailed', { error: err.message }), 'error');
            return false;
        }
    }

    // Keep the index honest when the bytes never landed: a row that claims
    // "recording" forever is worse than an explicit failure.
    async _markRecordingFailed(recordingId, token, base) {
        try {
            await fetch(`${base}/v1/me/recordings/${recordingId}`, {
                method: 'PATCH',
                headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
                body: JSON.stringify({ status: 'failed' }),
            });
        } catch { /* the failure is already reported to the user */ }
    }

    _setRecordButtonState(recording) {
        const button = document.getElementById('btnRecord');
        if (!button) return;
        button.classList.toggle('recording', !!recording);
        button.textContent = recording ? '■' : '●';
        button.disabled = !!recording || this._recordingUploading;
        button.title = this._recordingUploading
            ? t('remote.recordingUpload', { percent: 0 })
            : (recording ? t('remote.stopRecording') : t('remote.startRecording'));
    }

    _setRecordingUploadProgress(percent) {
        const button = document.getElementById('btnRecord');
        if (!button) return;
        button.disabled = true;
        button.title = t('remote.recordingUpload', { percent });
    }

    _log(message, level = 'info') {
        const container = document.getElementById('logContainer');
        if (container) {
            const time = new Date().toLocaleTimeString();
            const entry = document.createElement('div');
            entry.className = `log-entry log-${level}`;
            entry.innerHTML = `<span class="log-time">[${time}]</span> ${message}`;
            container.appendChild(entry);
            container.scrollTop = container.scrollHeight;
            while (container.children.length > 500) {
                container.removeChild(container.firstChild);
            }
        }
        console.log(`[RemoteDesktop] ${message}`);
    }
}

// Character → USB HID keycode mapping for mobile virtual keyboard input
const P = 0x070000;
const CHAR_TO_USB = {
    'a':P|0x04,'b':P|0x05,'c':P|0x06,'d':P|0x07,'e':P|0x08,'f':P|0x09,
    'g':P|0x0A,'h':P|0x0B,'i':P|0x0C,'j':P|0x0D,'k':P|0x0E,'l':P|0x0F,
    'm':P|0x10,'n':P|0x11,'o':P|0x12,'p':P|0x13,'q':P|0x14,'r':P|0x15,
    's':P|0x16,'t':P|0x17,'u':P|0x18,'v':P|0x19,'w':P|0x1A,'x':P|0x1B,
    'y':P|0x1C,'z':P|0x1D,
    '1':P|0x1E,'2':P|0x1F,'3':P|0x20,'4':P|0x21,'5':P|0x22,'6':P|0x23,
    '7':P|0x24,'8':P|0x25,'9':P|0x26,'0':P|0x27,
    ' ':P|0x2C,'-':P|0x2D,'=':P|0x2E,'[':P|0x2F,']':P|0x30,'\\':P|0x31,
    ';':P|0x33,"'":P|0x34,'`':P|0x35,',':P|0x36,'.':P|0x37,'/':P|0x38,
    '\t':P|0x2B,
};
// Shift symbols → base key mapping
const SHIFT_CHAR_MAP = {
    '~':'`','!':'1','@':'2','#':'3','$':'4','%':'5','^':'6','&':'7',
    '*':'8','(':'9',')':'0','_':'-','+':'=','{':'[','}':']','|':'\\',
    ':':';','"':"'",'<':',','>':'.','?':'/',
};
for (const [shifted, base] of Object.entries(SHIFT_CHAR_MAP)) {
    if (CHAR_TO_USB[base]) CHAR_TO_USB[shifted] = CHAR_TO_USB[base];
}
const SHIFT_CHARS = new Set(Object.keys(SHIFT_CHAR_MAP));

// Module scripts run after parsing, so the class can be applied before the first
// paint and a wall tile never flashes the full control chrome.
if (new URLSearchParams(window.location.search).get('wall') === '1') {
    document.body.classList.add('qd-wall-tile');
}

document.addEventListener('DOMContentLoaded', () => {
    const app = new RemoteDesktopApp();
    app.init();
    window.remoteDesktopApp = app;

    const recordButton = document.getElementById('btnRecord');
    if (recordButton) {
        recordButton.addEventListener('click', () => {
            app.toggleRecording();
        });
        app._setRecordButtonState(false);
    }
});
