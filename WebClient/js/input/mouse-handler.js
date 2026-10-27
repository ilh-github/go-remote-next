/**
 * mouse-handler.js - 鼠标输入处理
 * 
 * 参照 QuickDesk/qml/quickdeskcomponent/RemoteDesktopView.qml
 * 处理鼠标事件的坐标映射和 protobuf 编码
 */

import { MouseButton } from '../protocol/protobuf-messages.js';

export class MouseHandler {
    /**
     * @param {HTMLVideoElement} videoElement - 视频元素
     * @param {DataChannelHandler} dataChannelHandler - DataChannel 处理器
     * @param {HTMLElement} focusElement - Element receiving keyboard input
     */
    constructor(videoElement, dataChannelHandler, focusElement = videoElement) {
        this.videoElement = videoElement;
        this.dcHandler = dataChannelHandler;
        this.focusElement = focusElement;
        this.remoteWidth = 0;
        this.remoteHeight = 0;
        this._enabled = false;
        this._handlers = {};
        this._capturedPointerId = null;
    }

    /**
     * 设置远程桌面分辨率
     * @param {number} width 
     * @param {number} height 
     */
    setRemoteResolution(width, height) {
        this.remoteWidth = width;
        this.remoteHeight = height;
    }

    /**
     * 启用鼠标捕获
     */
    enable() {
        if (this._enabled) return;
        this._enabled = true;

        const el = this.videoElement;

        this._handlers.mousemove = (e) => this._onMouseMove(e);
        this._handlers.mousedown = (e) => this._onMouseButton(e, true);
        this._handlers.mouseup = (e) => this._onMouseButton(e, false);
        this._handlers.wheel = (e) => this._onWheel(e);
        this._handlers.contextmenu = (e) => e.preventDefault();

        el.addEventListener('mousemove', this._handlers.mousemove);
        el.addEventListener('mousedown', this._handlers.mousedown);
        el.addEventListener('mouseup', this._handlers.mouseup);
        el.addEventListener('wheel', this._handlers.wheel, { passive: false });
        el.addEventListener('contextmenu', this._handlers.contextmenu);
    }

    /**
     * 禁用鼠标捕获
     */
    disable() {
        if (!this._enabled) return;
        this._enabled = false;

        // A capture outliving its handler retargets every later mouse event to
        // an element that no longer listens, so the remote cursor would freeze
        // while the page still looked live.
        if (this._capturedPointerId !== null && this._capturedPointerId !== undefined) {
            try {
                this.videoElement.releasePointerCapture?.(this._capturedPointerId);
            } catch (_) {
                // The element may already be detached.
            }
            this._capturedPointerId = null;
        }

        const el = this.videoElement;
        el.removeEventListener('mousemove', this._handlers.mousemove);
        el.removeEventListener('mousedown', this._handlers.mousedown);
        el.removeEventListener('mouseup', this._handlers.mouseup);
        el.removeEventListener('wheel', this._handlers.wheel);
        el.removeEventListener('contextmenu', this._handlers.contextmenu);
    }

    /**
     * 将浏览器坐标映射到远程桌面坐标
     * 处理 object-fit: contain 的黑边偏移
     * 
     * @param {MouseEvent} event 
     * @returns {{x: number, y: number}|null}
     */
    _mapCoordinates(event) {
        if (this.remoteWidth <= 0 || this.remoteHeight <= 0) return null;

        const rect = this.videoElement.getBoundingClientRect();
        const videoWidth = this.videoElement.videoWidth || this.remoteWidth;
        const videoHeight = this.videoElement.videoHeight || this.remoteHeight;

        if (videoWidth <= 0 || videoHeight <= 0) return null;

        // 计算 object-fit: contain 下的实际渲染区域
        const containerAspect = rect.width / rect.height;
        const videoAspect = videoWidth / videoHeight;

        let renderWidth, renderHeight, offsetX, offsetY;

        if (containerAspect > videoAspect) {
            // 容器更宽，左右有黑边
            renderHeight = rect.height;
            renderWidth = renderHeight * videoAspect;
            offsetX = (rect.width - renderWidth) / 2;
            offsetY = 0;
        } else {
            // 容器更高，上下有黑边
            renderWidth = rect.width;
            renderHeight = renderWidth / videoAspect;
            offsetX = 0;
            offsetY = (rect.height - renderHeight) / 2;
        }

        // 鼠标在容器内的坐标
        const localX = event.clientX - rect.left;
        const localY = event.clientY - rect.top;

        // 减去黑边偏移
        const contentX = localX - offsetX;
        const contentY = localY - offsetY;

        // 检查是否在视频内容区域内
        if (contentX < 0 || contentX > renderWidth || contentY < 0 || contentY > renderHeight) {
            return null;
        }

        // 映射到远程桌面坐标
        const remoteX = Math.round((contentX / renderWidth) * this.remoteWidth);
        const remoteY = Math.round((contentY / renderHeight) * this.remoteHeight);

        return {
            x: Math.max(0, Math.min(remoteX, this.remoteWidth - 1)),
            y: Math.max(0, Math.min(remoteY, this.remoteHeight - 1)),
        };
    }

