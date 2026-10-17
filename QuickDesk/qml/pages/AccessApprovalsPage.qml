pragma ComponentBehavior: Bound

import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import QuickDesk 1.0

import "../component"

Item {
    id: root

    required property var mainController

    signal showToast(string message, int toastType)
    signal loginRequested()

    property var manager: mainController && mainController.accessApprovalManager
                          ? mainController.accessApprovalManager : null
    property var cloudManager: mainController && mainController.cloudDeviceManager
                               ? mainController.cloudDeviceManager : null
    property bool isLoggedIn: mainController && mainController.authManager
                              ? mainController.authManager.isLoggedIn : false
    property string selectedDeviceId: ""
    property string statusFilter: "pending"
    property var decisionApproval: null
    property bool decisionApprove: true
    property var deniedSources: root.manager && root.manager.summary
                                && root.manager.summary.denied_sources
                                ? root.manager.summary.denied_sources : []
    property var approvalStatuses: ["pending", "approved", "denied", "expired", "consumed"]
    property var filterOptions: [
        { value: "pending", label: qsTr("Pending") },
        { value: "approved,denied", label: qsTr("Decided") },
        { value: "expired", label: qsTr("Expired") },
        { value: "consumed", label: qsTr("Consumed") },
        { value: "pending,approved,denied,expired,consumed", label: qsTr("All") }
    ]

    function rebuildDevices() {
        deviceModel.clear()
        var devices = root.cloudManager ? root.cloudManager.myDevices : []
        for (var i = 0; i < devices.length; ++i) {
            var device = devices[i]
            deviceModel.append({
                value: device.device_id || "",
                text: device.remark || device.device_name || device.device_id || qsTr("Device")
            })
        }
        if (!selectedDeviceId && deviceModel.count > 0) {
            selectedDeviceId = deviceModel.get(0).value
            deviceCombo.currentIndex = 0
            Qt.callLater(root.refresh)
        } else if (selectedDeviceId) {
            var index = deviceCombo.indexOfValue(selectedDeviceId)
            if (index >= 0) deviceCombo.currentIndex = index
        }
    }

    function refresh() {
        if (!root.manager || !root.isLoggedIn || !selectedDeviceId) return
        root.manager.refresh(selectedDeviceId, statusFilter)
    }

    function setFilter(value) {
        if (statusFilter === value) return
        statusFilter = value
        refresh()
    }

    function formatTime(value) {
        if (!value) return "-"
        var parsed = new Date(value)
        return isNaN(parsed.getTime()) ? String(value) : parsed.toLocaleString()
    }

    function statusLabel(status) {
        if (status === "pending") return qsTr("Pending")
        if (status === "approved") return qsTr("Approved")
        if (status === "denied") return qsTr("Denied")
        if (status === "expired") return qsTr("Expired")
        if (status === "consumed") return qsTr("Consumed")
        return status || qsTr("Unknown")
    }

    function statusChipType(status) {
        if (status === "approved") return QDChip.Type.Success
        if (status === "denied") return QDChip.Type.Error
        if (status === "expired") return QDChip.Type.Warning
        if (status === "consumed") return QDChip.Type.Info
        return QDChip.Type.Primary
    }

    function sourceLabel(source) {
        if (source === "device_code") return qsTr("Device code")
        if (source === "grant") return qsTr("Guest grant")
        return source || qsTr("Unknown source")
    }

    function summaryCount(status) {
        if (!root.manager || !root.manager.summary || !root.manager.summary.by_status) return 0
        return Number(root.manager.summary.by_status[status] || 0)
    }

    function openDecision(approval, approve) {
        decisionApproval = approval
        decisionApprove = approve
        decisionReason.text = ""
        decisionDialog.title = approve ? qsTr("Approve access") : qsTr("Deny access")
        decisionDialog.show()
    }

    function submitDecision() {
        if (!root.manager || !decisionApproval) return
        root.manager.decide(Number(decisionApproval.id), decisionApprove, decisionReason.text.trim())
    }

    Connections {
        target: root.cloudManager

        function onMyDevicesChanged() {
            root.rebuildDevices()
        }

        function onSyncMessage(message) {
            var type = message && message.type ? message.type : ""
            if (type !== "device.access_approval.requested"
                    && type !== "device.access_approval.decided") {
                return
            }
            var data = message.data || {}
            if (!data.device_id || data.device_id === root.selectedDeviceId) {
                if (type === "device.access_approval.requested") {
                    root.showToast(qsTr("A new access request is waiting"), QDToast.Type.Info)
                }
                root.refresh()
            }
        }
    }

    Connections {
        target: root.manager

        function onRequestFailed(message) {
            root.showToast(message, QDToast.Type.Error)
        }

        function onApprovalDecided(approvalId, status) {
            decisionDialog.hide()
            root.showToast(root.decisionApprove
                           ? qsTr("Access approved")
                           : qsTr("Access denied"), QDToast.Type.Success)
            root.refresh()
        }

        function onDecisionFailed(approvalId, httpStatus, currentStatus, message) {
            if (httpStatus === 409) {
                root.showToast(qsTr("The request was already %1")
                               .arg(root.statusLabel(currentStatus)), QDToast.Type.Warning)
            } else {
                root.showToast(message, QDToast.Type.Error)
            }
            root.refresh()
        }
    }

    Timer {
        interval: 15000
        repeat: true
        running: root.isLoggedIn && root.manager && !root.manager.busy
        onTriggered: root.refresh()
    }

    Component.onCompleted: {
        Qt.callLater(root.rebuildDevices)
    }

    ListModel {
        id: deviceModel
    }

    Item {
        anchors.fill: parent
        visible: !root.isLoggedIn

        ColumnLayout {
            anchors.centerIn: parent
            spacing: Theme.spacingLarge

            Text {
                text: FluentIconGlyph.shieldGlyph
                font.family: "Segoe Fluent Icons"
                font.pixelSize: 48
                color: Theme.textDisabled
                Layout.alignment: Qt.AlignHCenter
            }

            Text {
                text: qsTr("Please login to review access requests")
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
                    text: qsTr("Access Approvals")
                    font.pixelSize: Theme.fontSizeHeading
                    font.weight: Font.Bold
                    color: Theme.text
                }

                Text {
                    text: qsTr("Review second-factor access requests and repeat denials.")
                    font.pixelSize: Theme.fontSizeSmall
                    color: Theme.textSecondary
                    wrapMode: Text.WordWrap
                }
            }

            QDButton {
                text: qsTr("Refresh")
                iconText: FluentIconGlyph.syncGlyph
                buttonType: QDButton.Type.Secondary
                loading: root.manager ? root.manager.busy : false
                enabled: !!root.manager && !root.manager.busy
                onClicked: root.refresh()
            }
        }

        Rectangle {
            Layout.fillWidth: true
            Layout.preferredHeight: filterLayout.implicitHeight + Theme.spacingLarge * 2
            color: Theme.surface
            radius: Theme.radiusMedium
            border.width: Theme.borderWidthThin
            border.color: Theme.border

            RowLayout {
                id: filterLayout
                anchors.fill: parent
                anchors.margins: Theme.spacingLarge
                spacing: Theme.spacingMedium

                QDComboBox {
                    id: deviceCombo
                    Layout.preferredWidth: 260
                    model: deviceModel
                    textRole: "text"
                    valueRole: "value"
                    onActivated: {
                        root.selectedDeviceId = selectedValue
                        root.refresh()
                    }
                }

                Repeater {
                    model: root.filterOptions

                    QDButton {
                        required property var modelData
                        text: modelData.label
                        buttonType: root.statusFilter === modelData.value
                                     ? QDButton.Type.Primary : QDButton.Type.Secondary
                        onClicked: root.setFilter(modelData.value)
                    }
                }

                Item { Layout.fillWidth: true }
            }
        }

        RowLayout {
            Layout.fillWidth: true
            spacing: Theme.spacingSmall

            Repeater {
                model: root.approvalStatuses

                Rectangle {
                    id: statusCell
                    required property var modelData
                    Layout.fillWidth: true
                    Layout.preferredHeight: 72
                    color: Theme.surface
                    radius: Theme.radiusMedium
                    border.width: Theme.borderWidthThin
                    border.color: Theme.border

                    Column {
                        anchors.centerIn: parent
                        spacing: 2

                        Text {
                            anchors.horizontalCenter: parent.horizontalCenter
                            text: root.summaryCount(statusCell.modelData)
                            font.pixelSize: Theme.fontSizeXLarge
                            font.weight: Font.Bold
                            color: statusCell.modelData === "denied" ? Theme.error : Theme.text
                        }

                        Text {
                            anchors.horizontalCenter: parent.horizontalCenter
                            text: root.statusLabel(statusCell.modelData)
                            font.pixelSize: Theme.fontSizeSmall
                            color: Theme.textSecondary
                        }
                    }
                }
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
            visible: root.deniedSources.length > 0
            Layout.preferredHeight: deniedLayout.implicitHeight + Theme.spacingLarge * 2
            color: Qt.rgba(Theme.error.r, Theme.error.g, Theme.error.b, 0.05)
            radius: Theme.radiusMedium
            border.width: Theme.borderWidthThin
            border.color: Qt.rgba(Theme.error.r, Theme.error.g, Theme.error.b, 0.3)

            ColumnLayout {
                id: deniedLayout
                anchors.fill: parent
                anchors.margins: Theme.spacingLarge
                spacing: Theme.spacingSmall

                Text {
                    text: qsTr("Repeatedly denied sources")
                    font.pixelSize: Theme.fontSizeMedium
                    font.weight: Font.DemiBold
                    color: Theme.text
                }

                Flow {
                    Layout.fillWidth: true
                    spacing: Theme.spacingSmall

                    Repeater {
                        model: root.deniedSources

                        QDChip {
                            required property var modelData
                            text: (modelData.client_ip || qsTr("Unknown IP"))
                                  + " · " + Number(modelData.denials || 0)
                            chipType: QDChip.Type.Error
                            outlined: true
                        }
                    }
                }

                Text {
                    visible: root.manager && root.manager.summary
                    text: qsTr("Deny cooldown: %1 s (x%2, max %3 s)")
                          .arg(root.manager && root.manager.summary
                               ? Number(root.manager.summary.denial_cooldown_seconds || 0) : 0)
                          .arg(root.manager && root.manager.summary
                               ? Number(root.manager.summary.denial_cooldown_multiplier || 1) : 1)
                          .arg(root.manager && root.manager.summary
                               ? Number(root.manager.summary.denial_cooldown_max_seconds || 0) : 0)
                    font.pixelSize: Theme.fontSizeSmall
                    color: Theme.textSecondary
                }
            }
        }

        Text {
            text: qsTr("Approval queue (%1)")
                  .arg(root.manager ? root.manager.approvals.length : 0)
            font.pixelSize: Theme.fontSizeLarge
            font.weight: Font.DemiBold
            color: Theme.text
        }

        Flickable {
            Layout.fillWidth: true
            Layout.fillHeight: true
            contentWidth: width
            contentHeight: approvalsColumn.implicitHeight
            clip: true

            ScrollBar.vertical: QDScrollBar {}

            ColumnLayout {
                id: approvalsColumn
                width: parent.width
                spacing: Theme.spacingSmall

                Text {
                    visible: !root.manager || root.manager.approvals.length === 0
                    Layout.fillWidth: true
                    Layout.topMargin: Theme.spacingXLarge
                    horizontalAlignment: Text.AlignHCenter
                    text: root.manager && root.manager.busy
                          ? qsTr("Loading access requests...")
                          : qsTr("No access requests match this filter.")
                    font.pixelSize: Theme.fontSizeMedium
                    color: Theme.textSecondary
                }

                Repeater {
                    model: root.manager ? root.manager.approvals : []

                    Rectangle {
                        id: approvalCard
                        required property var modelData
                        Layout.fillWidth: true
                        Layout.preferredHeight: approvalRow.implicitHeight + Theme.spacingLarge * 2
                        color: Theme.surface
                        radius: Theme.radiusMedium
                        border.width: Theme.borderWidthThin
                        border.color: approvalCard.modelData.status === "pending"
                                      ? Qt.rgba(Theme.primary.r, Theme.primary.g, Theme.primary.b, 0.5)
                                      : Theme.border

                        RowLayout {
                            id: approvalRow
                            anchors.fill: parent
                            anchors.margins: Theme.spacingLarge
                            spacing: Theme.spacingLarge

                            ColumnLayout {
                                Layout.fillWidth: true
                                spacing: Theme.spacingXSmall

                                RowLayout {
                                    spacing: Theme.spacingSmall

                                    QDChip {
                                        text: root.statusLabel(approvalCard.modelData.status)
                                        chipType: root.statusChipType(approvalCard.modelData.status)
                                    }

                                    Text {
                                        text: "#" + Number(approvalCard.modelData.id || 0)
                                              + " · " + root.sourceLabel(approvalCard.modelData.source)
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.textSecondary
                                    }
                                }

                                Text {
                                    Layout.fillWidth: true
                                    text: (approvalCard.modelData.client_ip || qsTr("Unknown IP"))
                                          + " · " + (approvalCard.modelData.client_id || qsTr("Unknown client"))
                                    font.pixelSize: Theme.fontSizeMedium
                                    font.weight: Font.DemiBold
                                    color: Theme.text
                                    elide: Text.ElideRight
                                }

                                Text {
                                    Layout.fillWidth: true
                                    text: qsTr("Requested %1 · Expires %2")
                                          .arg(root.formatTime(approvalCard.modelData.created_at))
                                          .arg(root.formatTime(approvalCard.modelData.expires_at))
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                    wrapMode: Text.WordWrap
                                }

                                Text {
                                    visible: !!approvalCard.modelData.decided_at
                                    Layout.fillWidth: true
                                    text: qsTr("Decided %1").arg(root.formatTime(approvalCard.modelData.decided_at))
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                }

                                Text {
                                    visible: !!approvalCard.modelData.reason
                                    Layout.fillWidth: true
                                    text: qsTr("Reason: %1").arg(approvalCard.modelData.reason || "")
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                    wrapMode: Text.WordWrap
                                }
                            }

                            ColumnLayout {
                                visible: approvalCard.modelData.status === "pending"
                                spacing: Theme.spacingSmall

                                QDButton {
                                    text: qsTr("Approve")
                                    iconText: FluentIconGlyph.acceptGlyph
                                    buttonType: QDButton.Type.Primary
                                    onClicked: root.openDecision(approvalCard.modelData, true)
                                }

                                QDButton {
                                    text: qsTr("Deny")
                                    iconText: FluentIconGlyph.cancelGlyph
                                    buttonType: QDButton.Type.Danger
                                    onClicked: root.openDecision(approvalCard.modelData, false)
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    QDDialog {
        id: decisionDialog
        dialogWidth: Math.min(520, root.width - 80)
        dialogHeight: 350
        closeOnOverlay: false

        ColumnLayout {
            anchors.fill: parent
            spacing: Theme.spacingMedium

            Text {
                Layout.fillWidth: true
                text: root.decisionApproval
                      ? qsTr("%1 from %2")
                        .arg(root.decisionApproval.client_id || qsTr("Unknown client"))
                        .arg(root.decisionApproval.client_ip || qsTr("Unknown IP"))
                      : ""
                font.pixelSize: Theme.fontSizeMedium
                color: Theme.text
                wrapMode: Text.WordWrap
            }

            Text {
                visible: !root.decisionApprove
                Layout.fillWidth: true
                text: qsTr("Denying also starts the source cooldown. The requester cannot immediately retry.")
                font.pixelSize: Theme.fontSizeSmall
                color: Theme.warning
                wrapMode: Text.WordWrap
            }

            Text {
                text: qsTr("Reason (optional)")
                font.pixelSize: Theme.fontSizeSmall
                color: Theme.textSecondary
            }

            QDTextArea {
                id: decisionReason
                Layout.fillWidth: true
                Layout.fillHeight: true
                placeholderText: root.decisionApprove
                                 ? qsTr("Why is this request approved?")
                                 : qsTr("Why is this request denied?")
                onTextChanged: {
                    if (text.length > 255) text = text.substring(0, 255)
                }
            }

            Text {
                Layout.fillWidth: true
                horizontalAlignment: Text.AlignRight
                text: decisionReason.text.length + " / 255"
                font.pixelSize: Theme.fontSizeSmall
                color: Theme.textSecondary
            }
        }

        footer: [
            QDButton {
                text: qsTr("Cancel")
                buttonType: QDButton.Type.Secondary
                onClicked: decisionDialog.reject()
            },
            QDButton {
                text: root.decisionApprove ? qsTr("Approve") : qsTr("Deny")
                buttonType: root.decisionApprove ? QDButton.Type.Primary : QDButton.Type.Danger
                loading: root.manager ? root.manager.busy : false
                onClicked: root.submitDecision()
            }
        ]
    }
}
