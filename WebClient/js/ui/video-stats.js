/**
 * video-stats.js - Video statistics overlay
 * 
 * Displays connection info, bandwidth, framerate, codec, and ICE/route details.
 * Modeled after Qt QuickDesk's VideoStatsOverlay.qml.
 */

import { t } from '../i18n.js';
import { deriveVideoRates, isComparableSample } from './videoRates.js';

export class VideoStats {
    /**
     * @param {HTMLElement} overlayElement
     * @param {RTCPeerConnection} pc
     */
    constructor(overlayElement, pc) {
        this.overlay = overlayElement;
        this.pc = pc;
        this._visible = false;
        this._updateInterval = null;
        this._prevStats = null;
        this._prevTimestamp = 0;
    }

    show() {
        this._visible = true;
        this.overlay.style.display = 'block';
        this._startUpdate();
    }

    hide() {
        this._visible = false;
        this.overlay.style.display = 'none';
        this._stopUpdate();
    }

    toggle() {
        if (this._visible) {
            this.hide();
        } else {
            this.show();
        }
    }

    /** @private */
    _startUpdate() {
        if (this._updateInterval) return;
        this._updateInterval = setInterval(() => this._update(), 1000);
        this._update();
    }

    /** @private */
    _stopUpdate() {
        if (this._updateInterval) {
            clearInterval(this._updateInterval);
            this._updateInterval = null;
        }
    }

    /** @private */
    async _update() {
        if (!this.pc || this.pc.connectionState === 'closed') return;

        try {
            const stats = await this.pc.getStats();
            const now = Date.now();
            const timeDelta = this._prevTimestamp ? (now - this._prevTimestamp) / 1000 : 1;

            let videoStats = {};
            let audioStats = {};
            let candidatePair = null;
            let activePairLocalId = null;
            let activePairRemoteId = null;
            const localCandidates = [];
            const remoteCandidates = [];

            stats.forEach(report => {
                if (report.type === 'inbound-rtp' && report.kind === 'video') {
                    videoStats = {
                        bytesReceived: report.bytesReceived,
                        framesDecoded: report.framesDecoded,
                        framesReceived: report.framesReceived,
                        framesDropped: report.framesDropped,
                        frameWidth: report.frameWidth,
                        frameHeight: report.frameHeight,
                        jitter: report.jitter,
                        packetsLost: report.packetsLost,
                        packetsReceived: report.packetsReceived,
                        decoderImplementation: report.decoderImplementation,
                        codec: null,
                    };

                    if (report.codecId) {
                        const codecReport = stats.get(report.codecId);
                        if (codecReport) {
                            videoStats.codec = codecReport.mimeType;
                        }
                    }
                }

                if (report.type === 'inbound-rtp' && report.kind === 'audio') {
                    audioStats = {
                        bytesReceived: report.bytesReceived,
                        packetsLost: report.packetsLost,
                        jitter: report.jitter,
                    };
                }

                if (report.type === 'candidate-pair' && report.state === 'succeeded') {
                    if (!candidatePair || report.nominated) {
                        candidatePair = {
                            currentRoundTripTime: report.currentRoundTripTime,
                            availableOutgoingBitrate: report.availableOutgoingBitrate,
                            bytesReceived: report.bytesReceived,
                            bytesSent: report.bytesSent,
                        };
                        activePairLocalId = report.localCandidateId;
                        activePairRemoteId = report.remoteCandidateId;
                    }
                }

                if (report.type === 'local-candidate') {
                    localCandidates.push({
                        id: report.id,
                        type: report.candidateType,
                        protocol: report.protocol,
                        address: report.address,
                        port: report.port,
                        isIpv6: report.address ? report.address.includes(':') : false,
                    });
                }

                if (report.type === 'remote-candidate') {
                    remoteCandidates.push({
                        id: report.id,
                        type: report.candidateType,
                        protocol: report.protocol,
                        address: report.address,
                        port: report.port,
                        isIpv6: report.address ? report.address.includes(':') : false,
                    });
                }
            });

            // Resolve selected local/remote candidate
            let selectedLocal = null;
            let selectedRemote = null;
            if (activePairLocalId) {
                selectedLocal = localCandidates.find(c => c.id === activePairLocalId) || null;
            }
            if (activePairRemoteId) {
                selectedRemote = remoteCandidates.find(c => c.id === activePairRemoteId) || null;
            }

            // Switching monitors replaces the stream and its counters restart from
            // zero, so the previous sample is dropped instead of subtracted from:
            // a negative difference would put a negative frame rate on the overlay.
            const previousVideo = isComparableSample(this._prevStats?.video, videoStats)
                ? this._prevStats.video
                : null;
            const { fps, bitrateKbps: bitrate, packetRate } = deriveVideoRates(
                previousVideo, videoStats, timeDelta,
            );

            this._render({
                video: videoStats,
                audio: audioStats,
                network: candidatePair,
                fps,
                bitrate,
                packetRate,
                selectedLocal,
                selectedRemote,
                localCandidates,
                remoteCandidates,
            });

            this._prevStats = { video: videoStats, audio: audioStats };
            this._prevTimestamp = now;

        } catch (e) {
            console.warn('[VideoStats] Failed to get stats:', e);
        }
    }

