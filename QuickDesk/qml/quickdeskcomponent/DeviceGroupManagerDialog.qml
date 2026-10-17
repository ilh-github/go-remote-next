pragma ComponentBehavior: Bound

import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

import "../component"

Popup {
    id: root

    required property var groupManager
    property var myDevices: []

    modal: true
    anchors.centerIn: parent
    width: 820
    height: 580
    padding: 0
    closePolicy: Popup.CloseOnEscape

    property int selectedGroupId: 0
    property var draftDeviceIds: []
    property var pendingDeviceIds: []
    property string savePhase: "idle"
    property bool saveInProgress: false
    property bool deleteInProgress: false
    property string errorMessage: ""
    property var colorOptions: [
        "#1f6feb",
        "#2da44e",
        "#bf8700",
        "#cf222e",
        "#8250df",
        "#0969da"
    ]

    signal showToast(string message, int toastType)

    function deviceLabel(device) {
        return device.remark || device.device_name || device.device_id || qsTr("Device")
    }

    function cloneList(value) {
        return JSON.parse(JSON.stringify(value || []))
    }

    function findGroup(groupId) {
        if (!root.groupManager) return null
        var groups = root.groupManager.groups || []
        for (var i = 0; i < groups.length; ++i) {
            if (Number(groups[i].id) === Number(groupId))
                return groups[i]
        }
        return null
    }

    function hasDraftDevice(deviceId) {
        return root.draftDeviceIds.indexOf(deviceId) >= 0
    }

    function setDraftDevice(deviceId, selected) {
        var next = root.draftDeviceIds.slice()
        var index = next.indexOf(deviceId)
        if (selected && index < 0) next.push(deviceId)
        if (!selected && index >= 0) next.splice(index, 1)
        root.draftDeviceIds = next
    }

    function selectGroup(group) {
        if (!group) {
            beginNewGroup()
            return
        }
        root.selectedGroupId = Number(group.id || 0)
        groupNameField.text = group.name || ""
        groupDescriptionField.text = group.description || ""
        groupColorField.text = group.color || ""
        root.draftDeviceIds = cloneList(group.device_ids)
        root.errorMessage = ""
    }

    function beginNewGroup() {
        root.selectedGroupId = 0
        groupNameField.text = ""
        groupDescriptionField.text = ""
        groupColorField.text = "#1f6feb"
        root.draftDeviceIds = []
        root.errorMessage = ""
    }

    function ensureSelection() {
        if (root.saveInProgress || root.deleteInProgress) return
        var groups = root.groupManager ? root.groupManager.groups || [] : []
        var selected = root.findGroup(root.selectedGroupId)
        if (selected) {
            root.selectGroup(selected)
        } else if (groups.length > 0) {
            root.selectGroup(groups[0])
        } else {
            root.beginNewGroup()
        }
    }

    function finishSave() {
        root.saveInProgress = false
        root.savePhase = "idle"
        root.showToast(qsTr("Device group saved"), QDToast.Type.Success)
        root.close()
    }

    function saveGroup() {
        if (!root.groupManager || root.saveInProgress) return
        var name = groupNameField.text.trim()
        if (name === "") {
            root.errorMessage = qsTr("Group name is required")
            return
        }
        root.errorMessage = ""
        root.saveInProgress = true
        root.pendingDeviceIds = root.draftDeviceIds.slice()
        if (root.selectedGroupId > 0) {
            root.savePhase = "update"
            root.groupManager.updateGroup(
                root.selectedGroupId,
                name,
                groupDescriptionField.text,
                groupColorField.text)
        } else {
            root.savePhase = "create"
            root.groupManager.createGroup(
                name,
                groupDescriptionField.text,
                groupColorField.text)
        }
    }

    function requestDelete() {
        if (root.selectedGroupId <= 0 || root.saveInProgress) return
        deleteConfirmPopup.open()
    }

    function confirmDelete() {
        if (!root.groupManager || root.selectedGroupId <= 0) return
        root.deleteInProgress = true
        root.errorMessage = ""
        deleteConfirmPopup.close()
        root.groupManager.deleteGroup(root.selectedGroupId)
    }

    onOpened: {
        root.errorMessage = ""
        if (root.groupManager) root.groupManager.refreshGroups()
        root.ensureSelection()
    }

    background: Rectangle {
        color: Theme.surface
        radius: Theme.radiusLarge
        border.width: Theme.borderWidthThin
        border.color: Theme.border
    }

    contentItem: ColumnLayout {
        spacing: 0

        Rectangle {
            Layout.fillWidth: true
            Layout.preferredHeight: 60
            color: Theme.surfaceVariant
            radius: Theme.radiusLarge

            Rectangle {
                anchors.bottom: parent.bottom
                width: parent.width
                height: parent.radius
                color: parent.color
            }

            RowLayout {
                anchors.fill: parent
                anchors.leftMargin: Theme.spacingLarge
                anchors.rightMargin: Theme.spacingLarge
                spacing: Theme.spacingMedium

                Text {
                    text: FluentIconGlyph.groupGlyph
                    font.family: "Segoe Fluent Icons"
                    font.pixelSize: Theme.iconSizeMedium
                    color: Theme.primary
                }

                Text {
                    Layout.fillWidth: true
                    text: qsTr("Manage device groups")
                    font.pixelSize: Theme.fontSizeTitle
                    font.weight: Font.DemiBold
                    color: Theme.text
                }

                QDSpinner {
                    size: 24
                    running: root.groupManager ? root.groupManager.busy : false
                    visible: running
                }

                QDIconButton {
                    iconSource: FluentIconGlyph.cancelGlyph
                    toolTipText: qsTr("Close")
                    buttonSize: QDIconButton.Size.Small
                    enabled: !root.saveInProgress && !root.deleteInProgress
                    onClicked: root.close()
                }
            }
        }

        RowLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            Layout.margins: Theme.spacingLarge
            spacing: Theme.spacingLarge

            Rectangle {
                Layout.preferredWidth: 230
                Layout.fillHeight: true
                radius: Theme.radiusMedium
                color: Theme.surfaceVariant
                border.width: Theme.borderWidthThin
                border.color: Theme.border

                ColumnLayout {
                    anchors.fill: parent
                    anchors.margins: Theme.spacingMedium
                    spacing: Theme.spacingSmall

                    QDButton {
                        Layout.fillWidth: true
                        text: qsTr("New group")
                        iconText: FluentIconGlyph.addGlyph
                        enabled: !root.saveInProgress && !root.deleteInProgress
                        onClicked: root.beginNewGroup()
                    }

                    ListView {
                        id: groupsList
                        Layout.fillWidth: true
                        Layout.fillHeight: true
                        clip: true
                        spacing: 2
                        model: root.groupManager ? root.groupManager.groups : []
                        ScrollBar.vertical: QDScrollBar {}

                        delegate: Rectangle {
                            id: groupDelegate
                            required property var modelData

                            width: groupsList.width
                            height: 48
                            radius: Theme.radiusSmall
                            color: Number(modelData.id) === root.selectedGroupId
                                   ? Theme.surfaceHover
                                   : groupMouse.containsMouse
                                     ? Theme.surfaceHover : "transparent"

                            RowLayout {
                                anchors.fill: parent
                                anchors.leftMargin: Theme.spacingSmall
                                anchors.rightMargin: Theme.spacingSmall
                                spacing: Theme.spacingSmall

                                Rectangle {
                                    Layout.preferredWidth: 10
                                    Layout.preferredHeight: 10
                                    radius: 5
                                    color: groupDelegate.modelData.color
                                           || Theme.primary
                                }

                                Text {
                                    Layout.fillWidth: true
                                    text: groupDelegate.modelData.name
                                          || qsTr("Untitled group")
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.text
                                    elide: Text.ElideRight
                                }

                                Text {
                                    text: String(Number(
                                                     groupDelegate.modelData.device_count
                                                     || 0))
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                }
                            }

                            MouseArea {
                                id: groupMouse
                                anchors.fill: parent
                                hoverEnabled: true
                                enabled: !root.saveInProgress && !root.deleteInProgress
                                cursorShape: Qt.PointingHandCursor
                                onClicked: root.selectGroup(
                                               groupDelegate.modelData)
                            }
                        }
                    }
                }
            }

            ColumnLayout {
                Layout.fillWidth: true
                Layout.fillHeight: true
                spacing: Theme.spacingMedium

                Text {
                    visible: root.errorMessage !== ""
                    Layout.fillWidth: true
                    text: root.errorMessage
                    wrapMode: Text.Wrap
                    font.pixelSize: Theme.fontSizeSmall
                    color: Theme.error
                }

                QDTextField {
                    id: groupNameField
                    Layout.fillWidth: true
                    placeholderText: qsTr("Group name")
                    maximumLength: 100
                    enabled: !root.saveInProgress && !root.deleteInProgress
                }

                QDTextField {
                    id: groupDescriptionField
                    Layout.fillWidth: true
                    placeholderText: qsTr("Description")
                    maximumLength: 500
                    enabled: !root.saveInProgress && !root.deleteInProgress
                }

                RowLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingSmall

                    Text {
                        text: qsTr("Color")
                        font.pixelSize: Theme.fontSizeSmall
                        color: Theme.textSecondary
                    }

                    Repeater {
                        model: root.colorOptions

                        delegate: Rectangle {
                            id: colorOption
                            required property string modelData

                            width: 22
                            height: 22
                            radius: 4
                            color: modelData
                            border.width: groupColorField.text.toLowerCase()
                                          === colorOption.modelData.toLowerCase()
                                          ? 2 : 1
                            border.color: border.width === 2
                                          ? Theme.text : Theme.border

                            MouseArea {
                                anchors.fill: parent
                                enabled: !root.saveInProgress && !root.deleteInProgress
                                cursorShape: Qt.PointingHandCursor
                                onClicked: groupColorField.text =
                                           colorOption.modelData
                            }
                        }
                    }

                    QDTextField {
                        id: groupColorField
                        Layout.fillWidth: true
                        placeholderText: qsTr("Color value")
                        maximumLength: 20
                        enabled: !root.saveInProgress && !root.deleteInProgress
                    }
                }

                RowLayout {
                    Layout.fillWidth: true

                    Text {
                        Layout.fillWidth: true
                        text: qsTr("Devices in this group")
                        font.pixelSize: Theme.fontSizeMedium
                        font.weight: Font.DemiBold
                        color: Theme.text
                    }

                    Text {
                        text: qsTr("%1 selected").arg(root.draftDeviceIds.length)
                        font.pixelSize: Theme.fontSizeSmall
                        color: Theme.textSecondary
                    }
                }

                Rectangle {
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    radius: Theme.radiusMedium
                    color: Theme.surfaceVariant
                    border.width: Theme.borderWidthThin
                    border.color: Theme.border

                    ListView {
                        id: devicesList
                        anchors.fill: parent
                        anchors.margins: Theme.spacingSmall
                        clip: true
                        spacing: 1
                        model: root.myDevices
                        ScrollBar.vertical: QDScrollBar {}

                        delegate: QDCheckBox {
                            id: deviceDelegate
                            required property var modelData

                            width: devicesList.width
                            height: 40
                            text: root.deviceLabel(deviceDelegate.modelData)
                            checked: root.hasDraftDevice(
                                         deviceDelegate.modelData.device_id
                                         || "")
                            enabled: !root.saveInProgress && !root.deleteInProgress
                            onToggled: root.setDraftDevice(
                                           deviceDelegate.modelData.device_id
                                           || "",
                                           checked)
                        }
                    }

                    Text {
                        anchors.centerIn: parent
                        visible: devicesList.count === 0
                        text: qsTr("No bound devices")
                        font.pixelSize: Theme.fontSizeSmall
                        color: Theme.textSecondary
                    }
                }
            }
        }

        Rectangle {
            Layout.fillWidth: true
            Layout.preferredHeight: 68
            color: Theme.surfaceVariant
            radius: Theme.radiusLarge

            Rectangle {
                anchors.top: parent.top
                width: parent.width
                height: parent.radius
                color: parent.color
            }

            RowLayout {
                anchors.fill: parent
                anchors.leftMargin: Theme.spacingLarge
                anchors.rightMargin: Theme.spacingLarge
                spacing: Theme.spacingSmall

                QDButton {
                    text: qsTr("Delete group")
                    iconText: FluentIconGlyph.deleteGlyph
                    buttonType: QDButton.Type.Danger
                    visible: root.selectedGroupId > 0
                    enabled: !root.saveInProgress && !root.deleteInProgress
                    onClicked: root.requestDelete()
                }

                Item { Layout.fillWidth: true }

                QDButton {
                    text: qsTr("Cancel")
                    enabled: !root.saveInProgress && !root.deleteInProgress
                    onClicked: root.close()
                }

                QDButton {
                    text: root.selectedGroupId > 0 ? qsTr("Save changes")
                                                   : qsTr("Create group")
                    highlighted: true
                    enabled: !root.saveInProgress && !root.deleteInProgress
                    onClicked: root.saveGroup()
                }
            }
        }
    }

    Popup {
        id: deleteConfirmPopup
        parent: Overlay.overlay
        anchors.centerIn: parent
        modal: true
        width: 360
        padding: Theme.spacingLarge

        background: Rectangle {
            color: Theme.surface
            radius: Theme.radiusLarge
            border.width: Theme.borderWidthThin
            border.color: Theme.border
        }

        contentItem: ColumnLayout {
            spacing: Theme.spacingLarge

            Text {
                Layout.fillWidth: true
                text: qsTr("Delete this device group? Devices are not unbound.")
                wrapMode: Text.Wrap
                font.pixelSize: Theme.fontSizeMedium
                color: Theme.text
            }

            RowLayout {
                Layout.alignment: Qt.AlignRight
                spacing: Theme.spacingSmall

                QDButton {
                    text: qsTr("Cancel")
                    onClicked: deleteConfirmPopup.close()
                }

                QDButton {
                    text: qsTr("Delete")
                    buttonType: QDButton.Type.Danger
                    onClicked: root.confirmDelete()
                }
            }
        }
    }

    Connections {
        target: root.groupManager

        function onGroupsChanged() {
            root.ensureSelection()
        }

        function onGroupCreated(groupId) {
            if (!root.saveInProgress || root.savePhase !== "create") return
            root.selectedGroupId = groupId
            if (root.pendingDeviceIds.length > 0) {
                root.savePhase = "members-create"
                root.groupManager.replaceGroupDevices(
                    groupId, root.pendingDeviceIds)
            } else {
                root.finishSave()
            }
        }

        function onGroupUpdated(groupId) {
            if (!root.saveInProgress || Number(groupId) !== root.selectedGroupId)
                return
            if (root.savePhase === "update") {
                root.savePhase = "members"
                root.groupManager.replaceGroupDevices(
                    groupId, root.pendingDeviceIds)
            } else if (root.savePhase === "members"
                       || root.savePhase === "members-create") {
                root.finishSave()
            }
        }

        function onGroupDeleted(groupId) {
            if (!root.deleteInProgress) return
            root.deleteInProgress = false
            root.selectedGroupId = 0
            root.showToast(qsTr("Device group deleted"), QDToast.Type.Success)
            root.ensureSelection()
        }

        function onRequestFailed(message) {
            root.saveInProgress = false
            root.deleteInProgress = false
            root.savePhase = "idle"
            root.errorMessage = message
        }
    }
}
