/**
 * Tracks which screen-wall tile owns audio playback.
 *
 * The wall page owns the iframe messages. This selector only decides which
 * device should be enabled now and which previous device must be disabled.
 */
export class WallAudioSelector {
    constructor() {
        this.activeId = '';
    }

    setActive(deviceId) {
        const next = deviceId || '';
        const previous = this.activeId;
        this.activeId = next;
        return previous === next ? '' : previous;
    }

    clear(deviceId = '') {
        if (deviceId && this.activeId !== deviceId) return '';
        return this.setActive('');
    }
}
