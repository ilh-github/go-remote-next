import QtQuick
import QtQuick.Layouts
import "../component"

Item {
    id: root

    property var updateManager
    property bool showing: false
    property bool mandatory: false

    function open(isMandatory) {
        mandatory = isMandatory
        showing = true
    }

    function close() {
        if (!mandatory) {
            showing = false
        }
    }

    anchors.fill: parent
    visible: showing
    z: Theme.zIndexModal + 1

    Rectangle {
        anchors.fill: parent
        color: Theme.overlay

        MouseArea {
            anchors.fill: parent
            onClicked: {}
        }
    }

    Item {
        width: Math.min(560, parent.width - 80)
        height: updateContent.implicitHeight + Theme.spacingXXLarge * 2
        anchors.centerIn: parent

        QDShadow {
            anchors.fill: container
            target: container
            shadowSize: 18
            shadowColor: Qt.rgba(0, 0, 0, 0.45)
        }

        Rectangle {
            id: container
            anchors.fill: parent
            color: Theme.surface
            radius: Theme.radiusLarge
            border.width: Theme.borderWidthThin
            border.color: Theme.border

            ColumnLayout {
                id: updateContent
                anchors.fill: parent
                anchors.margins: Theme.spacingXXLarge
                spacing: Theme.spacingLarge

                RowLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingMedium

                    Text {
                        text: FluentIconGlyph.downloadGlyph
                        font.family: "Segoe Fluent Icons"
                        font.pixelSize: 30
                        color: Theme.primary
                    }

                    ColumnLayout {
                        Layout.fillWidth: true
                        spacing: 2

                        Text {
                            text: root.mandatory ? qsTr("Required Update") : qsTr("Update Available")
                            font.pixelSize: Theme.fontSizeHeading
                            font.weight: Font.DemiBold
                            color: Theme.text
                        }

                        Text {
                            text: qsTr("Version %1 is available. You are running %2.")
                                .arg(root.updateManager ? root.updateManager.latestVersion : "")
                                .arg(root.updateManager ? root.updateManager.currentVersion : "")
                            font.pixelSize: Theme.fontSizeSmall
                            color: Theme.textSecondary
                        }
                    }
                }

                Rectangle {
                    Layout.fillWidth: true
                    Layout.preferredHeight: 1
                    color: Theme.border
                }

                Text {
                    visible: root.updateManager && root.updateManager.releaseNotes.length > 0
                    text: root.updateManager ? root.updateManager.releaseNotes : ""
                    font.pixelSize: Theme.fontSizeMedium
                    color: Theme.text
                    wrapMode: Text.WordWrap
                    Layout.fillWidth: true
                    maximumLineCount: 8
                    elide: Text.ElideRight
                }

                QDProgressBar {
                    visible: root.updateManager &&
                             (root.updateManager.state === "downloading" ||
                              root.updateManager.state === "ready")
                    Layout.fillWidth: true
                    from: 0
                    to: 1
                    value: root.updateManager ? root.updateManager.downloadProgress : 0
                }

                Text {
                    Layout.fillWidth: true
                    wrapMode: Text.WordWrap
                    font.pixelSize: Theme.fontSizeSmall
                    color: root.updateManager && root.updateManager.state === "failed"
                           ? Theme.error : Theme.textSecondary
                    text: {
                        if (!root.updateManager) return ""
                        switch (root.updateManager.state) {
                        case "available":
                            return root.mandatory
                                ? qsTr("This update is required. Download it to continue using QuickDesk.")
                                : qsTr("Download the installer when you are ready.")
                        case "downloading":
                            return qsTr("Downloading installer: %1%")
                                .arg(Math.round(root.updateManager.downloadProgress * 100))
                        case "ready":
                            return qsTr("Download verified. QuickDesk will close while the installer runs.")
                        case "installing":
                            return qsTr("Starting the installer...")
                        case "failed":
                            return root.updateManager.errorMessage
                        default:
                            return ""
                        }
                    }
                }

                RowLayout {
                    Layout.fillWidth: true
                    Layout.alignment: Qt.AlignRight
                    spacing: Theme.spacingMedium

                    QDButton {
                        visible: !root.mandatory &&
                                 root.updateManager &&
                                 (root.updateManager.state === "available" ||
                                  root.updateManager.state === "failed" ||
                                  root.updateManager.state === "ready")
                        text: qsTr("Later")
                        buttonType: QDButton.Type.Secondary
                        onClicked: {
                            root.updateManager.dismissUpdate()
                            root.close()
                        }
                    }

                    QDButton {
                        visible: root.updateManager && root.updateManager.state === "downloading"
                        text: qsTr("Cancel")
                        buttonType: QDButton.Type.Secondary
                        onClicked: root.updateManager.cancelDownload()
                    }

                    QDButton {
                        visible: root.updateManager &&
                                 (root.updateManager.state === "available" ||
                                  root.updateManager.state === "failed")
                        text: root.mandatory ? qsTr("Download Required Update") : qsTr("Download")
                        iconText: FluentIconGlyph.downloadGlyph
                        buttonType: QDButton.Type.Primary
                        onClicked: root.updateManager.downloadUpdate()
                    }

                    QDButton {
                        visible: root.updateManager && root.updateManager.state === "ready"
                        text: qsTr("Install and Restart")
                        iconText: FluentIconGlyph.acceptGlyph
                        buttonType: QDButton.Type.Primary
                        onClicked: root.updateManager.installUpdate()
                    }
                }
            }
        }
    }
}
