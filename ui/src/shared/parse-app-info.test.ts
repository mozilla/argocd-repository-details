import { describe, it, expect } from "vitest";
import { getAppDetails } from "./parse-app-info";

const makeInfo = (entries: { name: string; value: string }[]) => entries;

const makeImages = (images: string[]) => images;

describe("getAppDetails", () => {
  const appRepo = "mozilla-services/some-app";
  const appRepoEntry = { name: "Application Repository", value: appRepo };

  describe("with multiple images", () => {
    const info = makeInfo([
      appRepoEntry,
      { name: '"mars" Image Repository', value: "us-docker.pkg.dev/moz-fx-ads-prod/ads-prod/mars" },
      { name: '"shepherd" Image Repository', value: "us-docker.pkg.dev/moz-fx-ads-prod/ads-prod/shepherd" },
    ]);

    it("returns one entry per image alias", () => {
      const images = makeImages([
        "us-docker.pkg.dev/moz-fx-ads-prod/ads-prod/mars:v1.2.3",
        "us-docker.pkg.dev/moz-fx-ads-prod/ads-prod/shepherd:v4.5.6",
      ]);
      const { appRepository, imageEntries } = getAppDetails(images, info);

      expect(appRepository).toBe(appRepo);
      expect(imageEntries).toHaveLength(2);
      expect(imageEntries[0]).toEqual({ alias: "mars", imageTag: "v1.2.3" });
      expect(imageEntries[1]).toEqual({ alias: "shepherd", imageTag: "v4.5.6" });
    });

    it("strips digest suffix from image tag", () => {
      const images = makeImages([
        "us-docker.pkg.dev/moz-fx-ads-prod/ads-prod/mars:v1.2.3@sha256:abc123",
        "us-docker.pkg.dev/moz-fx-ads-prod/ads-prod/shepherd:v4.5.6@sha256:def456",
      ]);
      const { imageEntries } = getAppDetails(images, info);

      expect(imageEntries[0].imageTag).toBe("v1.2.3");
      expect(imageEntries[1].imageTag).toBe("v4.5.6");
    });

    it("returns null imageTag when image is not in the status summary", () => {
      const images = makeImages([
        "us-docker.pkg.dev/moz-fx-ads-prod/ads-prod/mars:v1.2.3",
        // shepherd missing
      ]);
      const { imageEntries } = getAppDetails(images, info);

      expect(imageEntries[0].imageTag).toBe("v1.2.3");
      expect(imageEntries[1].imageTag).toBeNull();
    });
  });

  describe("with no image entries", () => {
    it("returns an empty imageEntries array", () => {
      const info = makeInfo([appRepoEntry]);
      const { appRepository, imageEntries } = getAppDetails([], info);

      expect(appRepository).toBe(appRepo);
      expect(imageEntries).toHaveLength(0);
    });
  });

  describe("with missing Application Repository", () => {
    it("returns null appRepository", () => {
      const info = makeInfo([
        { name: '"mars" Image Repository', value: "us-docker.pkg.dev/moz-fx-ads-prod/ads-prod/mars" },
      ]);
      const { appRepository } = getAppDetails([], info);

      expect(appRepository).toBeNull();
    });
  });
});