    /**
     * 浏览器鼠标按钮到 protobuf 枚举的映射
     * @param {number} browserButton 
     * @returns {number}
     */
    _mapButton(browserButton) {
        switch (browserButton) {
            case 0: return MouseButton.BUTTON_LEFT;
            case 1: return MouseButton.BUTTON_MIDDLE;
            case 2: return MouseButton.BUTTON_RIGHT;
            case 3: return MouseButton.BUTTON_BACK;
            case 4: return MouseButton.BUTTON_FORWARD;
            default: return MouseButton.BUTTON_UNDEFINED;
        }
    }

    /**
     * 鼠标移动事件
     * @private
     */
    _onMouseMove(event) {
        const coords = this._mapCoordinates(event);
        if (!coords) return;

        this.dcHandler.sendMouseEvent({
            x: coords.x,
            y: coords.y,
        });
    }

    /**
     * 鼠标按钮事件
     * @private
     */
    _onMouseButton(event, isDown) {
        if (isDown && this.focusElement && document.activeElement !== this.focusElement) {
            try {
                this.focusElement.focus({ preventScroll: true });
            } catch (_) {
                this.focusElement.focus();
            }
        }

        event.preventDefault();
        const coords = this._mapCoordinates(event);
        if (!coords) return;

        // Capturing the pointer on press is what keeps a drag alive once the
        // cursor wanders off the video. Without it the release lands on
        // whatever is underneath, this handler never sees it, and the remote
        // machine is left holding the button down until some later click
        // happens to land back on the video.
        if (isDown && event.pointerId !== undefined) {
            try {
                this.videoElement.setPointerCapture?.(event.pointerId);
                this._capturedPointerId = event.pointerId;
            } catch (_) {
                // A pointer that has already gone away cannot be captured; the
                // press itself is still valid and must go through.
            }
        } else if (!isDown && this._capturedPointerId !== null && this._capturedPointerId !== undefined) {
            try {
                this.videoElement.releasePointerCapture?.(this._capturedPointerId);
            } catch (_) {
                // Releasing an uncaptured pointer is a no-op on the host side.
            }
            this._capturedPointerId = null;
        }

        this.dcHandler.sendMouseEvent({
            x: coords.x,
            y: coords.y,
            button: this._mapButton(event.button),
            buttonDown: isDown,
        });
    }

    /**
     * 滚轮事件
     * @private
     */
    _onWheel(event) {
        event.preventDefault();
        const coords = this._mapCoordinates(event);
        if (!coords) return;

        // deltaMode: 0=pixels, 1=lines, 2=pages
        let deltaX = event.deltaX;
        let deltaY = event.deltaY;

        if (event.deltaMode === 1) {
            // lines -> pixels
            deltaX *= 40;
            deltaY *= 40;
        } else if (event.deltaMode === 2) {
            // pages -> pixels
            deltaX *= 800;
            deltaY *= 800;
        }

        // ticks (通常 1 tick = 120 单位)
        const ticksX = deltaX / 120;
        const ticksY = deltaY / 120;

        this.dcHandler.sendMouseEvent({
            x: coords.x,
            y: coords.y,
            wheelDeltaX: -deltaX,
            wheelDeltaY: -deltaY,
            wheelTicksX: -ticksX,
            wheelTicksY: -ticksY,
        });
    }

    /**
     * 清理
     */
    destroy() {
        this.disable();
    }
}
