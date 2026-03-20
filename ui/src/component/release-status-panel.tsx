import * as React from "react";
import { useState, useEffect } from "react";
import { HelpIcon } from 'argo-ui/src/components/help-icon/help-icon';
import { ARGO_GRAY6_COLOR } from '../shared/colors';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faGithub } from '@fortawesome/free-brands-svg-icons';
import { getHeaders } from "../shared/headers";
import { getAppDetails } from "../shared/parse-app-info";
import { ReleaseInfo } from "../shared/release-info";



export const ReleaseStatusPanel = ({ application, openFlyout }) => {
  const [entries, setEntries] = useState<Array<{ alias: string | null; releaseInfo: ReleaseInfo | null }>>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const applicationNamespace = application?.metadata?.namespace || "";
  const applicationName = application?.metadata?.name || "";
  const project = application?.spec?.project || "";
  const images = application?.status?.summary?.images || "";
  const info = application?.spec?.info || "";

  useEffect(() => {
    const { appRepository, imageEntries } = getAppDetails(images, info);

    if (!appRepository || imageEntries.length === 0 || imageEntries.every((e) => !e.imageTag)) {
      setError(
        `Missing required fields: ${!appRepository ? "Application Repository" : ""} ${!appRepository && imageEntries.every((e) => !e.imageTag) ? "and" : ""} ${imageEntries.every((e) => !e.imageTag) ? "Image Tag" : ""}.`
      );
      setLoading(false);
      return;
    }

    const fetchReleaseInfo = async () => {
      try {
        const results = await Promise.all(
          imageEntries.map(async ({ alias, imageTag }) => {
            const cacheKey = `${appRepository}-${imageTag}`;
            const cachedData = sessionStorage.getItem(cacheKey);
            if (cachedData) {
              return { alias, releaseInfo: JSON.parse(cachedData) };
            }

            const response = await fetch(
              `/extensions/repository-details/api/references?repo=${appRepository}&gitRef=${imageTag}`,
              { headers: getHeaders({ applicationName, applicationNamespace, project }) }
            );
            if (!response.ok) {
              throw new Error(`Failed to fetch release info for ${imageTag} tag from ${appRepository} git repository`);
            }
            const data = await response.json();
            sessionStorage.setItem(cacheKey, JSON.stringify(data));
            return { alias, releaseInfo: data };
          })
        );
        setEntries(results);
      } catch (err) {
        setError(err.message);
      } finally {
        setLoading(false);
      }
    };

    const timeoutId = setTimeout(fetchReleaseInfo, 300);
    return () => clearTimeout(timeoutId);
  }, [application, applicationNamespace, applicationName, project]);

  const renderStatusPanel = (message, color, messageStyle = {}) => (
    <div
      key="status-panel"
      qe-id="status-panel"
      className="application-status-panel__item"
      style={{
        fontSize: "12px",
        fontWeight: 600,
        color: color || ARGO_GRAY6_COLOR,
        display: "flex",
        flexDirection: "column",
        alignItems: "flex-start",
        marginBottom: "0.5em",
      }}
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
        DEPLOYED RELEASE &nbsp;
        {<HelpIcon title="The GitHub Release or Commit currently deployed by this ArgoCD Application. Click for more details and to see the latest application release." />}
      </label>
      <div style={{ ...messageStyle }}>{message}</div>
    </div>
  );

  if (loading) {
    return renderStatusPanel("Loading release information...", ARGO_GRAY6_COLOR);
  }

  if (error) {
    return renderStatusPanel(
      error,
      ARGO_GRAY6_COLOR,
      {
        wordWrap: "break-word",
        overflowWrap: "break-word",
        maxWidth: "360px",
        whiteSpace: "normal",
      }
    );
  }

  if (entries.length === 0) {
    return renderStatusPanel("No release information available for this application.", ARGO_GRAY6_COLOR);
  }

  const isMulti = entries.length > 1;

  const formatRef = (ref) => {
    if (!ref) return "N/A";
    return /^[0-9a-f]{40}$/i.test(ref) ? ref.slice(0, 7) : ref;
  };

  return (
    <div
      key="current-release-details-icon"
      qe-id="current-release-details"
      className="application-status-panel__item"
      style={{ cursor: "pointer" }}
      onClick={() => openFlyout(application)}
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
        {isMulti ? "DEPLOYED RELEASES" : "DEPLOYED RELEASE"} &nbsp;
        {<HelpIcon title="The GitHub Release or Commit currently deployed by this ArgoCD Application. Click for more details and to see the latest application release." />}
      </label>
      <div style={{ display: "flex", flexDirection: "column", alignItems: "flex-start" }}>
        {entries.map(({ alias, releaseInfo }, i) => (
          <div
            key={i}
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
                gap: "8px",
              }}
            >
              <FontAwesomeIcon icon={faGithub} style={{ color: "#333", fontSize: "22px" }} />
              <span>
                {isMulti && alias ? `${alias}: ` : ""}
                {formatRef(releaseInfo?.current?.ref)}
              </span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

  export default ReleaseStatusPanel
