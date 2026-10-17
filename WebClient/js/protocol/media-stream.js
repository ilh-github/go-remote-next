export function addTrackOnce(stream, track) {
    if (!stream || !track) return false;
    const tracks = typeof stream.getTracks === 'function' ? stream.getTracks() : [];
    if (tracks.includes(track)) return false;
    stream.addTrack(track);
    return true;
}