    // ── Helpers ──

    /** @private */
    _fmtBandwidth(kbps) {
        if (!kbps || kbps <= 0) return '—';
        if (kbps >= 1000) return (kbps / 1000).toFixed(1) + ' Mbps';
        return kbps + ' kbps';
    }

    /** @private */
    _routeTypeLabel(localType, remoteType) {
        if (localType === 'relay' || remoteType === 'relay') return 'Relay (TURN)';
        if (localType === 'srflx' || localType === 'prflx' ||
            remoteType === 'srflx' || remoteType === 'prflx') return 'P2P (STUN)';
        if (localType === 'host' && remoteType === 'host') return 'P2P (Direct)';
        return '—';
    }

    /** @private */
    _routeTypeColor(localType, remoteType) {
        if (localType === 'relay' || remoteType === 'relay') return '#FFA726';
        return '#66BB6A';
    }

    /** @private */
    _candidateTypeColor(type) {
        if (type === 'host') return '#66BB6A';
        if (type === 'srflx' || type === 'prflx') return '#4FC3F7';
        if (type === 'relay') return '#FFA726';
        return 'rgba(255,255,255,0.7)';
    }

    /** @private */
    _ipVersionColor(isIpv6) {
        return isIpv6 ? '#CE93D8' : '#90CAF9';
    }

    /** @private */
    _formatAddress(addr, port) {
        if (!addr) return '—';
        if (addr.includes('redacted-ip.invalid')) return `hidden:${port || '?'}`;
        return port ? `${addr}:${port}` : addr;
    }

