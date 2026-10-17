pragma ComponentBehavior: Bound

import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import QtMultimedia
import QuickDesk 1.0

import "../component"

Item {
    id: root

    required property var mainController

    signal showToast(string message, int toastType)
    signal loginRequested()

    property var manager: mainController && mainController.recordingManager
                          ? mainController.recordingManager : null
    property var cloudManager: mainController && mainController.cloudDeviceManager
                               ? mainController.cloudDeviceManager : null
    property bool isLoggedIn: mainController && mainController.authManager
                              ? mainController.authManager.isLoggedIn : false
    property var thumbnailUrls: ({})
    property var thumbnailPending: ({})
    property int pendingRecordingId: 0
    property string pendingAction: ""
    property int deleteRecordingId: 0
    property string deleteRecordingName: ""
    property bool refreshPending: false
    property string playerError: ""

    function statusLabel(status) {
        if (status === "completed") return qsTr("Completed")
        if (status === "recording") return qsTr("Recording")
        if (status === "failed") return qsTr("Failed")
        return qsTr("Unknown")
    }

    function statusType(status) {
        if (status === "completed") return QDChip.Type.Success
        if (status === "recording") return QDChip.Type.Warning
        if (status === "failed") return QDChip.Type.Error
        return QDChip.Type.Default
    }

    function formatTime(value) {
        if (!value) return "-"
        var parsed = new Date(value)
        return isNaN(parsed.getTime()) ? String(value) : parsed.toLocaleString()
    }

    function formatDuration(seconds) {
        var total = Math.max(0, Math.floor(Number(seconds || 0)))
        var hours = Math.floor(total / 3600)
        var minutes = Math.floor((total % 3600) / 60)
        var remaining = total % 60
        function pad(value) {
            return value < 10 ? "0" + value : String(value)
        }
        return hours > 0
                ? hours + ":" + pad(minutes) + ":" + pad(remaining)
                : minutes + ":" + pad(remaining)
    }

    function formatSize(bytes) {
        var value = Number(bytes || 0)
        if (value < 1024) return value + " B"
        var units = ["KB", "MB", "GB", "TB"]
        var index = 0
        value /= 1024
        while (value >= 1024 && index < units.length - 1) {
            value /= 1024
            ++index
        }
        return (value >= 10 ? value.toFixed(0) : value.toFixed(1))
                + " " + units[index]
    }

    function hasMedia(recording) {
        return recording
                && (!!recording.artifact_url
                    || recording.status === "completed")
    }

    function thumbUrl(recordingId) {
        return root.thumbnailUrls[recordingId] || ""
    }

    function requestThumbnails() {
        if (!root.manager) return
        var pending = Object.assign({}, root.thumbnailPending)
        var urls = Object.assign({}, root.thumbnailUrls)
        for (var i = 0; i < root.manager.recordings.length; ++i) {
            var recording = root.manager.recordings[i]
            var id = Number(recording.id || 0)
            if (id <= 0 || !recording.thumbnail_url
                    || pending[id] || urls[id]) {
                continue
            }
            pending[id] = true
            root.manager.requestThumbnailUrl(id)
        }
        root.thumbnailPending = pending
    }

    function requestMediaAction(recording, action) {
        if (!root.manager || !recording || !root.hasMedia(recording)) {
            root.showToast(qsTr("This recording has no media"), QDToast.Type.Info)
            return
        }
        root.pendingRecordingId = Number(recording.id || 0)
        root.pendingAction = action
        root.playerError = ""
        root.manager.requestArtifactUrl(root.pendingRecordingId)
    }

    function requestDelete(recording) {
        root.deleteRecordingId = Number(recording.id || 0)
        root.deleteRecordingName = recording.device_name
                                     || recording.device_id
                                     || qsTr("Unknown device")
        deleteDialog.show()
    }

    function scheduleRealtimeRefresh() {
        if (!root.manager || !root.isLoggedIn) return
        if (root.manager.busy) {
            root.refreshPending = true
            return
        }
        realtimeRefreshTimer.restart()
    }

    Connections {
        target: root.mainController && root.mainController.authManager
                ? root.mainController.authManager : null

        function onLoginStateChanged() {
            if (root.isLoggedIn) Qt.callLater(root.manager.refresh)
        }
    }

    Connections {
        target: root.cloudManager

        function onSyncMessage(message) {
            if (root.manager) root.manager.handleRealtimeEvent(message)
        }
    }

    Connections {
        target: root.manager

        function onRequestFailed(message) {
            root.showToast(message, QDToast.Type.Error)
        }

        function onRecordingsChanged() {
            root.requestThumbnails()
        }

        function onThumbnailUrlReady(recordingId, url) {
            var pending = Object.assign({}, root.thumbnailPending)
            var urls = Object.assign({}, root.thumbnailUrls)
            delete pending[recordingId]
            if (url) urls[recordingId] = url
            root.thumbnailPending = pending
            root.thumbnailUrls = urls
        }

        function onArtifactUrlReady(recordingId, url) {
            if (recordingId !== root.pendingRecordingId || !url) return
            if (root.pendingAction === "open") {
                Qt.openUrlExternally(url)
                return
            }
            mediaPlayer.stop()
            mediaPlayer.source = url
            playerDialog.show()
            mediaPlayer.play()
        }

        function onRecordingDeleted(recordingId) {
            deleteDialog.hide()
            var urls = Object.assign({}, root.thumbnailUrls)
            delete urls[recordingId]
            root.thumbnailUrls = urls
            root.showToast(qsTr("Recording deleted"), QDToast.Type.Success)
        }

        function onRealtimeRefreshRequested() {
            root.scheduleRealtimeRefresh()
        }

        function onBusyChanged() {
            if (root.manager && !root.manager.busy && root.refreshPending) {
                root.scheduleRealtimeRefresh()
            }
        }
    }

    Timer {
        id: realtimeRefreshTimer
        interval: 250
        repeat: false
        onTriggered: {
            root.refreshPending = false
            root.manager.refresh()
        }
    }

    Component.onCompleted: {
        if (root.isLoggedIn) Qt.callLater(root.manager.refresh)
    }

    MediaPlayer {
        id: mediaPlayer
        videoOutput: videoOutput
        audioOutput: AudioOutput {
            volume: 0.8
        }
        onErrorOccurred: function(error, errorString) {
            root.playerError = errorString || qsTr("Playback failed")
        }
    }

    Item {
        anchors.fill: parent
        visible: !root.isLoggedIn

        ColumnLayout {
            anchors.centerIn: parent
            spacing: Theme.spacingLarge

            Text {
                text: FluentIconGlyph.videoGlyph
                font.family: "Segoe Fluent Icons"
                font.pixelSize: 48
                color: Theme.textDisabled
                Layout.alignment: Qt.AlignHCenter
            }

            Text {
                text: qsTr("Please login to view recordings")
                font.pixelSize: Theme.fontSizeMedium
                color: Theme.textSecondary
            }

            QDButton {
                text: qsTr("Login")
                buttonType: QDButton.Type.Primary
                Layout.alignment: Qt.AlignHCenter
                onClicked: root.loginRequested()
            }
        }
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: Theme.spacingLarge
        visible: root.isLoggedIn
        spacing: Theme.spacingMedium

        RowLayout {
            Layout.fillWidth: true
            spacing: Theme.spacingMedium

            ColumnLayout {
                Layout.fillWidth: true
                spacing: 2

                Text {
                    text: qsTr("Session Recordings")
                    font.pixelSize: Theme.fontSizeHeading
                    font.weight: Font.Bold
                    color: Theme.text
                }

                Text {
                    text: qsTr("Review, play, download and delete recorded remote sessions.")
                    font.pixelSize: Theme.fontSizeSmall
                    color: Theme.textSecondary
                    wrapMode: Text.WordWrap
                }
            }

            QDChip {
                text: qsTr("%1 recordings")
                      .arg(root.manager ? root.manager.recordings.length : 0)
                chipType: QDChip.Type.Info
                outlined: true
            }

            QDButton {
                text: qsTr("Refresh")
                iconText: FluentIconGlyph.syncGlyph
                buttonType: QDButton.Type.Secondary
                loading: root.manager ? root.manager.busy : false
                enabled: !!root.manager && !root.manager.busy
                onClicked: root.manager.refresh()
            }
        }

        Rectangle {
            Layout.fillWidth: true
            visible: root.manager && root.manager.lastError.length > 0
            Layout.preferredHeight: errorText.implicitHeight + Theme.spacingMedium * 2
            color: Qt.rgba(Theme.error.r, Theme.error.g, Theme.error.b, 0.08)
            radius: Theme.radiusSmall

            Text {
                id: errorText
                anchors.fill: parent
                anchors.margins: Theme.spacingMedium
                text: root.manager ? root.manager.lastError : ""
                color: Theme.error
                wrapMode: Text.WordWrap
                font.pixelSize: Theme.fontSizeSmall
            }
        }

        Rectangle {
            Layout.fillWidth: true
            Layout.fillHeight: true
            color: Theme.surface
            radius: Theme.radiusMedium
            border.width: Theme.borderWidthThin
            border.color: Theme.border

            Flickable {
                anchors.fill: parent
                anchors.margins: Theme.spacingMedium
                contentWidth: width
                contentHeight: recordingsColumn.implicitHeight
                clip: true

                ScrollBar.vertical: QDScrollBar {}

                ColumnLayout {
                    id: recordingsColumn
                    width: parent.width
                    spacing: Theme.spacingSmall

                    Text {
                        visible: !root.manager
                                 || root.manager.recordings.length === 0
                        Layout.fillWidth: true
                        Layout.topMargin: Theme.spacingXLarge
                        horizontalAlignment: Text.AlignHCenter
                        text: root.manager && root.manager.busy
                              ? qsTr("Loading recordings...")
                              : qsTr("No recorded sessions yet.")
                        font.pixelSize: Theme.fontSizeMedium
                        color: Theme.textSecondary
                    }

                    Repeater {
                        model: root.manager ? root.manager.recordings : []

                        Rectangle {
                            id: recordingCard
                            required property var modelData

                            Layout.fillWidth: true
                            Layout.preferredHeight: recordingContent.implicitHeight
                                                    + Theme.spacingLarge * 2
                            color: Theme.surfaceVariant
                            radius: Theme.radiusMedium
                            border.width: Theme.borderWidthThin
                            border.color: recordingCard.modelData.status === "failed"
                                          ? Qt.rgba(Theme.error.r, Theme.error.g,
                                                    Theme.error.b, 0.45)
                                          : Theme.border

                            RowLayout {
                                id: recordingContent
                                anchors.fill: parent
                                anchors.margins: Theme.spacingLarge
                                spacing: Theme.spacingLarge

                                Rectangle {
                                    Layout.preferredWidth: 128
                                    Layout.preferredHeight: 72
                                    radius: Theme.radiusMedium
                                    color: Theme.surface
                                    clip: true

                                    Image {
                                        anchors.fill: parent
                                        source: root.thumbUrl(
                                                    Number(recordingCard.modelData.id || 0))
                                        fillMode: Image.PreserveAspectCrop
                                        asynchronous: true
                                    }

                                    Text {
                                        anchors.centerIn: parent
                                        visible: !root.thumbUrl(
                                                    Number(recordingCard.modelData.id || 0))
                                        text: FluentIconGlyph.videoGlyph
                                        font.family: "Segoe Fluent Icons"
                                        font.pixelSize: 28
                                        color: Theme.textDisabled
                                    }
                                }

                                ColumnLayout {
                                    Layout.fillWidth: true
                                    spacing: 2

                                    RowLayout {
                                        spacing: Theme.spacingSmall

                                        Text {
                                            text: recordingCard.modelData.device_name
                                                  || recordingCard.modelData.device_id
                                                  || qsTr("Unknown device")
                                            font.pixelSize: Theme.fontSizeMedium
                                            font.weight: Font.DemiBold
                                            color: Theme.text
                                        }

                                        QDChip {
                                            text: root.statusLabel(
                                                      recordingCard.modelData.status)
                                            chipType: root.statusType(
                                                          recordingCard.modelData.status)
                                        }
                                    }

                                    Text {
                                        Layout.fillWidth: true
                                        text: qsTr("%1 · %2 · %3 · %4")
                                              .arg(root.formatTime(
                                                   recordingCard.modelData.started_at))
                                              .arg(root.formatDuration(
                                                   recordingCard.modelData.duration_seconds))
                                              .arg(String(
                                                   recordingCard.modelData.format || "mp4").toUpperCase())
                                              .arg(root.formatSize(
                                                   recordingCard.modelData.size_bytes))
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.textSecondary
                                        wrapMode: Text.WordWrap
                                    }

                                    Text {
                                        Layout.fillWidth: true
                                        text: qsTr("Session: %1 · Device: %2")
                                              .arg(recordingCard.modelData.session_id
                                                   || "-")
                                              .arg(recordingCard.modelData.device_id
                                                   || "-")
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.textSecondary
                                        elide: Text.ElideRight
                                    }
                                }

                                QDButton {
                                    text: qsTr("Play")
                                    iconText: FluentIconGlyph.playGlyph
                                    buttonType: QDButton.Type.Primary
                                    enabled: root.hasMedia(recordingCard.modelData)
                                    onClicked: root.requestMediaAction(
                                                   recordingCard.modelData, "play")
                                }

                                QDButton {
                                    text: qsTr("Open externally")
                                    buttonType: QDButton.Type.Secondary
                                    enabled: root.hasMedia(recordingCard.modelData)
                                    onClicked: root.requestMediaAction(
                                                   recordingCard.modelData, "open")
                                }

                                QDButton {
                                    text: qsTr("Delete")
                                    iconText: FluentIconGlyph.deleteGlyph
                                    buttonType: QDButton.Type.Danger
                                    onClicked: root.requestDelete(
                                                   recordingCard.modelData)
                                }
                            }
                        }
                    }

                    QDButton {
                        visible: root.manager && root.manager.hasMore
                        Layout.alignment: Qt.AlignHCenter
                        Layout.topMargin: Theme.spacingMedium
                        text: root.manager && root.manager.busy
                              ? qsTr("Loading...") : qsTr("Load more")
                        buttonType: QDButton.Type.Secondary
                        enabled: root.manager && !root.manager.busy
                        onClicked: root.manager.loadMore()
                    }
                }
            }
        }
    }

    QDDialog {
        id: playerDialog
        title: qsTr("Recording playback")
        dialogWidth: Math.min(960, root.width - 48)
        dialogHeight: Math.min(680, root.height - 48)
        closeOnOverlay: false
        onClosed: {
            mediaPlayer.stop()
            mediaPlayer.source = ""
        }

        ColumnLayout {
            anchors.fill: parent
            spacing: Theme.spacingMedium

            Rectangle {
                Layout.fillWidth: true
                Layout.fillHeight: true
                color: "#000000"
                radius: Theme.radiusMedium
                clip: true

                VideoOutput {
                    id: videoOutput
                    anchors.fill: parent
                }

                Text {
                    anchors.centerIn: parent
                    visible: root.playerError.length > 0
                    text: root.playerError
                    color: Theme.error
                    wrapMode: Text.WordWrap
                    width: parent.width - Theme.spacingXLarge * 2
                    horizontalAlignment: Text.AlignHCenter
                }
            }

            RowLayout {
                Layout.fillWidth: true

                QDButton {
                    text: mediaPlayer.playbackState === MediaPlayer.PlayingState
                          ? qsTr("Pause") : qsTr("Play")
                    iconText: mediaPlayer.playbackState === MediaPlayer.PlayingState
                              ? FluentIconGlyph.pauseGlyph
                              : FluentIconGlyph.playGlyph
                    buttonType: QDButton.Type.Primary
                    onClicked: {
                        if (mediaPlayer.playbackState === MediaPlayer.PlayingState)
                            mediaPlayer.pause()
                        else
                            mediaPlayer.play()
                    }
                }

                QDButton {
                    text: qsTr("Stop")
                    buttonType: QDButton.Type.Secondary
                    onClicked: mediaPlayer.stop()
                }

                Item { Layout.fillWidth: true }

                QDButton {
                    text: qsTr("Close")
                    buttonType: QDButton.Type.Ghost
                    onClicked: playerDialog.hide()
                }
            }
        }
    }

    QDDialog {
        id: deleteDialog
        title: qsTr("Delete recording")
        dialogWidth: Math.min(500, root.width - 48)
        dialogHeight: 230

        ColumnLayout {
            anchors.fill: parent
            spacing: Theme.spacingLarge

            Text {
                Layout.fillWidth: true
                text: qsTr("Delete the recording for “%1”? The media and preview will be removed.")
                      .arg(root.deleteRecordingName)
                color: Theme.text
                wrapMode: Text.WordWrap
            }

            Item { Layout.fillHeight: true }

            RowLayout {
                Layout.fillWidth: true

                Item { Layout.fillWidth: true }

                QDButton {
                    text: qsTr("Cancel")
                    buttonType: QDButton.Type.Secondary
                    onClicked: deleteDialog.hide()
                }

                QDButton {
                    text: qsTr("Delete")
                    buttonType: QDButton.Type.Danger
                    onClicked: {
                        deleteDialog.hide()
                        root.manager.deleteRecording(root.deleteRecordingId)
                    }
                }
            }
        }
    }
}
