{ config, pkgs, settings, ... }:
let
    themeConf = pkgs.writeText "gjallaros-sddm-theme.conf" ''
        [General]
    '';
    metadata = pkgs.writeText "gjallaros-sddm-metadata.desktop" ''
        [SddmGreeterTheme]
        Name=GjallarOS
        Description=Minimal GjallarOS SDDM greeter
        Type=sddm-theme
        MainScript=Main.qml
        Theme-Id=gjallaros
        Theme-API=2.0
    '';
    mainQml = pkgs.writeText "gjallaros-sddm-main.qml" ''
        import QtQuick 2.15
        import QtQuick.Controls 2.15
        import SddmComponents 2.0

        Rectangle {
            id: root
            width: 1920
            height: 1080
            color: "#11111b"

            property color foreground: "#cdd6f4"
            property color muted: "#a6adc8"
            property color accent: "#cba6f7"
            property color surface: "#181825"
            property color border: "#313244"

            Rectangle {
                anchors.fill: parent
                anchors.margins: 48
                color: "transparent"
                border.color: root.border
                border.width: 1

                Column {
                    anchors.centerIn: parent
                    width: 400
                    spacing: 18

                    Text {
                        text: "GJALLAROS"
                        color: root.foreground
                        font.pixelSize: 30
                        font.letterSpacing: 4
                        font.bold: true
                    }
                    Text {
                        text: "HYPRLAND WORKSTATION"
                        color: root.muted
                        font.pixelSize: 12
                        font.letterSpacing: 2
                    }
                    Item { width: 1; height: 22 }

                    TextField {
                        id: username
                        width: parent.width
                        height: 52
                        text: userModel.lastUser
                        placeholderText: "USER"
                        color: root.foreground
                        selectByMouse: true
                        background: Rectangle {
                            color: root.surface
                            border.color: username.activeFocus ? root.accent : root.border
                            border.width: 1
                        }
                    }
                    TextField {
                        id: password
                        width: parent.width
                        height: 52
                        placeholderText: "PASSWORD"
                        echoMode: TextInput.Password
                        color: root.foreground
                        selectByMouse: true
                        onAccepted: sddm.login(username.text, password.text, session.currentIndex)
                        background: Rectangle {
                            color: root.surface
                            border.color: password.activeFocus ? root.accent : root.border
                            border.width: 1
                        }
                    }
                    ComboBox {
                        id: session
                        width: parent.width
                        height: 46
                        model: sessionModel
                        textRole: "name"
                        currentIndex: sessionModel.lastIndex
                        contentItem: Text {
                            leftPadding: 14
                            text: session.displayText
                            color: root.foreground
                            verticalAlignment: Text.AlignVCenter
                        }
                        background: Rectangle {
                            color: root.surface
                            border.color: session.activeFocus ? root.accent : root.border
                            border.width: 1
                        }
                    }
                    Button {
                        width: parent.width
                        height: 52
                        text: "SIGN IN"
                        onClicked: sddm.login(username.text, password.text, session.currentIndex)
                        background: Rectangle { color: parent.down ? "#b4befe" : root.accent }
                        contentItem: Text {
                            text: parent.text
                            color: "#11111b"
                            horizontalAlignment: Text.AlignHCenter
                            verticalAlignment: Text.AlignVCenter
                            font.bold: true
                            font.letterSpacing: 2
                        }
                    }
                }
            }
        }
    '';
    gjallarSddmTheme = pkgs.runCommand "gjallaros-sddm-theme" {} ''
        install -Dm444 ${themeConf} "$out/share/sddm/themes/gjallaros/theme.conf"
        install -Dm444 ${metadata} "$out/share/sddm/themes/gjallaros/metadata.desktop"
        install -Dm444 ${mainQml} "$out/share/sddm/themes/gjallaros/Main.qml"
    '';
in

{
    environment.systemPackages = with pkgs; [
        wayland
        wl-clipboard
    ] ++ [ gjallarSddmTheme ];

    # Configure xwayland
    services.xserver = {
        enable = true;
        xkb = {
            variant = settings.keyboardVariant;
            layout = settings.keyboardLayout;
            options = "grp:win_space_toggle";
        };
    };

    # SDDM provides a graphical login screen while keeping Hyprland as the
    # only supported desktop session.
    services.displayManager = {
        defaultSession = "hyprland";
        sddm = {
            enable = true;
            wayland.enable = true;
            theme = "gjallaros";
        };
    };
}