    /** @private */
    _render(data) {
        const rtt = data.network ? Math.round((data.network.currentRoundTripTime || 0) * 1000) : 0;
        const rttColor = rtt < 50 ? '#4caf50' : rtt < 100 ? '#ffc107' : '#f44336';

        const resolution = (data.video.frameWidth && data.video.frameHeight)
            ? `${data.video.frameWidth} × ${data.video.frameHeight}` : '—';
        const codec = data.video.codec || '—';
        const decoder = data.video.decoderImplementation || '—';
        const jitter = data.video.jitter != null ? `${(data.video.jitter * 1000).toFixed(1)} ms` : '—';
        const packetsLost = data.video.packetsLost || 0;

        // ICE / Route
        const localType = data.selectedLocal ? data.selectedLocal.type : '';
        const remoteType = data.selectedRemote ? data.selectedRemote.type : '';
        const routeLabel = this._routeTypeLabel(localType, remoteType);
        const routeColor = this._routeTypeColor(localType, remoteType);
        const protocol = data.selectedLocal ? (data.selectedLocal.protocol || '').toUpperCase() : '—';
        const localAddr = data.selectedLocal
            ? `${this._formatAddress(data.selectedLocal.address, data.selectedLocal.port)} (${data.selectedLocal.type})`
            : '—';
        const remoteAddr = data.selectedRemote
            ? `${this._formatAddress(data.selectedRemote.address, data.selectedRemote.port)} (${data.selectedRemote.type})`
            : '—';

        // Build candidate rows
        const buildCandidateRows = (candidates) => {
            return candidates.map(c => {
                const typeColor = this._candidateTypeColor(c.type);
                const ipColor = this._ipVersionColor(c.isIpv6);
                const ipTag = c.isIpv6 ? 'IPv6' : 'IPv4';
                const addr = this._formatAddress(c.address, c.port);
                return `<div class="stats-candidate-row">
                    <span class="stats-cand-type" style="color:${typeColor}">${c.type || '?'}</span>
                    <span class="stats-cand-proto">${(c.protocol || '').toLowerCase()}</span>
                    <span class="stats-cand-ip" style="color:${ipColor}">${ipTag}</span>
                    <span class="stats-cand-addr">${addr}</span>
                </div>`;
            }).join('');
        };

        this.overlay.innerHTML = `
            <div class="stats-title">${t('stats.title')}</div>
            <div class="stats-divider"></div>

            <div class="stats-section-label">${t('stats.sectionConnection')}</div>
            <div class="stats-grid">
                <div class="stats-row">
                    <span class="stats-label">${t('stats.resolution')}</span>
                    <span class="stats-value">${resolution}</span>
                </div>
                <div class="stats-row">
                    <span class="stats-label">${t('stats.codec')}</span>
                    <span class="stats-value">${codec}</span>
                </div>
                <div class="stats-row">
                    <span class="stats-label">${t('stats.decoder')}</span>
                    <span class="stats-value">${decoder}</span>
                </div>
                <div class="stats-row">
                    <span class="stats-label">${t('stats.fps')}</span>
                    <span class="stats-value">${data.fps} fps</span>
                </div>
                <div class="stats-row">
                    <span class="stats-label">${t('stats.bitrate')}</span>
                    <span class="stats-value">${this._fmtBandwidth(data.bitrate)}</span>
                </div>
                <div class="stats-row">
                    <span class="stats-label">${t('stats.rtt')}</span>
                    <span class="stats-value" style="color:${rttColor}">${rtt} ms</span>
                </div>
                <div class="stats-row">
                    <span class="stats-label">${t('stats.jitter')}</span>
                    <span class="stats-value">${jitter}</span>
                </div>
                <div class="stats-row">
                    <span class="stats-label">${t('stats.packetRate')}</span>
                    <span class="stats-value">${data.packetRate} /s</span>
                </div>
                <div class="stats-row">
                    <span class="stats-label">${t('stats.packetsLost')}</span>
                    <span class="stats-value">${packetsLost}</span>
                </div>
                <div class="stats-row">
                    <span class="stats-label">${t('stats.framesDropped')}</span>
                    <span class="stats-value">${data.video.framesDropped || 0}</span>
                </div>
                ${rtt > 0 ? `
                <div class="stats-bar">
                    <div class="stats-bar-fill" style="width:${Math.min(rtt / 2, 100)}%;background:${rttColor}"></div>
                </div>` : ''}
            </div>

            <div class="stats-divider"></div>
            <div class="stats-section-label">${t('stats.sectionIce')}</div>
            <div class="stats-grid">
                <div class="stats-row">
                    <span class="stats-label">${t('stats.routeType')}</span>
                    <span class="stats-value" style="color:${routeColor};font-weight:600">${routeLabel}</span>
                </div>
                <div class="stats-row">
                    <span class="stats-label">${t('stats.protocol')}</span>
                    <span class="stats-value">${protocol}</span>
                </div>
                <div class="stats-row">
                    <span class="stats-label">${t('stats.localAddr')}</span>
                    <span class="stats-value">${localAddr}</span>
                </div>
                <div class="stats-row">
                    <span class="stats-label">${t('stats.remoteAddr')}</span>
                    <span class="stats-value">${remoteAddr}</span>
                </div>
            </div>

            ${data.localCandidates.length > 0 ? `
            <div class="stats-divider"></div>
            <div class="stats-section-label">${t('stats.clientCandidates')}</div>
            <div class="stats-candidates">${buildCandidateRows(data.localCandidates)}</div>` : ''}

            ${data.remoteCandidates.length > 0 ? `
            <div class="stats-divider"></div>
            <div class="stats-section-label">${t('stats.hostCandidates')}</div>
            <div class="stats-candidates">${buildCandidateRows(data.remoteCandidates)}</div>` : ''}
        `;
    }

    /**
     * @param {RTCPeerConnection} pc 
     */
    setPeerConnection(pc) {
        this.pc = pc;
    }

    getCurrentStats() {
        return this._prevStats;
    }

    destroy() {
        this._stopUpdate();
        this.overlay.innerHTML = '';
    }
}
