export function getHeaders({
    applicationName,
    applicationNamespace,
    project
  }: {
    applicationName: string;
    applicationNamespace: string;
    project: string;
  }) {
    const argocdApplicationName = `${applicationNamespace}:${applicationName}`;
    return {
      'cache-control': 'no-cache',
      'Content-Type': 'application/json',
      "Argocd-Application-Name": `${argocdApplicationName}`,
      "Argocd-Project-Name": `${project}`,
    };
  }