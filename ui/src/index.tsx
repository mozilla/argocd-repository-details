import * as React from "react";
import { ReleaseStatusPanel } from './component/release-status-panel'
import { ReleaseDetailsPanel } from "./component/release-details-panel";

// Register the details extension
((window) => {
  window.extensionsAPI.registerStatusPanelExtension(
    ReleaseStatusPanel,
    "Release Details",
    "release_details",
    ReleaseDetailsPanel
  );
})(window);