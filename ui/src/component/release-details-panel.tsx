import * as React from "react";
import { useState, useEffect } from "react";
import ReactMarkdown from "react-markdown";
import { getHeaders } from "../shared/headers";
import { getAppDetails } from "../shared/parse-app-info";

interface ReleaseDetailsPanelFlyoutProps {
  application: any;
}

export const ReleaseDetailsPanel = ({ application }: ReleaseDetailsPanelFlyoutProps) => {
  const [releaseInfo, setReleaseInfo] = useState(null);
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
              {releaseInfo.current?.html_url ? (
                <a
                  href={releaseInfo.current.html_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  style={{ textDecoration: "none", color: "#007bff" }} // Optional: Add link styling
                >
                  {releaseInfo.current?.tag_name
                    ? releaseInfo.current.tag_name
                    : releaseInfo.current?.sha
                    ? releaseInfo.current.sha.slice(0, 7)
                    : "N/A"}
                </a>
              ) : (
                releaseInfo.current?.tag_name
                  ? releaseInfo.current.tag_name
                  : releaseInfo.current?.sha
                  ? releaseInfo.current.sha.slice(0, 7)
                  : "N/A"
              )}
              </div>
            </div>
            <div className="row white-box__details-row">
              <div className="columns small-3">DESCRIPTION</div>
              <div className="columns small-9">
              {releaseInfo.current.body ? (
                <ReactMarkdown
                  components={{
                    h1: ({ children }) => <h3>{children}</h3>, // Shrink h1 to h3
                    h2: ({ children }) => <h4>{children}</h4>, // Shrink h2 to h4
                    h3: ({ children }) => <h5>{children}</h5>, // Shrink h3 to h5
                    h4: ({ children }) => <h6>{children}</h6>, // Shrink h4 to h6
                  }}
                >
                  {releaseInfo.current.body}
                </ReactMarkdown>
              ) : releaseInfo.current?.commit?.message ? (
                <ReactMarkdown
                  components={{
                    h1: ({ children }) => <h3>{children}</h3>, // Shrink h1 to h3
                    h2: ({ children }) => <h4>{children}</h4>, // Shrink h2 to h4
                    h3: ({ children }) => <h5>{children}</h5>, // Shrink h3 to h5
                    h4: ({ children }) => <h6>{children}</h6>, // Shrink h4 to h6
                  }}
                >
                  {releaseInfo.current.commit.message}
                </ReactMarkdown>
              ) : (
                "No description available"
              )}
              </div>
            </div>
            <div className="row white-box__details-row">
              <div className="columns small-3">PUBLISHED AT</div>
              <div className="columns small-9">
              {releaseInfo.current?.published_at
                  ? releaseInfo.current.published_at
                  : releaseInfo.current?.commit?.author?.date
                  ? releaseInfo.current?.commit?.author.date
                  : "N/A"
              }
              </div>
            </div>
            <div className="row white-box__details-row">
              <div className="columns small-3">AUTHOR</div>
              <div className="columns small-9">{releaseInfo.current.author?.login || "Unknown, not working"}</div>
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
              {releaseInfo.latest?.html_url ? (
                <a
                  href={releaseInfo.latest.html_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  style={{ textDecoration: "none", color: "#007bff" }} // Optional: Add link styling
                >
                  {releaseInfo.latest?.tag_name
                    ? releaseInfo.latest.tag_name
                    : releaseInfo.latest?.sha
                    ? releaseInfo.latest.sha.slice(0, 7)
                    : "N/A"}
                </a>
              ) : (
                releaseInfo.latest?.tag_name
                  ? releaseInfo.latest.tag_name
                  : releaseInfo.latest?.sha
                  ? releaseInfo.latest.sha.slice(0, 7)
                  : "N/A"
              )}
              </div>
            </div>
            <div className="row white-box__details-row">
              <div className="columns small-3">DESCRIPTION</div>
              <div className="columns small-9">
              {releaseInfo.latest.body ? (
                <ReactMarkdown
                  components={{
                    h1: ({ children }) => <h3>{children}</h3>, // Shrink h1 to h3
                    h2: ({ children }) => <h4>{children}</h4>, // Shrink h2 to h4
                    h3: ({ children }) => <h5>{children}</h5>, // Shrink h3 to h5
                    h4: ({ children }) => <h6>{children}</h6>, // Shrink h4 to h6
                  }}
                >
                  {releaseInfo.latest.body}
                </ReactMarkdown>
              ) : releaseInfo.latest?.commit?.message ? (
                <ReactMarkdown
                  components={{
                    h1: ({ children }) => <h3>{children}</h3>, // Shrink h1 to h3
                    h2: ({ children }) => <h4>{children}</h4>, // Shrink h2 to h4
                    h3: ({ children }) => <h5>{children}</h5>, // Shrink h3 to h5
                    h4: ({ children }) => <h6>{children}</h6>, // Shrink h4 to h6
                  }}
                >
                  {releaseInfo.latest.commit.message}
                </ReactMarkdown>
              ) : (
                "No description available"
              )}
              </div>
            </div>
            <div className="row white-box__details-row">
              <div className="columns small-3">PUBLISHED AT</div>
              <div className="columns small-9">
              {releaseInfo.latest?.published_at
                  ? releaseInfo.latest.published_at
                  : releaseInfo.latest?.commit?.author?.date
                  ? releaseInfo.latest?.commit?.author.date
                  : "N/A"
              }
              </div>
            </div>
            <div className="row white-box__details-row">
              <div className="columns small-3">AUTHOR</div>
              <div className="columns small-9">{releaseInfo.latest.author?.login || "Unknown"}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
  export default ReleaseDetailsPanel