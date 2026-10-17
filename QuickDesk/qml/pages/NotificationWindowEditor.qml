pragma ComponentBehavior: Bound

import QtQuick
import QtQuick.Layouts
import QuickDesk 1.0

import "../component"

ColumnLayout {
    id: root

    property var windows: []
    property string emptyText: qsTr("No recurring windows")

    signal windowsEdited(var value)

    spacing: Theme.spacingSmall

    function cloneWindows() {
        return JSON.parse(JSON.stringify(root.windows || []))
    }

    function hasDay(index, day) {
        var list = root.windows[index] && root.windows[index].days
                   ? root.windows[index].days : []
        return list.indexOf(day) >= 0
    }

    function updateWindow(index, key, value) {
        var next = cloneWindows()
        if (!next[index]) return
        next[index][key] = value
        root.windows = next
        root.windowsEdited(next)
    }

    function toggleDay(index, day, checked) {
        var next = cloneWindows()
        if (!next[index]) return
        var days = next[index].days || []
        var position = days.indexOf(day)
        if (checked && position < 0) days.push(day)
        if (!checked && position >= 0) days.splice(position, 1)
        days.sort(function(left, right) { return left - right })
        next[index].days = days
        root.windows = next
        root.windowsEdited(next)
    }

    function addWindow() {
        var next = cloneWindows()
        next.push({ days: [1, 2, 3, 4, 5], start: "22:00", end: "07:00" })
        root.windows = next
        root.windowsEdited(next)
    }

    function removeWindow(index) {
        var next = cloneWindows()
        next.splice(index, 1)
        root.windows = next
        root.windowsEdited(next)
    }

    Repeater {
        model: root.windows

        Rectangle {
            id: windowCard
            required property int index
            required property var modelData

            Layout.fillWidth: true
            Layout.preferredHeight: windowContent.implicitHeight + Theme.spacingLarge * 2
            color: Theme.surfaceVariant
            radius: Theme.radiusMedium
            border.width: Theme.borderWidthThin
            border.color: Theme.border

            ColumnLayout {
                id: windowContent
                anchors.fill: parent
                anchors.margins: Theme.spacingLarge
                spacing: Theme.spacingSmall

                RowLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingSmall

                    Text {
                        Layout.fillWidth: true
                        text: qsTr("Recurring window %1").arg(windowCard.index + 1)
                        font.pixelSize: Theme.fontSizeMedium
                        font.weight: Font.DemiBold
                        color: Theme.text
                    }

                    QDButton {
                        text: qsTr("Remove")
                        iconText: FluentIconGlyph.deleteGlyph
                        buttonType: QDButton.Type.Ghost
                        onClicked: root.removeWindow(windowCard.index)
                    }
                }

                RowLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingSmall

                    QDTextField {
                        Layout.fillWidth: true
                        text: windowCard.modelData.start || ""
                        placeholderText: qsTr("Start HH:MM")
                        onEditingFinished: root.updateWindow(
                                               windowCard.index, "start", text)
                    }

                    QDTextField {
                        Layout.fillWidth: true
                        text: windowCard.modelData.end || ""
                        placeholderText: qsTr("End HH:MM")
                        onEditingFinished: root.updateWindow(
                                               windowCard.index, "end", text)
                    }
                }

                Flow {
                    Layout.fillWidth: true

                    Repeater {
                        model: [
                            { value: 1, label: qsTr("Mon") },
                            { value: 2, label: qsTr("Tue") },
                            { value: 3, label: qsTr("Wed") },
                            { value: 4, label: qsTr("Thu") },
                            { value: 5, label: qsTr("Fri") },
                            { value: 6, label: qsTr("Sat") },
                            { value: 0, label: qsTr("Sun") }
                        ]

                        QDCheckBox {
                            required property var modelData
                            text: modelData.label
                            checked: root.hasDay(windowCard.index, modelData.value)
                            onToggled: root.toggleDay(
                                           windowCard.index,
                                           modelData.value,
                                           checked)
                        }
                    }
                }
            }
        }
    }

    Text {
        visible: !root.windows || root.windows.length === 0
        Layout.fillWidth: true
        text: root.emptyText
        color: Theme.textSecondary
        font.pixelSize: Theme.fontSizeSmall
    }

    QDButton {
        text: qsTr("Add window")
        iconText: FluentIconGlyph.addGlyph
        buttonType: QDButton.Type.Secondary
        onClicked: root.addWindow()
    }
}
