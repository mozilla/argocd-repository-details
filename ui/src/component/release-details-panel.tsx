import * as React from "react";
import { useState, useEffect } from "react";
import ReactMarkdown from "react-markdown";
import { getHeaders } from "../shared/headers";
import { getAppDetails } from "../shared/parse-app-info";
import { ReleaseInfo } from "../shared/release-info";


interface ReleaseDetailsPanelFlyoutProps {
  application: any;
}

export const ReleaseDetailsPanel = ({ application }: ReleaseDetailsPanelFlyoutProps) => {
  const [releaseInfo, setReleaseInfo] = useState<ReleaseInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const applicationNamespace = application?.metadata?.namespace || "";
  const applicationName = application?.metadata?.name || "";
  const project = application?.spec?.project || "";
  const images = application?.status?.summary?.images || "";
  const info = application?.spec?.info || "";

  useEffect(() => {
    if (!application) return;

    // We are checking the application repository for a gitRef that matches the imageTag
    const { appRepository, imageTag } = getAppDetails(images, info);

    const cacheKey = `${appRepository}-${imageTag}`;
    const cachedData = sessionStorage.getItem(cacheKey);

    const fetchReleaseInfo = async () => {
      try {
        if (cachedData) {
          // Use cached data if available
          setReleaseInfo(JSON.parse(cachedData));
          setLoading(false);
          return;
        }

        const response = await fetch(
          `/extensions/repository-details/api/references?repo=${appRepository}&gitRef=${imageTag}`,
          { headers: getHeaders({ applicationName, applicationNamespace, project }) }
        );

        if (!response.ok) {
          throw new Error(`Failed to fetch release info for ${imageTag} tag from ${appRepository} git repository`);
        }

        const data = await response.json();

        // Cache the data in sessionStorage
        sessionStorage.setItem(cacheKey, JSON.stringify(data));
        setReleaseInfo(data);
      } catch (err) {
        setError(err.message);
      } finally {
        setLoading(false);
      }
    };

    fetchReleaseInfo();
  }, [application]);

  if (loading) {
    return <div>Loading release information...</div>;
  }

  if (error) {
    return <div>Error loading release information: {error}</div>;
  }

  if (!releaseInfo) {
    return <div>No release information available for this application.</div>;
  }

  return (
    <div className="row" style={{ marginTop: "15px" }}>
      {/* Current Release Details */}
      <div className="columns small-6">
        <div className="white-box">
          <div className="white-box__details">
            <p>CURRENT RELEASE</p>
            <div className="row white-box__details-row">
              <div className="columns small-3">REF</div>
              <div className="columns small-9">
              {releaseInfo.current?.url ? (
                <a
                  href={releaseInfo.current.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  style={{ textDecoration: "none", color: "#007bff" }} // Optional: Add link styling
                >
                  {
                    releaseInfo.current?.ref
                    ? /^[0-9a-f]{40}$/i.test(releaseInfo.current.ref) // Check if ref is a valid Git SHA
                      ? releaseInfo.current.ref.slice(0, 7) // Shorten full SHA
                      : releaseInfo.current.ref // Return tag or non-SHA ref as-is
                    : "N/A"
                    }
                </a>
              ) : (
                    releaseInfo.current?.ref
                    ? /^[0-9a-f]{40}$/i.test(releaseInfo.current.ref) // Check if ref is a valid Git SHA
                      ? releaseInfo.current.ref.slice(0, 7) // Shorten full SHA
                      : releaseInfo.current.ref // Return tag or non-SHA ref as-is
                    : "N/A"
              )}
              </div>
            </div>
            <div className="row white-box__details-row">
              <div className="columns small-3">DESCRIPTION</div>
              <div className="columns small-9">
              {releaseInfo.current?.message ? (
                  <ReactMarkdown
                    components={{
                      h1: ({ children }) => <h3>{children}</h3>, // Shrink h1 to h3
                      h2: ({ children }) => <h4>{children}</h4>, // Shrink h2 to h4
                      h3: ({ children }) => <h5>{children}</h5>, // Shrink h3 to h5
                      h4: ({ children }) => <h6>{children}</h6>, // Shrink h4 to h6
                    }}
                  >
                    {releaseInfo.current?.message}
                  </ReactMarkdown>
                ) : (
                  "No description available"
                )}
              </div>
            </div>
            <div className="row white-box__details-row">
              <div className="columns small-3">PUBLISHED AT</div>
              <div className="columns small-9">
              {releaseInfo.current?.published_at || "N/A" }
              </div>
            </div>
            <div className="row white-box__details-row">
              <div className="columns small-3">AUTHOR</div>
              <div className="columns small-9">{releaseInfo.current?.author || "Unknown"}</div>
            </div>
          </div>
        </div>
      </div>

      {/* Latest Release Details */}
      <div className="columns small-6">
        <div className="white-box">
          <div className="white-box__details">
            <p>LATEST RELEASE</p>
            <div className="row white-box__details-row">
              <div className="columns small-3">REF</div>
              <div className="columns small-9">
              {releaseInfo.latest?.url ? (
                <a
                  href={releaseInfo.latest.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  style={{ textDecoration: "none", color: "#007bff" }} // Optional: Add link styling
                >
                  {
                  releaseInfo.latest?.ref
                    ? /^[0-9a-f]{40}$/i.test(releaseInfo.latest.ref) // Check if ref is a valid Git SHA
                      ? releaseInfo.latest.ref.slice(0, 7) // Shorten full SHA
                      : releaseInfo.latest.ref // Return tag or non-SHA ref as-is
                    : "N/A"
                    }
                </a>
              ) : (
                releaseInfo.latest?.ref
                ? /^[0-9a-f]{40}$/i.test(releaseInfo.latest.ref) // Check if ref is a valid Git SHA
                  ? releaseInfo.latest.ref.slice(0, 7) // Shorten full SHA
                  : releaseInfo.latest.ref // Return tag or non-SHA ref as-is
                : "N/A"
              )}
              </div>
            </div>
            <div className="row white-box__details-row">
              <div className="columns small-3">DESCRIPTION</div>
              <div className="columns small-9">
              {releaseInfo.latest?.message ? (
                  <ReactMarkdown
                    components={{
                      h1: ({ children }) => <h3>{children}</h3>, // Shrink h1 to h3
                      h2: ({ children }) => <h4>{children}</h4>, // Shrink h2 to h4
                      h3: ({ children }) => <h5>{children}</h5>, // Shrink h3 to h5
                      h4: ({ children }) => <h6>{children}</h6>, // Shrink h4 to h6
                    }}
                  >
                    {releaseInfo.latest.message}
                  </ReactMarkdown>
                ) : (
                  "No description available"
                )}
              </div>
            </div>
            <div className="row white-box__details-row">
              <div className="columns small-3">PUBLISHED AT</div>
              <div className="columns small-9">
              { releaseInfo.latest?.published_at || "N/A" }
              </div>
            </div>
            <div className="row white-box__details-row">
              <div className="columns small-3">AUTHOR</div>
              <div className="columns small-9">{releaseInfo.latest?.author || "Unknown"}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
  export default ReleaseDetailsPanel