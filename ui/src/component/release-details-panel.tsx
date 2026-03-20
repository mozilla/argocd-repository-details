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
  const [entries, setEntries] = useState<Array<{ alias: string | null; releaseInfo: ReleaseInfo | null }>>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const applicationNamespace = application?.metadata?.namespace || "";
  const applicationName = application?.metadata?.name || "";
  const project = application?.spec?.project || "";
  const images = application?.status?.summary?.images || "";
  const info = application?.spec?.info || "";

  useEffect(() => {
    if (!application) return;

    const { appRepository, imageEntries } = getAppDetails(images, info);

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

    fetchReleaseInfo();
  }, [application]);

  if (loading) {
    return <div>Loading release information...</div>;
  }

  if (error) {
    return <div>Error loading release information: {error}</div>;
  }

  if (entries.length === 0) {
    return <div>No release information available for this application.</div>;
  }

  const isMulti = entries.length > 1;

  const formatRef = (ref) => {
    if (!ref) return "N/A";
    return /^[0-9a-f]{40}$/i.test(ref) ? ref.slice(0, 7) : ref;
  };

  return (
    <div style={{ marginTop: "15px" }}>
      {entries.map(({ alias, releaseInfo }, i) => (
        <div key={i}>
          {isMulti && alias && <h4 style={{ marginBottom: "8px" }}>{alias}</h4>}
          <div className="row">
            {/* Deployed Release Details */}
            <div className="columns small-6">
              <div className="white-box">
                <div className="white-box__details">
                  <p>DEPLOYED RELEASE</p>
                  <div className="row white-box__details-row">
                    <div className="columns small-3">REF</div>
                    <div className="columns small-9">
                      {releaseInfo?.current?.url ? (
                        <a
                          href={releaseInfo.current.url}
                          target="_blank"
                          rel="noopener noreferrer"
                          style={{ textDecoration: "none", color: "#007bff" }}
                        >
                          {formatRef(releaseInfo.current?.ref)}
                        </a>
                      ) : (
                        formatRef(releaseInfo?.current?.ref)
                      )}
                    </div>
                  </div>
                  <div className="row white-box__details-row">
                    <div className="columns small-3">DESCRIPTION</div>
                    <div className="columns small-9">
                      {releaseInfo?.current?.message ? (
                        <ReactMarkdown
                          components={{
                            h1: ({ children }) => <h3>{children}</h3>,
                            h2: ({ children }) => <h4>{children}</h4>,
                            h3: ({ children }) => <h5>{children}</h5>,
                            h4: ({ children }) => <h6>{children}</h6>,
                          }}
                        >
                          {releaseInfo.current.message}
                        </ReactMarkdown>
                      ) : (
                        "No description available"
                      )}
                    </div>
                  </div>
                  <div className="row white-box__details-row">
                    <div className="columns small-3">PUBLISHED AT</div>
                    <div className="columns small-9">
                      {releaseInfo?.current?.published_at || "N/A"}
                    </div>
                  </div>
                  <div className="row white-box__details-row">
                    <div className="columns small-3">AUTHOR</div>
                    <div className="columns small-9">{releaseInfo?.current?.author || "Unknown"}</div>
                  </div>
                </div>
              </div>
            </div>

            {/* Application Release Details */}
            <div className="columns small-6">
              <div className="white-box">
                <div className="white-box__details">
                  <p>LATEST APPLICATION RELEASE</p>
                  <div className="row white-box__details-row">
                    <div className="columns small-3">REF</div>
                    <div className="columns small-9">
                      {releaseInfo?.latest?.url ? (
                        <a
                          href={releaseInfo.latest.url}
                          target="_blank"
                          rel="noopener noreferrer"
                          style={{ textDecoration: "none", color: "#007bff" }}
                        >
                          {formatRef(releaseInfo.latest?.ref)}
                        </a>
                      ) : (
                        formatRef(releaseInfo?.latest?.ref)
                      )}
                    </div>
                  </div>
                  <div className="row white-box__details-row">
                    <div className="columns small-3">DESCRIPTION</div>
                    <div className="columns small-9">
                      {releaseInfo?.latest?.message ? (
                        <ReactMarkdown
                          components={{
                            h1: ({ children }) => <h3>{children}</h3>,
                            h2: ({ children }) => <h4>{children}</h4>,
                            h3: ({ children }) => <h5>{children}</h5>,
                            h4: ({ children }) => <h6>{children}</h6>,
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
                      {releaseInfo?.latest?.published_at || "N/A"}
                    </div>
                  </div>
                  <div className="row white-box__details-row">
                    <div className="columns small-3">AUTHOR</div>
                    <div className="columns small-9">{releaseInfo?.latest?.author || "Unknown"}</div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
};
  export default ReleaseDetailsPanel
