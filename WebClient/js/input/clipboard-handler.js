/**
 * clipboard-handler.js - 剪贴板同步
 * 
 * 使用 Clipboard API 监听本地剪贴板变化，
 * 通过 control DataChannel 与远程主机同步
 */

import { buildClipboardAuditReport } from '../../src/api/clipboardAuditModel.js'
import { userApi } from '../../src/api/userApi.js'

export class ClipboardHandler {
    /**
     * @param {DataChannelHandler} dataChannelHandler 
     * @param {object} [options]
     * @param {string} [options.deviceId]  device the sync is against, for the audit trail
     * @param {Function} [options.now]     transfer-id clock, injectable for tests
     */
    constructor(dataChannelHandler, options = {}) {
        this.dcHandler = dataChannelHandler;
        this._enabled = false;
        this._lastClipboardText = '';
        this._pollInterval = null;
        this._deviceId = options.deviceId || '';
        this._now = options.now || (() => Date.now());
        // Injected rather than imported at call time so a test can observe what
        // would be sent without a live control plane behind it.
        this._reportAudit = options.reportAudit || ((body) => userApi.recordClipboardAudit(body));
        // One transfer id per direction, so a burst of syncs is reported as the
        // distinct transfers they are rather than collapsing into one row.
        this._nextTransferId = { upload: 0, download: 0 };
    }

    /**
     * 启用剪贴板同步
     */
    enable() {
        if (this._enabled) return;
        this._enabled = true;

        // 监听来自 Host 的剪贴板事件
        this.dcHandler.addEventListener('clipboardEvent', this._onRemoteClipboard.bind(this));

        // 定时轮询本地剪贴板 (Clipboard API 没有 change 事件)
        this._pollInterval = setInterval(() => this._pollClipboard(), 1000);
    }

    /**
     * 禁用剪贴板同步
     */
    disable() {
        if (!this._enabled) return;
        this._enabled = false;

        if (this._pollInterval) {
            clearInterval(this._pollInterval);
            this._pollInterval = null;
        }
    }

    /**
     * 轮询本地剪贴板
     * @private
     */
    async _pollClipboard() {
        try {
            // 需要文档有焦点才能读取剪贴板
            if (!document.hasFocus()) return;

            const text = await navigator.clipboard.readText();
            if (text && text !== this._lastClipboardText) {
                this._lastClipboardText = text;
                this._sendToRemote(text);
            }
        } catch (e) {
            // 剪贴板权限被拒绝，静默处理
        }
    }

    /**
     * 发送剪贴板内容到远程
     * @private
     */
    _sendToRemote(text) {
        const encoder = new TextEncoder();
        this.dcHandler.sendClipboardEvent({
            mimeType: 'text/plain',
            data: encoder.encode(text),
        });
        this._report(text, 'upload');
    }

    /**
     * 处理来自远程的剪贴板事件
     * @private
     */
    async _onRemoteClipboard(event) {
        const { mimeType, data } = event.detail;
        
        if (mimeType === 'text/plain' && data) {
            const text = new TextDecoder().decode(data);
            this._lastClipboardText = text; // 防止回环
            
            try {
                await navigator.clipboard.writeText(text);
                this._report(text, 'download');
            } catch (e) {
                console.warn('[Clipboard] Failed to write to clipboard:', e);
                // A sync that failed is still a fact worth recording: the bytes
                // did cross the data channel even though the local write did
                // not land, and an audit trail that only shows successes cannot
                // answer "did anything reach this machine".
                this._report(text, 'download', 'failed');
            }
        }
    }

    /**
     * QD-909: report that a clipboard sync happened, without reporting what was
     * in it. The builder is the boundary - it takes the text only to measure it
     * and returns an object with no field that could hold it, so the report
     * cannot become a credential store even by accident.
     * @private
     */
    _report(text, direction, status = 'completed') {
        if (!this._deviceId) return;
        this._nextTransferId[direction] = (this._nextTransferId[direction] || 0) + 1;
        const report = buildClipboardAuditReport({
            deviceId: this._deviceId,
            transferId: `clip-${direction}-${this._now()}-${this._nextTransferId[direction]}`,
            direction,
            status,
            mimeType: 'text/plain',
            text: typeof text === 'string' ? text : '',
        });
        if (!report) return;
        // Best effort, like every other audit report on this path: a failed
        // report must never break the sync the user is actually doing.
        Promise.resolve(this._reportAudit(report)).catch(() => {});
    }

    /**
     * 清理
     */
    destroy() {
        this.disable();
    }
}
