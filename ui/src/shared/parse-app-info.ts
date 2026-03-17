function parseImageRepositories(infoArray) {
  try {
    // Format: '"<alias>" Image Repository' (one entry per image)
    return infoArray
      .filter((item) => /^"[^"]+" Image Repository$/.test(item.name))
      .map((entry) => ({
        alias: entry.name.match(/^"([^"]+)" Image Repository$/)[1],
        imageRepository: entry.value?.split(":")[0] || null,
      }));
  } catch (err) {
    console.error("Error parsing info array for Image Repository:", err);
  }
  return [];
}


function parseAppRepository(infoArray) {
  try {
    // Find the object with the name "Application Repository"
    const appRepoEntry = infoArray.find((item) => item.name === "Application Repository");
    if (appRepoEntry && appRepoEntry.value) {
      // Return the value directly
      return appRepoEntry.value.trim() || null;
    }
  } catch (err) {
    console.error("Error parsing info array for Application Repository:", err);
  }
  return null; // Return null if parsing fails or value is not found
}


function findMatchingImage(images, repoName) {
  if (!repoName) return null;

  try {
    // Find the matching image
    const matchingImage = images.find((image) => image.startsWith(repoName));
    if (matchingImage) {
      // Extract the tag part between the colon and the "@" symbol
      const partsAfterColon = matchingImage.split(":");
      if (partsAfterColon.length > 1) {
        const tagPart = partsAfterColon[1].split("@")[0];
        return tagPart; // Return the part before the "@" symbol
      }
    }
  } catch (err) {
    console.error("Error processing images array:", err);
  }

  return null; // Return null if no match or parsing fails
}


export function getAppDetails(images, info) {
  const appRepository = parseAppRepository(info);

  const imageEntries = parseImageRepositories(info).map(({ alias, imageRepository }) => ({
    alias,
    imageTag: findMatchingImage(images, imageRepository),
  }));

  return {
    appRepository,
    imageEntries,
  };
}
