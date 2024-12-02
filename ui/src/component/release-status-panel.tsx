import * as React from "react";
import { useState, useEffect } from "react";
import { HelpIcon } from 'argo-ui/src/components/help-icon/help-icon';
import { ARGO_GRAY6_COLOR } from '../shared/colors';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faGithub } from '@fortawesome/free-brands-svg-icons';
import { getHeaders } from "../shared/headers";
import { getAppDetails } from "../shared/parse-app-info";



export const ReleaseStatusPanel = ({ application, openFlyout }) => {
  const [releaseInfo, setReleaseInfo] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const applicationNamespace = application?.spec?.destination?.namespace || "";
  const applicationName = application?.metadata?.name || "";
  const project = application?.spec?.project || "";
  const images = application?.status?.summary?.images || "";
  const info = application?.spec?.info || "";
  // const spec = application || "";


  useEffect(() => {
    // We are checking the application repository for a gitRef that matches the imageTag
    const { appRepository, imageTag } = getAppDetails(images, info);

    console.info(appRepository, imageTag)

    const fetchReleaseInfo = async () => {
      const cacheKey = `${appRepository}-${imageTag}`;
      const cachedData = sessionStorage.getItem(cacheKey);

      if (cachedData) {
        // Use cached data if available
        setReleaseInfo(JSON.parse(cachedData));
        setLoading(false);
        return;
      }

      try {
        const response = await fetch(
          `/extensions/repository-details/api/references?repo=${appRepository}&gitRef=${imageTag}`,
          { headers: getHeaders({ applicationName, applicationNamespace, project }) }
        );
        if (!response.ok) {
          throw new Error(`Failed to fetch release info: ${response.statusText}`);
        }
        const data = await response.json();

        // Cache the response for future use
        sessionStorage.setItem(cacheKey, JSON.stringify(data));
        setReleaseInfo(data);
      } catch (err) {
        setError(err.message);
      } finally {
        setLoading(false);
      }
    };

    // Use a timeout to debounce frequent updates
    const timeoutId = setTimeout(fetchReleaseInfo, 300);

    // Cleanup timeout if the component unmounts
    return () => clearTimeout(timeoutId);
  }, [application, applicationNamespace, applicationName, project]);

  if (loading) {
    return <div>Loading release information...</div>;
  }

  if (error) {
    return <div>Error loading release information: {error}</div>;
  }

  if (!releaseInfo) {
    return <div>No release information available for this application.</div>;
  }
  //padding: "10px",
  return (
    <div
      key="current-release-details-icon"
      qe-id="current-release-details"
      className="application-status-panel__item"
      style={{ cursor: "pointer" }}
      onClick={() => openFlyout(application)} // Trigger the flyout with application details
    >
      <label
        style={{
          fontSize: "12px",
          fontWeight: 600,
          color: ARGO_GRAY6_COLOR,
          display: "flex",
          alignItems: "center",
          marginBottom: "0.5em",
        }}
      >
        CURRENT RELEASE &nbsp;
        {<HelpIcon title="The GitHub Release currently deployed by this ArgoCD Application. Click for more details and to see the latest release." />}
      </label>
      <div style={{ display: "flex", flexDirection: "column", alignItems: "flex-start" }}>
        {/* Tag Row */}
        <div
          style={{
            marginRight: "5px",
            position: "relative",
            top: '2px',
            display: "flex",
            paddingTop: '2px',
            alignItems: "center",
            fontFamily: "inherit",
          }}
        >
          <div
            className="application-status-panel__item-value"
            style={{
              display: "flex",
              alignItems: "center",
              gap: "8px", // Space between the icon and tag_name
            }}
          >
            {/* GitHub Icon */}
            <FontAwesomeIcon icon={faGithub} style={{ color: "#333", fontSize: "22px" }} />
            {/* Tag Name */}
            <span>{releaseInfo.current?.tag_name || "N/A"}</span>
          </div>
        </div>
      </div>
    </div>
  );
};

  export default ReleaseStatusPanel