import QtQuick
import Quickshell
import qs.Commons
import qs.Ui

// jianmu's icon in Omarchy's bar, put there by jianmu (Settings → Bar icon).
// A click drops down its panel under the bar, where the click was; a right
// click opens its window. jianmu is started if it isn't running.
BarWidget {
  id: root
  moduleName: "usejianmu.jianmu"

  // written in by jianmu when it puts the widget here
  readonly property string jianmu: "__JIANMU__"

  implicitWidth: button.implicitWidth
  implicitHeight: button.implicitHeight

  BarIconButton {
    id: button
    anchors.fill: parent
    bar: root.bar
    text: String.fromCodePoint(0xf15c6) // nf-md-bird
    tooltipText: "jianmu"
    onPressed: function(b) {
      Quickshell.execDetached(b === Qt.RightButton ? [root.jianmu, "app"] : [root.jianmu, "panel"])
    }
  }
}
