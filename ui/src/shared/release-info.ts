// Standardized ReleaseEntity fields

interface ReleaseEntity {
    ref?: string;          // Git SHA or tag
    url?: string;          // URL to the release or commit
    message?: string;      // Release description or commit message
    published_at?: string; // Publication date
    author?: string;       // Author login
}

export interface ReleaseInfo {
    current?: ReleaseEntity; // Current release or commit
    latest?: ReleaseEntity;  // Latest release or commit
}